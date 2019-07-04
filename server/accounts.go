package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/johnstarich/sage/client"
	sageErrors "github.com/johnstarich/sage/errors"
	"github.com/pkg/errors"
)

func getAccount(accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Param("id")
		account, exists := accountStore.Find(accountID)
		if !exists {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusOK, map[string]interface{}{
			"Account": account,
		})
	}
}

func getAccounts(accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"Accounts": accountStore,
		})
	}
}

type bankLike struct {
	AccountType   string
	RoutingNumber string
}

func (b bankLike) isBank() bool {
	return b.RoutingNumber != ""
}

func updateAccount(accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Param("id")
		currentAccount, exists := accountStore.Find(accountID)
		if !exists {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		var account client.Account

		maybeBank := bankLike{}
		if err := c.ShouldBindBodyWith(&maybeBank, binding.JSON); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		maybeBank.AccountType = strings.ToUpper(maybeBank.AccountType)
		if maybeBank.isBank() {
			if client.IsChecking(maybeBank.AccountType) {
				checkingAccount := &client.Checking{}
				if err := c.ShouldBindBodyWith(checkingAccount, binding.JSON); err != nil {
					c.AbortWithError(http.StatusBadRequest, err)
					return
				}
				account = checkingAccount
			} else if client.IsSavings(maybeBank.AccountType) {
				savingsAccount := &client.Savings{}
				if err := c.ShouldBindBodyWith(savingsAccount, binding.JSON); err != nil {
					c.AbortWithError(http.StatusBadRequest, err)
					return
				}
				account = savingsAccount
			} else {
				c.AbortWithError(http.StatusBadRequest, errors.New("Invalid bank AccountType"))
				return
			}
		} else {
			creditCard := &client.CreditCard{}
			if err := c.ShouldBindBodyWith(creditCard, binding.JSON); err != nil {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
			account = creditCard
		}

		var errs sageErrors.Errors
		check := func(condition bool, msg string) bool {
			if condition {
				errs = append(errs, errors.New(msg))
			}
			return condition
		}
		abortWithErrors := func() {
			c.AbortWithStatusJSON(http.StatusBadRequest, map[string]string{
				"Error": "New account data is malformed: " + errs.Error(),
			})
		}

		check(account.ID() == "", "Account ID is required")
		check(account.Description() == "", "Account description is required")
		inst := account.Institution()
		if check(inst == nil, "Institution is required") {
			abortWithErrors()
			return
		}

		check(inst.Description() == "", "Institution description is required")
		check(inst.FID() == "", "Institution FID is required")
		check(inst.Org() == "", "Institution Org is required")
		check(inst.URL() == "", "Institution URL is required")
		check(inst.Username() == "", "Institution username is required")

		switch impl := account.(type) {
		case client.Bank:
			check(impl.BankID() == "", "Bank ID is required")
		}

		if len(errs) > 0 {
			abortWithErrors()
			return
		}

		if inst.Password().IsEmpty() {
			// if no password provided, use existing password
			inst.Password().Set(currentAccount.Institution().Password())
		}

		accountStore.Update(accountID, account)
		// TODO sync to disk
	}
}
