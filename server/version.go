package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/consts"
	"github.com/patrickmn/go-cache"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func getVersion(client *http.Client, githubEndpoint, repo string) gin.HandlerFunc {
	const cacheDuration = 4 * time.Hour
	versionCache := cache.New(cacheDuration, cacheDuration*2)
	return func(c *gin.Context) {
		var latestVersion string
		if version, exists := versionCache.Get(""); exists {
			latestVersion = version.(string)
		}
		if latestVersion == "" {
			var err error
			latestVersion, err = fetchUpstreamVersion(c, client, githubEndpoint, repo)
			if err != nil {
				abortWithClientError(c, http.StatusInternalServerError, err)
				return
			}
			versionCache.SetDefault("", latestVersion)
		}

		c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(cacheDuration.Seconds())))
		c.JSON(http.StatusOK, map[string]interface{}{
			"Version":         consts.Version,
			"UpdateAvailable": latestVersion != consts.Version,
		})
	}
}

func fetchUpstreamVersion(ctx context.Context, client *http.Client, githubEndpoint, repo string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://"+path.Join(githubEndpoint, "repos", repo, "releases/latest"), nil)
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err := resp.Body.Close(); err != nil {
		return "", err
	}

	var latest githubRelease
	err = json.Unmarshal(buf, &latest)
	return latest.TagName, err
}
