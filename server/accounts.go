package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
)

func getAccounts(c *gin.Context) {
	accounts := c.MustGet(accountsKey).([]client.Account)
	c.JSON(http.StatusOK, map[string]interface{}{
		"Accounts": accounts,
	})
}
