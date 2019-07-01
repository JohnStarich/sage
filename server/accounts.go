package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
)

func getAccount(c *gin.Context) {
	accounts := c.MustGet(accountsKey).([]client.Account)
	accountID := c.Param("id")
	for _, account := range accounts {
		if account.ID() == accountID {
			c.JSON(http.StatusOK, map[string]interface{}{
				"Account": account,
			})
			return
		}
	}
	c.AbortWithStatus(http.StatusNotFound)
}

func getAccounts(c *gin.Context) {
	accounts := c.MustGet(accountsKey).([]client.Account)
	c.JSON(http.StatusOK, map[string]interface{}{
		"Accounts": accounts,
	})
}
