package web

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

// DownloadRequest contains a download request's details to help reconstruct your own download HTTP request
type DownloadRequest struct {
	Cookies   []*http.Cookie
	URL       string
	UserAgent string
}

// HTTPRequest creates an *http.Request from this download's details
func (d *DownloadRequest) HTTPRequest(ctx context.Context) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, d.URL, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", d.UserAgent)
	for _, cookie := range d.Cookies {
		req.AddCookie(cookie)
	}
	return req, nil
}

// Fetch downloads the file with the given download request details and returns the response body as bytes
func (d *DownloadRequest) Fetch(ctx context.Context) ([]byte, error) {
	return d.fetch(ctx, http.DefaultClient)
}

func (d *DownloadRequest) fetch(ctx context.Context, client *http.Client) ([]byte, error) {
	req, err := d.HTTPRequest(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create download request")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to request download")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch download")
	}
	if resp.StatusCode/100 != 2 {
		return nil, errors.Errorf("Bad response code [%d]: %s\n%s", resp.StatusCode, d.URL, string(body))
	}
	return body, nil
}
