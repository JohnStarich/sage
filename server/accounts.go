package server

import (
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	sageErrors "github.com/johnstarich/sage/errors"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/sync"
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

func updateAccount(accountsFileName string, accountStore *client.AccountStore, ledgerFileName string, ldg *ledger.Ledger) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Param("id")
		currentAccount, exists := accountStore.Find(accountID)
		if !exists {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		var errs sageErrors.Errors
		abortWithErrors := func() {
			c.AbortWithStatusJSON(http.StatusBadRequest, map[string]string{
				"Error": "New account data is malformed: " + errs.Error(),
			})
		}

		b, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			errs = append(errs, err)
			abortWithErrors()
			return
		}
		account, err := client.UnmarshalBuiltinAccount(b)
		if err != nil {
			errs = append(errs, err)
			abortWithErrors()
			return
		}

		check := func(condition bool, msg string) bool {
			if condition {
				errs = append(errs, errors.New(msg))
			}
			return condition
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

		err = accountStore.Update(accountID, account)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{
				"Error": err.Error(),
			})
			return
		}

		oldAccountName := client.LedgerAccountName(currentAccount)
		newAccountName := client.LedgerAccountName(account)
		// TODO handle condition where account store was updated but ledger rename failed?
		if oldAccountName != newAccountName {
			if err := ldg.UpdateAccount1(oldAccountName, newAccountName); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{
					"Error": err.Error(),
				})
				return
			}
			if err := sync.LedgerFile(ldg, ledgerFileName); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{
					"Error": err.Error(),
				})
				return
			}
		}

		sync.Accounts(accountsFileName, accountStore)
	}
}
