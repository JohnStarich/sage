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
	"github.com/johnstarich/sage/rules"
	"github.com/johnstarich/sage/sync"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	accountTypesQuery = "accountTypes[]" // include [] suffix to support query param arrays
	// MaxResults is the maximum number of results from a paginated request
	MaxResults = 50
)

func syncLedger(ledgerFileName string, ldg *ledger.Ledger, accountStore *client.AccountStore, r rules.Rules) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := c.MustGet(loggerKey).(*zap.Logger)
		err := sync.Sync(logger, ledgerFileName, ldg, accountStore, r)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusOK)
	}
}

func getTransactions(ldg *ledger.Ledger, accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var page, results int = 1, 10
		if pageQuery, ok := c.GetQuery("page"); ok {
			if parsedPage, err := strconv.ParseInt(pageQuery, 10, 64); err != nil {
				c.Error(errors.Errorf("Invalid integer: %s", pageQuery))
			} else if parsedPage < 1 {
				c.Error(errors.New("Page must be a positive integer"))
			} else {
				page = int(parsedPage)
			}
		}
		if resultsQuery, ok := c.GetQuery("results"); ok {
			if parsedResults, err := strconv.ParseInt(resultsQuery, 10, 64); err != nil {
				c.Error(errors.Errorf("Invalid integer: %s", resultsQuery))
			} else if parsedResults < 1 || parsedResults > MaxResults {
				c.Error(errors.Errorf("Results must be a positive integer no more than %d", MaxResults))
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
		query := ldg.Query(c.Query("search"), page, results)
		// attempt to make asset and liability accounts more descriptive
		accountIDMap := newAccountIDMap(accountStore)
		for i := range query.Transactions {
			if clientAccount, ok := accountIDMap.Find(query.Transactions[i].Postings[0].Account); ok {
				query.Transactions[i].Postings[0].Account = clientAccount.Description()
			}
		}
		c.JSON(http.StatusOK, query)
	}
}

// BalanceResponse is the response type for fetching account balances
type BalanceResponse struct {
	Start, End time.Time
	Accounts   []AccountResponse
}

// AccountResponse contains details for an account's balance over time
type AccountResponse struct {
	ID          string
	Account     string
	AccountType string
	Balances    []decimal.Decimal
	Institution string `json:",omitempty"`
}

type txnToAccountMap map[string]map[string]client.Account

// newAccountIDMap returns a mapping from an institution's description, then account ID suffix (without '*'s), and finally to the source account
func newAccountIDMap(accountStore *client.AccountStore) txnToAccountMap {
	// inst name -> account ID suffix -> account
	accountIDMap := make(txnToAccountMap)
	accountStore.Iterate(func(clientAccount client.Account) bool {
		instName := clientAccount.Institution().Description()
		id := clientAccount.ID()
		if len(id) > client.RedactSuffixLength {
			id = id[len(id)-client.RedactSuffixLength:]
		}
		if accountIDMap[instName] == nil {
			accountIDMap[instName] = make(map[string]client.Account)
		}
		accountIDMap[instName][id] = clientAccount
		return true
	})
	return accountIDMap
}

func (t txnToAccountMap) Find(accountName string) (account client.Account, found bool) {
	components := strings.Split(accountName, ":")
	if len(components) == 0 {
		return nil, false
	}
	accountType := components[0]
	if accountType != client.AssetAccount && accountType != client.LiabilityAccount {
		return nil, false
	}
	if len(components) < 3 {
		// require accountType:institution:accountNumber format
		return nil, false
	}
	institutionName, accountID := components[1], strings.Join(components[2:], ":")

	idSuffix := accountID
	if len(idSuffix) > client.RedactSuffixLength {
		idSuffix = idSuffix[len(idSuffix)-client.RedactSuffixLength:]
	}
	clientAccount, found := t[institutionName][idSuffix]
	return clientAccount, found
}

func getBalances(ldg *ledger.Ledger, accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		start, end, balanceMap := ldg.Balances()
		resp := BalanceResponse{
			Start: start,
			End:   end,
		}
		accountIDMap := newAccountIDMap(accountStore)

		accountTypes := map[string]bool{
			// return assets and liabilities by default
			// useful for a simple balance table
			client.AssetAccount:     true,
			client.LiabilityAccount: true,
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
			case client.AssetAccount, client.LiabilityAccount:
				if len(components) < 3 {
					// require accountType:institution:accountNumber format
					continue
				}
				institutionName, accountID := components[1], strings.Join(components[2:], ":")

				account.Account = accountID
				account.Institution = institutionName

				if clientAccount, found := accountIDMap.Find(accountName); found {
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
}

func getExpenseAndRevenueAccounts(ldg *ledger.Ledger) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, _, balanceMap := ldg.Balances()
		accounts := make([]string, 0, len(balanceMap)+1)
		accounts = append(accounts, client.Uncategorized)
		for account := range balanceMap {
			if strings.HasPrefix(account, client.ExpenseAccount+":") || strings.HasPrefix(account, client.RevenueAccount+":") {
				accounts = append(accounts, account)
			}
		}
		sort.Strings(accounts)
		c.JSON(http.StatusOK, map[string]interface{}{
			"Accounts": accounts,
		})
	}
}

func updateTransaction(ledgerFileName string, ldg *ledger.Ledger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var txn ledger.Transaction
		if err := c.BindJSON(&txn); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		switch err := ldg.UpdateTransaction(id, txn).(type) {
		case ledger.Error:
			c.AbortWithError(http.StatusBadRequest, err)
			return
		case nil: // skip
		default:
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if err := sync.LedgerFile(ldg, ledgerFileName); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}
