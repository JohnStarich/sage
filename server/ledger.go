package server

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/ledger"
	"github.com/shopspring/decimal"
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

type BalanceResponse struct {
	Start, End time.Time
	Accounts   []AccountResponse
}

type AccountResponse struct {
	ID          string
	Account     string
	AccountType string
	Balances    []decimal.Decimal
	Institution string
}

func getBalances(c *gin.Context) {
	ledger := c.MustGet(ledgerKey).(*ledger.Ledger)
	accounts := c.MustGet(accountsKey).([]client.Account)
	start, end, balanceMap := ledger.Balances()
	resp := BalanceResponse{
		Start: start,
		End:   end,
	}
	// inst name -> account ID suffix -> account
	accountIDMap := make(map[string]map[string]client.Account)
	for _, clientAccount := range accounts {
		instName := clientAccount.Institution().Description()
		id := clientAccount.ID()
		if len(id) > client.RedactSuffixLength {
			id = id[len(id)-client.RedactSuffixLength:]
		}
		if accountIDMap[instName] == nil {
			accountIDMap[instName] = make(map[string]client.Account)
		}
		accountIDMap[instName][id] = clientAccount
	}

	for accountName, balances := range balanceMap {
		account := AccountResponse{
			ID:       accountName,
			Balances: balances,
		}
		components := strings.Split(accountName, ":")
		if len(components) < 3 {
			continue
		}
		accountType, institutionName, accountID := components[0], components[1], strings.Join(components[2:], ":")
		if accountType != "assets" && accountType != "liabilities" {
			continue
		}

		account.AccountType = accountType
		account.Account = accountID
		account.Institution = institutionName

		idSuffix := accountID
		if len(idSuffix) > client.RedactSuffixLength {
			idSuffix = idSuffix[len(idSuffix)-client.RedactSuffixLength:]
		}
		if clientAccount, ok := accountIDMap[institutionName][idSuffix]; ok {
			account.Account = clientAccount.Description()
		}
		resp.Accounts = append(resp.Accounts, account)
	}
	sort.Slice(resp.Accounts, func(a, b int) bool {
		return resp.Accounts[a].ID < resp.Accounts[b].ID
	})
	c.JSON(http.StatusOK, resp)
}
