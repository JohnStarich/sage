package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
)

func getAccount(accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Param("id")
		account, exists := accountStore.Find(accountID)
		if !exists {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusOK, map[string]interface{}{
			"Account": account,
		})
	}
}

func getAccounts(accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"Accounts": accountStore,
		})
	}
}
