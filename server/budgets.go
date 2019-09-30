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

type monthlyBudget struct {
	budgets.Budget

	Amount decimal.Decimal
}

func getBudgets(db plaindb.DB, ldg *ledger.Ledger) gin.HandlerFunc {
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
		sort.Slice(budgets, func(a, b int) bool {
			return budgets[a].Account < budgets[b].Account
		})

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
		monthlyBudgets := make([]monthlyBudget, 0, len(budgets))
		for _, b := range budgets {
			balance := ldg.AccountBalance(b.Account, start, end)
			if strings.HasPrefix(b.Account, model.RevenueAccount+":") {
				balance = balance.Neg()
			}
			monthlyBudgets = append(monthlyBudgets, monthlyBudget{
				Budget: b,
				Amount: balance,
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
		account := c.Query("account")
		if account == "" {
			abortWithClientError(c, http.StatusBadRequest, errors.New("Account name is required"))
			return
		}
		if err := store.Remove(account); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
