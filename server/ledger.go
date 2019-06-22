package server

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/ledger"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	accountTypesQuery = "accountTypes[]" // include [] suffix to support query param arrays
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
	var page, results int = 1, 10
	if pageQuery, ok := c.GetQuery("page"); ok {
		if parsedPage, err := strconv.ParseInt(pageQuery, 10, 64); err != nil {
			c.Error(errors.Errorf("Invalid integer: %s", pageQuery))
		} else if page < 1 {
			c.Error(errors.New("Page must be a positive integer"))
		} else {
			page = int(parsedPage)
		}
	}
	if resultsQuery, ok := c.GetQuery("results"); ok {
		if parsedResults, err := strconv.ParseInt(resultsQuery, 10, 64); err != nil {
			c.Error(errors.Errorf("Invalid integer: %s", resultsQuery))
		} else if results < 1 {
			c.Error(errors.New("Results must be a positive integer"))
		} else {
			results = int(parsedResults)
		}
	}
	if len(c.Errors) > 0 {
		errMsg := ""
		for _, e := range c.Errors {
			errMsg += e.Error() + "\n"
		}
		c.AbortWithError(http.StatusBadRequest, errors.New(errMsg))
		return
	}
	ledger.WriteJSON(page, results, c.Writer)
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
	Institution string `json:",omitempty"`
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

	accountTypes := map[string]bool{
		"assets":      true,
		"liabilities": true,
	}
	if accountTypesQueryArray := c.QueryArray(accountTypesQuery); len(accountTypesQueryArray) > 0 {
		accountTypes = make(map[string]bool, len(accountTypesQueryArray))
		for _, value := range accountTypesQueryArray {
			accountTypes[value] = true
		}
	}

	for accountName, balances := range balanceMap {
		account := AccountResponse{
			ID:       accountName,
			Balances: balances,
		}
		components := strings.Split(accountName, ":")
		if len(components) == 0 {
			continue
		}
		accountType := components[0]
		if len(accountTypes) > 0 && !accountTypes[accountType] {
			// filter by account type
			continue
		}

		account.AccountType = accountType
		switch accountType {
		case "assets", "liabilities":
			if len(components) < 3 {
				// require accountType:institution:accountNumber format
				continue
			}
			institutionName, accountID := components[1], strings.Join(components[2:], ":")

			account.Account = accountID
			account.Institution = institutionName

			idSuffix := accountID
			if len(idSuffix) > client.RedactSuffixLength {
				idSuffix = idSuffix[len(idSuffix)-client.RedactSuffixLength:]
			}
			if clientAccount, ok := accountIDMap[institutionName][idSuffix]; ok {
				account.Account = clientAccount.Description()
			}
		default:
			account.ID = strings.Join(components[1:], ":")
			account.Account = account.ID
		}

		resp.Accounts = append(resp.Accounts, account)
	}
	sort.Slice(resp.Accounts, func(a, b int) bool {
		return resp.Accounts[a].ID < resp.Accounts[b].ID
	})
	c.JSON(http.StatusOK, resp)
}
