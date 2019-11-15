package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/rules"
	"github.com/johnstarich/sage/sync"
	"github.com/johnstarich/sage/vcs"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	accountTypesQuery = "accountTypes[]" // include [] suffix to support query param arrays
	// MaxResults is the maximum number of results from a paginated request
	MaxResults = 50
)

func syncLedger(ledgerFile vcs.File, ldg *ledger.Ledger, accountStore *client.AccountStore, rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := c.MustGet(loggerKey).(*zap.Logger)
		_, syncFromStart := c.GetQuery("fromLedgerStart")
		err := sync.Sync(logger, ledgerFile, ldg, accountStore, rulesStore, syncFromStart)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusOK)
	}
}

type transactionsResponse struct {
	ledger.QueryResult
	AccountIDMap map[string]string
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
		result := transactionsResponse{
			QueryResult:  ldg.Query(c.Query("search"), page, results),
			AccountIDMap: make(map[string]string),
		}
		// attempt to make asset and liability accounts more descriptive
		accountIDMap, err := newAccountIDMap(accountStore)
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		for i := range result.Transactions {
			accountName := result.Transactions[i].Postings[0].Account
			if _, exists := result.AccountIDMap[accountName]; !exists {
				clientAccount, ok := accountIDMap.Find(accountName)
				if ok {
					result.AccountIDMap[accountName] = clientAccount.Description()
				}
			}
		}
		c.JSON(http.StatusOK, result)
	}
}

// BalanceResponse is the response type for fetching account balances
type BalanceResponse struct {
	Start, End         *time.Time
	OpeningBalanceDate *time.Time
	Messages           []AccountMessage
	Accounts           []AccountResponse
}

// AccountResponse contains details for an account's balance over time
type AccountResponse struct {
	ID             string
	Account        string
	AccountType    string
	OpeningBalance *decimal.Decimal
	Balances       []decimal.Decimal
	Institution    string `json:",omitempty"`
}

// AccountMessage contains important information for an account
type AccountMessage struct {
	AccountID   string `json:",omitempty"`
	AccountName string `json:",omitempty"`
	Message     string
}

type txnToAccountMap map[string]map[string]model.Account

// newAccountIDMap returns a mapping from an institution's description, then account ID suffix (without '*'s), and finally to the source account
func newAccountIDMap(accountStore *client.AccountStore) (txnToAccountMap, error) {
	// inst name -> account ID suffix -> account
	accountIDMap := make(txnToAccountMap)
	var clientAccount model.Account
	err := accountStore.Iter(&clientAccount, func(id string) bool {
		instName := clientAccount.Institution().Org()
		if len(id) > model.RedactSuffixLength {
			id = id[len(id)-model.RedactSuffixLength:]
		}
		if accountIDMap[instName] == nil {
			accountIDMap[instName] = make(map[string]model.Account)
		}
		accountIDMap[instName][id] = clientAccount
		return true
	})
	return accountIDMap, err
}

func (t txnToAccountMap) Find(accountName string) (account model.Account, found bool) {
	components := strings.Split(accountName, ":")
	if len(components) == 0 {
		return nil, false
	}
	accountType := components[0]
	if accountType != model.AssetAccount && accountType != model.LiabilityAccount {
		return nil, false
	}
	if len(components) < 3 {
		// require accountType:institution:accountNumber format
		return nil, false
	}
	institutionName, accountID := components[1], strings.Join(components[2:], ":")

	idSuffix := accountID
	if len(idSuffix) > model.RedactSuffixLength {
		idSuffix = idSuffix[len(idSuffix)-model.RedactSuffixLength:]
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
		accountIDMap, err := newAccountIDMap(accountStore)
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		accountTypes := map[string]bool{
			// return assets and liabilities by default
			// useful for a simple balance table
			model.AssetAccount:     true,
			model.LiabilityAccount: true,
		}
		if accountTypesQueryArray := c.QueryArray(accountTypesQuery); len(accountTypesQueryArray) > 0 {
			accountTypes = make(map[string]bool, len(accountTypesQueryArray))
			for _, value := range accountTypesQueryArray {
				accountTypes[value] = true
			}
		}

		var openingBalances ledger.Transaction
		if balances, found := ldg.OpeningBalances(); found {
			resp.OpeningBalanceDate = &balances.Date
			openingBalances = balances
		}
		findOpeningBalance := func(accountName string) *decimal.Decimal {
			for _, p := range openingBalances.Postings {
				if p.Account == accountName {
					return &p.Amount
				}
			}
			return nil
		}

		for accountName, balances := range balanceMap {
			account := AccountResponse{
				ID:             accountName,
				OpeningBalance: findOpeningBalance(accountName),
				Balances:       balances,
			}

			format, err := model.ParseLedgerFormat(accountName)
			if err != nil || format.AccountType == "" {
				continue
			}
			if len(accountTypes) > 0 && !accountTypes[format.AccountType] {
				// filter by account type
				continue
			}

			account.AccountType = format.AccountType
			switch format.AccountType {
			case model.AssetAccount, model.LiabilityAccount:
				account.Account = format.Institution + " " + format.AccountID
				account.Institution = format.Institution
				if clientAccount, found := accountIDMap.Find(accountName); found {
					account.Account = clientAccount.Description()
				}
			default:
				account.ID = format.Remaining
				account.Account = account.ID
			}

			resp.Accounts = append(resp.Accounts, account)
		}

		var accounts []model.Account
		var a model.Account
		err = accountStore.Iter(&a, func(id string) bool {
			format := model.LedgerFormat(a)
			if len(accountTypes) == 0 || accountTypes[format.AccountType] {
				accounts = append(accounts, a)
			}
			return true
		})
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		for _, account := range accounts {
			ledgerAccount := model.LedgerFormat(account)
			accountName := ledgerAccount.String()
			if _, inBalances := balanceMap[accountName]; !inBalances {
				resp.Accounts = append(resp.Accounts, AccountResponse{
					ID:             accountName,
					Account:        account.Description(),
					AccountType:    ledgerAccount.AccountType,
					OpeningBalance: findOpeningBalance(accountName),
				})
			}
		}
		sort.Slice(resp.Accounts, func(a, b int) bool {
			return resp.Accounts[a].ID < resp.Accounts[b].ID
		})

		resp.Messages = append(resp.Messages, getOpeningBalanceMessages(ldg, accounts)...)
		sort.Slice(resp.Messages, func(a, b int) bool {
			return resp.Messages[a].AccountID < resp.Messages[b].AccountID
		})

		c.JSON(http.StatusOK, resp)
	}
}

func getOpeningBalanceMessages(ldg *ledger.Ledger, accounts []model.Account) []AccountMessage {
	var messages []AccountMessage
	var openingPostings []ledger.Posting
	if openingBalances, ok := ldg.OpeningBalances(); ok {
		openingPostings = openingBalances.Postings
	}
	openingBalAccounts := make(map[string]bool)
	for _, p := range openingPostings {
		if !p.IsOpeningBalance() {
			openingBalAccounts[p.Account] = true
		}
	}
	for _, account := range accounts {
		id := model.LedgerAccountName(account)
		if !openingBalAccounts[id] {
			messages = append(messages, AccountMessage{
				AccountID:   id,
				AccountName: account.Description(),
				Message:     "Missing opening balance",
			})
		}
	}
	return messages
}

func getExpenseAndRevenueAccounts(ldg *ledger.Ledger, rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, _, balanceMap := ldg.Balances()
		accounts := make(map[string]bool, len(balanceMap)+1)
		accounts[model.Uncategorized] = true
		for account := range balanceMap {
			if strings.HasPrefix(account, model.ExpenseAccount+":") || strings.HasPrefix(account, model.RevenueAccount+":") {
				components := strings.Split(account, ":")
				account = ""
				for _, comp := range components {
					if len(account) > 0 {
						account += ":"
					}
					account += comp
					accounts[account] = true
				}
			}
		}

		for _, account := range rulesStore.Accounts() {
			accounts[account] = true
		}

		accountsSlice := make([]string, 0, len(accounts))
		for account := range accounts {
			accountsSlice = append(accountsSlice, account)
		}

		sort.Strings(accountsSlice)
		c.JSON(http.StatusOK, map[string]interface{}{
			"Accounts": accountsSlice,
		})
	}
}

func updateTransaction(ledgerFile vcs.File, ldg *ledger.Ledger) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		var txnJSON struct {
			ID string
		}
		if err := json.Unmarshal(body, &txnJSON); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		id := txnJSON.ID

		var txn ledger.Transaction
		if err := json.Unmarshal(body, &txn); err != nil {
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

		if err := sync.LedgerFile(ldg, ledgerFile); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func updateOpeningBalance(ledgerFile vcs.File, ldg *ledger.Ledger, accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var opening ledger.Transaction
		if err := c.ShouldBindJSON(&opening); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		var total decimal.Decimal
		for i := range opening.Postings {
			format, err := model.ParseLedgerFormat(opening.Postings[i].Account)
			if err != nil || format.AccountType == "" {
				abortWithClientError(c, http.StatusBadRequest, errors.Wrap(err, "Invalid ledger account ID"))
				return
			}
			total = total.Sub(opening.Postings[i].Amount)
			opening.Postings[i].Currency = "$"
		}

		opening.Postings = append(opening.Postings, ledger.Posting{
			Account:  "equity:Opening Balances",
			Amount:   total,
			Currency: "$",
			Tags:     map[string]string{"id": ledger.OpeningBalanceID},
		})

		if err := ldg.UpdateOpeningBalance(opening); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		if err := sync.LedgerFile(ldg, ledgerFile); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func importOFXFile(ledgerFile vcs.File, ldg *ledger.Ledger, accountStore *client.AccountStore, rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := c.MustGet(loggerKey).(*zap.Logger)
		skeletonAccounts, txns, err := client.ReadOFX(c.Request.Body)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		rulesStore.ApplyAll(txns)
		if err := ldg.AddTransactions(txns); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		if err := sync.LedgerFile(ldg, ledgerFile); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		accountsAdded := 0
		for _, account := range skeletonAccounts {
			if err := accountStore.Add(account); err != nil {
				logger.Warn("Failed to add bare-bones account from imported file", zap.String("error", err.Error()))
			} else {
				accountsAdded++
			}
		}
		c.Status(http.StatusNoContent)
	}
}

type renameParams struct {
	Old   string `binding:"required"`
	New   string `binding:"required"`
	OldID string
	NewID string
}

func renameLedgerAccount(ledgerFile vcs.File, ldg *ledger.Ledger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params renameParams
		if err := c.BindJSON(&params); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		if params.OldID != params.NewID && (params.OldID == "" || params.NewID == "") {
			abortWithClientError(c, http.StatusBadRequest, errors.New("If OldID or NewID is set, the other must also be set"))
			return
		}

		renameCount := ldg.RenameAccount(params.Old, params.New, params.OldID, params.NewID)
		if err := sync.LedgerFile(ldg, ledgerFile); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, map[string]interface{}{
			"Renamed": renameCount,
		})
	}
}

func renameSuggestions(ldg *ledger.Ledger, accountStore *client.AccountStore) gin.HandlerFunc {
	const DiscoverOldOrg = "Discover Financial Services"
	return func(c *gin.Context) {
		var suggestions []renameParams
		var account model.Account
		err := accountStore.Iter(&account, func(string) bool {
			// if old Discover direct connect account, show rename to use new Org and FID
			if account.Institution().Org() == DiscoverOldOrg && account.Institution().FID() == "7101" {
				ledgerAccount := model.LedgerFormat(account).String()
				accountID := account.ID()
				suggestions = append(suggestions, renameParams{
					Old:   ledgerAccount,
					New:   strings.Replace(ledgerAccount, DiscoverOldOrg, "Discover Card Account Center", 1),
					OldID: client.MakeUniqueTxnID("7101", accountID)(""),
					NewID: client.MakeUniqueTxnID("9625", accountID)(""),
				})
			}
			return true
		})
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, map[string]interface{}{
			"Suggestions": suggestions,
		})
	}
}
