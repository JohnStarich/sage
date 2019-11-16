package server

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/rules"
	"github.com/johnstarich/sage/sync"
	"github.com/johnstarich/sage/vcs"
	"github.com/pkg/errors"
)

func getRules(rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"Rules": rulesStore,
		})
	}
}

func updateRules(rulesFile vcs.File, rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		decoder := json.NewDecoder(c.Request.Body)
		var newRules rules.Rules
		if err := decoder.Decode(&newRules); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, map[string]string{
				"Error": errors.Wrap(err, "Malformed rules").Error(),
			})
			return
		}
		rulesStore.Replace(newRules)
		if err := sync.Rules(rulesFile, rulesStore); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
