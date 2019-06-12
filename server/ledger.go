package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/ledger"
)

func syncLedger(c *gin.Context) {
	runSync := c.MustGet(syncKey).(func() error)
	err := runSync()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusOK)
}

func getTransactions(c *gin.Context) {
	ledger := c.MustGet(ledgerKey).(*ledger.Ledger)
	c.Status(http.StatusOK)
	ledger.WriteJSON(c.Writer)
}

func getBalances(c *gin.Context) {
	ledger := c.MustGet(ledgerKey).(*ledger.Ledger)
	balances := ledger.Balances()
	c.JSON(http.StatusOK, balances)
}
