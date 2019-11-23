package server

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/budgets"
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
	budgets.Budget

	Amount decimal.Decimal
}

func getStartEndTimes(startQuery, endQuery string) (start, end time.Time, err error) {
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
		// round down to beginning of this month
		start = time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	return
}

func getEverythingElseSum(allBudgets []budgets.Budget, ldg *ledger.Ledger, start, end time.Time) decimal.Decimal {
	leftOverAccounts := ldg.LeftOverAccountBalances(start, end, everythingElseAccounts(allBudgets)...)
	var balance decimal.Decimal
	for _, amount := range leftOverAccounts {
		balance = balance.Add(amount.Abs()) // flip sign of revenues so nothing cancels out
	}
	return balance
}

func getBudgets(db plaindb.DB, ldg *ledger.Ledger) gin.HandlerFunc {
	store, err := budgets.New(db)
	if err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		allBudgets, err := store.GetAll()
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		sort.Slice(allBudgets, func(a, b int) bool {
			return allBudgets[a].Account < allBudgets[b].Account
		})

		start, end, err := getStartEndTimes(c.Query("start"), c.Query("end"))
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		foundEverythingElse := false
		monthlyBudgets := make([]monthlyBudget, 0, len(allBudgets))
		for _, b := range allBudgets {
			var balance decimal.Decimal
			if isBuiltinBudget(b.Account) {
				switch strings.ToLower(b.Account) {
				case everythingElseBudget:
					foundEverythingElse = true
					balance = getEverythingElseSum(allBudgets, ldg, start, end)
				default:
					abortWithClientError(c, http.StatusInternalServerError, errors.Errorf("Invalid builtin account: %s", b.Account))
					return
				}
			} else {
				balance = ldg.AccountBalance(b.Account, start, end)
			}
			if strings.HasPrefix(b.Account, model.RevenueAccount+":") || b.Account == model.RevenueAccount {
				balance = balance.Neg()
			}
			monthlyBudgets = append(monthlyBudgets, monthlyBudget{
				Budget: b,
				Amount: balance,
			})
		}

		if !foundEverythingElse {
			monthlyBudgets = append(monthlyBudgets, monthlyBudget{
				Budget: budgets.Budget{Account: everythingElseBudget},
				Amount: getEverythingElseSum(allBudgets, ldg, start, end),
			})
		}

		c.JSON(http.StatusOK, struct {
			Start, End string
			Budgets    []monthlyBudget
		}{
			Start:   start.UTC().Format(time.RFC3339),
			End:     end.UTC().Format(time.RFC3339),
			Budgets: monthlyBudgets,
		})
	}
}

func getBudget(db plaindb.DB, ldg *ledger.Ledger) gin.HandlerFunc {
	store, err := budgets.New(db)
	if err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		account := c.Query("account")
		if account == "" {
			abortWithClientError(c, http.StatusBadRequest, errors.New("Account name is required"))
			return
		}
		budget, err := store.Get(account)
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
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
			start = time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
		}

		balance := ldg.AccountBalance(budget.Account, start, end)
		if strings.HasPrefix(budget.Account, model.RevenueAccount+":") {
			balance = balance.Neg()
		}

		c.JSON(http.StatusOK, struct {
			Start, End string
			Budget     monthlyBudget
		}{
			Start: start.UTC().Format(time.RFC3339),
			End:   end.UTC().Format(time.RFC3339),
			Budget: monthlyBudget{
				Budget: budget,
				Amount: balance,
			},
		})
	}
}

func addBudget(db plaindb.DB) gin.HandlerFunc {
	store, err := budgets.New(db)
	if err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		var budget budgets.Budget
		if err := c.BindJSON(&budget); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		if isBuiltinBudget(budget.Account) {
			abortWithClientError(c, http.StatusBadRequest, errors.New("Account name is reserved and can not be added"))
			return
		}
		if err := store.Add(budget); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func updateBudget(db plaindb.DB) gin.HandlerFunc {
	store, err := budgets.New(db)
	if err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		var budget budgets.Budget
		if err := c.BindJSON(&budget); err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}
		if isBuiltinBudget(budget.Account) {
			switch strings.ToLower(budget.Account) {
			case everythingElseBudget:
			default:
				abortWithClientError(c, http.StatusBadRequest, errors.Errorf("Invalid builtin account name: %s", budget.Account))
				return
			}
			// ensure exists
			_, getErr := store.Get(budget.Account)
			if getErr != nil {
				if err := store.Add(budget); err != nil {
					abortWithClientError(c, http.StatusInternalServerError, err)
					return
				}
			}
		}
		if err := store.Update(budget.Account, budget); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func deleteBudget(db plaindb.DB) gin.HandlerFunc {
	store, err := budgets.New(db)
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
		if err := store.Remove(account); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func everythingElseAccounts(budgets []budgets.Budget) []string {
	accounts := make([]string, 0, len(budgets)+3)
	accounts = append(accounts,
		model.AssetAccount,
		model.LiabilityAccount,
		builtinBudget,
	)
	for _, b := range budgets {
		accounts = append(accounts, b.Account)
	}
	return accounts
}

func getEverythingElseBudgetDetails(db plaindb.DB, ldg *ledger.Ledger) gin.HandlerFunc {
	store, err := budgets.New(db)
	if err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		budgets, err := store.GetAll()
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		start, end, err := getStartEndTimes(c.Query("start"), c.Query("end"))
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		leftOverAccounts := ldg.LeftOverAccountBalances(start, end, everythingElseAccounts(budgets)...)
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
			Amount   decimal.Decimal
			Accounts map[string]decimal.Decimal
		}{
			Start:    start.UTC().Format(time.RFC3339),
			End:      end.UTC().Format(time.RFC3339),
			Amount:   sum,
			Accounts: leftOverAccounts,
		})
	}
}
