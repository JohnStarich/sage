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
	sErrors "github.com/johnstarich/sage/errors"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/prompter"
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

func getLedgerSyncStatus(ldgStore *ledger.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var errs sErrors.Errors // used for its marshaler
		syncing, prompt, err := ldgStore.SyncStatus()
		errs.AddErr(err)
		c.JSON(http.StatusOK, map[string]interface{}{
			"Syncing": syncing,
			"Prompt":  prompt,
			"Errors":  errs.ErrOrNil(),
		})
	}
}

func submitSyncPrompt(ldgStore *ledger.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var prompt prompter.Response
		if err := c.BindJSON(&prompt); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		ldgStore.SubmitSyncPrompt(prompt)
		c.Status(http.StatusAccepted)
	}
}

func syncLedger(ldgStore *ledger.Store, accountStore *client.AccountStore, rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, syncFromStart := c.GetQuery("fromLedgerStart")
		sync.Sync(ldgStore, accountStore, rulesStore, syncFromStart)
		c.Status(http.StatusAccepted)
	}
}

type transactionsResponse struct {
	ledger.QueryResult
	AccountIDMap map[string]string
}

func getTransactions(ldgStore *ledger.Store, accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var errs sErrors.Errors
		var page, results int = 1, 10
		if pageQuery, ok := c.GetQuery("page"); ok {
			parsedPage, parseErr := strconv.ParseInt(pageQuery, 10, 64)
			switch {
			case parseErr != nil:
				errs.AddErr(errors.Errorf("Invalid integer: %s", pageQuery))
			case parsedPage < 1:
				errs.AddErr(errors.New("Page must be a positive integer"))
			default:
				page = int(parsedPage)
			}
		}
		if resultsQuery, ok := c.GetQuery("results"); ok {
			parsedResults, parseErr := strconv.ParseInt(resultsQuery, 10, 64)
			switch {
			case parseErr != nil:
				errs.AddErr(errors.Errorf("Invalid integer: %s", resultsQuery))
			case parsedResults < 1 || parsedResults > MaxResults:
				errs.AddErr(errors.Errorf("Results must be a positive integer no more than %d", MaxResults))
			default:
				results = int(parsedResults)
			}
		}
		if len(errs) > 0 {
			abortWithClientError(c, http.StatusBadRequest, errs.ErrOrNil())
			return
		}

		var options ledger.QueryOptions
		if err := c.BindQuery(&options); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		result := transactionsResponse{
			QueryResult:  ldgStore.Query(options, page, results),
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

func getBalances(ldgStore *ledger.Store, accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := getBalancesResponse(ldgStore, accountStore, c.QueryArray(accountTypesQuery))
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func getBalancesResponse(ldgStore *ledger.Store, accountStore *client.AccountStore, accountTypesQueryArray []string) (interface{}, error) {
	start, end, balanceMap := ldgStore.Balances()
	resp := BalanceResponse{
		Start: start,
		End:   end,
	}
	accountIDMap, err := newAccountIDMap(accountStore)
	if err != nil {
		return nil, err
	}

	accountTypes := map[string]bool{
		// return assets and liabilities by default
		// useful for a simple balance table
		model.AssetAccount:     true,
		model.LiabilityAccount: true,
	}
	if len(accountTypesQueryArray) > 0 {
		accountTypes = make(map[string]bool, len(accountTypesQueryArray))
		for _, value := range accountTypesQueryArray {
			accountTypes[value] = true
		}
	}

	var openingBalances ledger.Transaction
	if balances, found := ldgStore.OpeningBalances(); found {
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
		if extractAccount(&account, accountName, accountTypes, accountIDMap.Find) {
			resp.Accounts = append(resp.Accounts, account)
		}
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
		return nil, err
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

	resp.Messages = append(resp.Messages, getOpeningBalanceMessages(ldgStore, accounts)...)
	sort.Slice(resp.Messages, func(a, b int) bool {
		return resp.Messages[a].AccountID < resp.Messages[b].AccountID
	})
	return resp, nil
}

// extractAccount attempts to fill in the account response, returns true if the account should be added
func extractAccount(account *AccountResponse, accountName string, filterAccountTypes map[string]bool, getAccount func(name string) (model.Account, bool)) bool {
	format, err := model.ParseLedgerFormat(accountName)
	if err != nil || format.AccountType == "" {
		return false
	}
	if len(filterAccountTypes) > 0 && !filterAccountTypes[format.AccountType] {
		return false
	}

	account.AccountType = format.AccountType
	switch format.AccountType {
	case model.AssetAccount, model.LiabilityAccount:
		account.Account = format.Institution + " " + format.AccountID
		account.Institution = format.Institution
		if clientAccount, found := getAccount(accountName); found {
			account.Account = clientAccount.Description()
		}
	default:
		account.ID = format.Remaining
		account.Account = account.ID
	}
	return true
}

func getOpeningBalanceMessages(ldgStore *ledger.Store, accounts []model.Account) []AccountMessage {
	var messages []AccountMessage
	var openingPostings []ledger.Posting
	if openingBalances, ok := ldgStore.OpeningBalances(); ok {
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

func getExpenseAndRevenueAccounts(ldgStore *ledger.Store, rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, _, balanceMap := ldgStore.Balances()
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

func updateTransaction(ldgStore *ledger.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		var txnJSON struct {
			ID string // the original transaction's ID
		}
		if err := json.Unmarshal(body, &txnJSON); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		id := txnJSON.ID

		var txn ledger.Transaction
		if err := json.Unmarshal(body, &txn); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		switch err := ldgStore.UpdateTransaction(id, txn).(type) {
		case ledger.Error:
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		case nil: // skip
		default:
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func updateTransactions(ldgStore *ledger.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var txns []struct {
			ID string `binding:"required"` // the original transaction's ID
			ledger.Transaction
		}
		if err := c.BindJSON(&txns); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		newTxns := make(map[string]ledger.Transaction, len(txns))
		for _, txn := range txns {
			newTxns[txn.ID] = txn.Transaction
		}

		switch err := ldgStore.UpdateTransactions(newTxns).(type) {
		case ledger.Error:
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		case nil: // skip
		default:
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func updateOpeningBalance(ldgStore *ledger.Store, accountStore *client.AccountStore) gin.HandlerFunc {
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

		switch err := ldgStore.UpdateOpeningBalance(opening).(type) {
		case ledger.Error:
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		case nil: // skip
		default:
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func importOFXFile(ldgStore *ledger.Store, accountStore *client.AccountStore, rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := c.MustGet(loggerKey).(*zap.Logger)
		skeletonAccounts, txns, err := client.ReadOFX(c.Request.Body)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		rulesStore.ApplyAll(txns)
		switch err := ldgStore.AddTransactions(txns).(type) {
		case ledger.Error:
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		case nil: // skip
		default:
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

func reimportTransactions(ldgStore *ledger.Store, rulesStore *rules.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Start, End string
			Accounts   []string
		}
		if err := c.BindJSON(&body); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		start, end, err := getStartEndTimes(body.Start, body.End, endOfMonth)
		if err != nil {
			abortWithClientError(c, http.StatusOK, err)
			return
		}
		if len(body.Accounts) == 0 {
			body.Accounts = []string{model.Uncategorized}
		}
		result := ldgStore.Query(ledger.QueryOptions{
			Start: start,
			End:   end,
			// currently accounts are fixed to "uncategorized" and "expenses:uncategorized"
			Accounts: body.Accounts,
		}, 1, ldgStore.Size())
		rulesStore.ApplyAll(result.Transactions)
		updatedTxns := make(map[string]ledger.Transaction, len(result.Transactions))
		for _, txn := range result.Transactions {
			updatedTxns[txn.ID()] = txn
		}

		if err := ldgStore.UpdateTransactions(updatedTxns); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		c.JSON(http.StatusOK, map[string]interface{}{
			"Count": result.Count,
		})
	}
}

type renameParams struct {
	Old   string `binding:"required"`
	New   string `binding:"required"`
	OldID string
	NewID string
}

func renameLedgerAccount(ldgStore *ledger.Store) gin.HandlerFunc {
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

		renameCount, err := ldgStore.RenameAccount(params.Old, params.New, params.OldID, params.NewID)
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, map[string]interface{}{
			"Renamed": renameCount,
		})
	}
}

func renameSuggestions(accountStore *client.AccountStore) gin.HandlerFunc {
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
