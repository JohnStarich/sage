package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/budget"
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/plaindb"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	builtinBudget        = "builtin"
	everythingElseBudget = builtinBudget + ":everything else"
)

func isBuiltinBudget(account string) bool {
	return strings.HasPrefix(strings.ToLower(account), builtinBudget+":")
}

type monthlyBudget struct {
	Account string
	Budget  decimal.Decimal
	Balance decimal.Decimal
}

func getStartEndTimes(startQuery, endQuery string, minStart func(end time.Time) time.Time) (start, end time.Time, err error) {
	if endQuery != "" {
		end, err = time.Parse(time.RFC3339, endQuery)
		if err != nil {
			return
		}
	} else {
		end = time.Now()
	}
	if startQuery != "" {
		start, err = time.Parse(time.RFC3339, startQuery)
		if err != nil {
			return
		}
	} else {
		start = minStart(end)
	}
	return
}

func endOfMonth(end time.Time) time.Time {
	return time.Date(end.Year(), end.Month()+1, 0, 0, 0, 0, 0, time.UTC)
}

func startOfMonth(end time.Time) time.Time {
	return time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func twelveMonthsTotal(end time.Time) time.Time {
	return startOfMonth(end).AddDate(0, -11, 0)
}

func addMonths(t time.Time, months int) time.Time {
	return time.Date(t.Year(), t.Month()+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
}

func getEverythingElseSum(accounts budget.Accounts, ldg *ledger.Ledger, start, end time.Time) decimal.Decimal {
	leftOverAccounts := ldg.LeftOverAccountBalances(start, end, everythingElseAccounts(accounts)...)
	var balance decimal.Decimal
	for _, amount := range leftOverAccounts {
		balance = balance.Add(amount.Abs()) // flip sign of revenues so nothing cancels out
	}
	return balance
}

func getBudgets(db plaindb.DB, ldg *ledger.Ledger) gin.HandlerFunc {
	store, err := budget.NewStore(db)
	if err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		start, end, err := getStartEndTimes(c.Query("start"), c.Query("end"), twelveMonthsTotal)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		now := time.Now()
		if end.After(now) {
			end = now
		}

		allMonthlyBudgets := make([]budget.Accounts, 0, 12)
		for current := start; current.Before(end); current = current.AddDate(0, 1, 0) {
			month, err := store.Month(current.Year(), current.Month())
			if err != nil {
				abortWithClientError(c, http.StatusInternalServerError, err)
				return
			}
			allMonthlyBudgets = append(allMonthlyBudgets, month)
		}
		budgetResults, err := calculateBudgetBalances(allMonthlyBudgets, ldg, start, end)
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, struct {
			Start, End string
			Budgets    [][]monthlyBudget
		}{
			Start:   start.UTC().Format(time.RFC3339),
			End:     end.UTC().Format(time.RFC3339),
			Budgets: budgetResults,
		})
	}
}

func calculateBudgetBalances(allMonthlyBudgets []budget.Accounts, ldg *ledger.Ledger, start, end time.Time) ([][]monthlyBudget, error) {
	budgetResults := make([][]monthlyBudget, 0, 12)
	for monthOffset, accounts := range allMonthlyBudgets {
		monthStart := addMonths(start, monthOffset)
		monthEnd := endOfMonth(monthStart)
		foundEverythingElse := false
		monthResults := make([]monthlyBudget, 0, len(accounts)+1)
		for account, budgetAmt := range accounts {
			var balance decimal.Decimal
			if isBuiltinBudget(account) {
				switch strings.ToLower(account) {
				case everythingElseBudget:
					foundEverythingElse = true
					balance = getEverythingElseSum(accounts, ldg, monthStart, monthEnd)
				default:
					return nil, errors.Errorf("Invalid builtin account: %s", account)
				}
			} else {
				balance = ldg.AccountBalance(account, monthStart, monthEnd)
			}
			if strings.HasPrefix(account, model.RevenueAccount+":") || account == model.RevenueAccount {
				balance = balance.Neg()
			}
			monthResults = append(monthResults, monthlyBudget{
				Account: account,
				Budget:  budgetAmt,
				Balance: balance,
			})
		}

		if !foundEverythingElse {
			monthResults = append(monthResults, monthlyBudget{
				Account: everythingElseBudget,
				Balance: getEverythingElseSum(accounts, ldg, monthStart, monthEnd),
			})
		}
		budgetResults = append(budgetResults, monthResults)
	}
	return budgetResults, nil
}

func getBudget(db plaindb.DB, ldg *ledger.Ledger) gin.HandlerFunc {
	store, err := budget.NewStore(db)
	if err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		account := strings.ToLower(c.Query("account"))
		if account == "" {
			abortWithClientError(c, http.StatusBadRequest, errors.New("Account name is required"))
			return
		}

		var start, end time.Time
		if endQuery := c.Query("end"); endQuery != "" {
			var err error
			end, err = time.Parse(time.RFC3339, endQuery)
			if err != nil {
				abortWithClientError(c, http.StatusBadRequest, err)
				return
			}
		} else {
			end = time.Now()
		}
		if startQuery := c.Query("start"); startQuery != "" {
			var err error
			start, err = time.Parse(time.RFC3339, startQuery)
			if err != nil {
				abortWithClientError(c, http.StatusBadRequest, err)
				return
			}
		} else {
			start = startOfMonth(end)
		}
		if start.Month() != end.Month() || start.Year() != end.Year() {
			// no more than one month allowed
			start = startOfMonth(end)
		}

		budget, err := store.Month(start.Year(), start.Month())
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		balance := ldg.AccountBalance(account, start, end)
		if strings.HasPrefix(account, model.RevenueAccount+":") {
			balance = balance.Neg()
		}

		c.JSON(http.StatusOK, struct {
			Start, End string
			Budget     monthlyBudget
		}{
			Start: start.UTC().Format(time.RFC3339),
			End:   end.UTC().Format(time.RFC3339),
			Budget: monthlyBudget{
				Account: account,
				Budget:  budget.Get(account),
				Balance: balance,
			},
		})
	}
}

func updateBudget(db plaindb.DB) gin.HandlerFunc {
	store, err := budget.NewStore(db)
	if err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		var monthBudget monthlyBudget
		if err := c.BindJSON(&monthBudget); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		if isBuiltinBudget(monthBudget.Account) {
			switch strings.ToLower(monthBudget.Account) {
			case everythingElseBudget:
			default:
				abortWithClientError(c, http.StatusBadRequest, errors.Errorf("Invalid builtin account name: %s", monthBudget.Account))
				return
			}
		}

		start, _, err := getStartEndTimes(c.Query("start"), time.Now().Format(time.RFC3339), startOfMonth)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		year, month := start.Year(), start.Month()
		if err := store.SetMonth(year, month, monthBudget.Account, monthBudget.Budget); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func deleteBudget(db plaindb.DB) gin.HandlerFunc {
	store, err := budget.NewStore(db)
	if err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		account := c.Query("budget") // budget is an account name internally
		if account == "" {
			abortWithClientError(c, http.StatusBadRequest, errors.New("Budget name is required"))
			return
		}
		if isBuiltinBudget(account) {
			abortWithClientError(c, http.StatusBadRequest, errors.New("Budget name is reserved and can not be deleted"))
			return
		}

		start, _, err := getStartEndTimes(c.Query("start"), time.Now().Format(time.RFC3339), startOfMonth)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		if err := store.RemoveMonth(start.Year(), start.Month(), account); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func everythingElseAccounts(accounts budget.Accounts) []string {
	accountNames := make([]string, 0, len(accounts)+3)
	accountNames = append(accountNames,
		model.AssetAccount,
		model.LiabilityAccount,
		builtinBudget,
	)
	for account := range accounts {
		accountNames = append(accountNames, account)
	}
	return accountNames
}

func getEverythingElseBudgetDetails(db plaindb.DB, ldg *ledger.Ledger) gin.HandlerFunc {
	store, err := budget.NewStore(db)
	if err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		start, end, err := getStartEndTimes(c.Query("start"), c.Query("end"), startOfMonth)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		if start.Year() != end.Year() || start.Month() != end.Month() {
			start = startOfMonth(end)
		}

		accounts, err := store.Month(start.Year(), start.Month())
		if err != nil {
			abortWithClientError(c, http.StatusNotFound, err)
			return
		}

		leftOverAccounts := ldg.LeftOverAccountBalances(start, end, everythingElseAccounts(accounts)...)
		var sum decimal.Decimal
		for account, balance := range leftOverAccounts {
			if strings.HasPrefix(account, model.RevenueAccount+":") {
				leftOverAccounts[account] = balance.Neg()
			}
			sum = sum.Add(balance)
		}
		c.JSON(http.StatusOK, struct {
			Start    string
			End      string
			Balance  decimal.Decimal
			Accounts map[string]decimal.Decimal
		}{
			Start:    start.UTC().Format(time.RFC3339),
			End:      end.UTC().Format(time.RFC3339),
			Balance:  sum,
			Accounts: leftOverAccounts,
		})
	}
}
