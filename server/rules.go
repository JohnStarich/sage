package server

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/rules"
	"github.com/johnstarich/sage/sync"
	"github.com/johnstarich/sage/vcs"
	"github.com/pkg/errors"
)

// CSVRule is the request model for changing a single rule
type CSVRule struct {
	Conditions []string
	Account2   string
}

func getRules(rulesStore *rules.Store, ldgStore *ledger.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var options struct {
			Transaction string `form:"transaction"`
		}
		if err := c.BindQuery(&options); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		var result interface{} = rulesStore
		if options.Transaction != "" {
			txn, found := ldgStore.Transaction(options.Transaction)
			if !found {
				abortWithClientError(c, http.StatusNotFound, errors.New("Transaction not found"))
				return
			}
			result = rulesStore.Matches(&txn)
		}
		c.JSON(http.StatusOK, map[string]interface{}{
			"Rules": result,
		})
	}
}

func getRule(rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var options struct {
			ID int
		}
		if err := c.BindQuery(&options); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		rule, err := rulesStore.Get(options.ID)
		if err != nil {
			abortWithClientError(c, http.StatusNotFound, err)
			return
		}
		c.JSON(http.StatusOK, map[string]interface{}{
			"Rule": rule,
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

func updateRule(rulesFile vcs.File, rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var bodyRule struct {
			CSVRule
			Index *int
		}
		if err := c.BindJSON(&bodyRule); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		if bodyRule.Index == nil {
			abortWithClientError(c, http.StatusBadRequest, errors.New("Rule index is required"))
			return
		}
		rule, err := rules.NewCSVRule("", bodyRule.Account2, "", bodyRule.Conditions...)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		err = rulesStore.Update(*bodyRule.Index, rule)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		if err := sync.Rules(rulesFile, rulesStore); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func addRule(rulesFile vcs.File, rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var bodyRule CSVRule
		if err := c.BindJSON(&bodyRule); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		rule, err := rules.NewCSVRule("", bodyRule.Account2, "", bodyRule.Conditions...)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		newIndex := rulesStore.Add(rule)
		if err := sync.Rules(rulesFile, rulesStore); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, map[string]interface{}{
			"Index": newIndex,
		})
	}
}

func deleteRule(rulesFile vcs.File, rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var bodyRule struct {
			Index *int
		}
		if err := c.BindJSON(&bodyRule); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		if bodyRule.Index == nil {
			abortWithClientError(c, http.StatusBadRequest, errors.New("Index is required"))
			return
		}
		if err := rulesStore.Remove(*bodyRule.Index); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		if err := sync.Rules(rulesFile, rulesStore); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
