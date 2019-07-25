package server

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	sageErrors "github.com/johnstarich/sage/errors"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/sync"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func validateAccount(account client.Account) sageErrors.Errors {
	var errs sageErrors.Errors
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
		return errs
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

	u, err := url.Parse(inst.URL())
	if err != nil {
		errs = append(errs, errors.Wrap(err, "Institution URL is malformed"))
	} else {
		check(u.Scheme != "https" && u.Hostname() != "localhost", "Institution URL is required to use HTTPS")
	}

	return errs
}

func abortWithClientError(c *gin.Context, status int, err error) {
	if status/100 == 5 {
		logger := c.MustGet(loggerKey).(*zap.Logger)
		logger.WithOptions(zap.AddCallerSkip(1)).Error("Aborting with server error", zap.Error(err))
	}
	c.AbortWithStatusJSON(status, map[string]string{
		"Error": err.Error(),
	})
}

func readAndValidateAccount(r io.ReadCloser) (client.Account, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	account, err := client.UnmarshalBuiltinAccount(b)
	if err != nil {
		return nil, err
	}

	if err := validateAccount(account); err != nil {
		return nil, err
	}
	return account, nil
}

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

		account, err := readAndValidateAccount(c.Request.Body)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		if pass := account.Institution().Password(); pass.IsEmpty() {
			// if no password provided, use existing password
			pass.Set(currentAccount.Institution().Password())
		}

		if err := accountStore.Update(accountID, account); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		oldAccountName := client.LedgerAccountName(currentAccount)
		newAccountName := client.LedgerAccountName(account)
		// TODO handle condition where account store was updated but ledger rename failed?
		if oldAccountName != newAccountName {
			if err := ldg.UpdateAccount(oldAccountName, newAccountName); err != nil {
				abortWithClientError(c, http.StatusInternalServerError, err)
				return
			}
			if err := sync.LedgerFile(ldg, ledgerFileName); err != nil {
				abortWithClientError(c, http.StatusInternalServerError, err)
				return
			}
		}

		if err := sync.Accounts(accountsFileName, accountStore); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
	}
}

func addAccount(accountsFileName string, accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		account, err := readAndValidateAccount(c.Request.Body)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		if err := accountStore.Add(account); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		if err := sync.Accounts(accountsFileName, accountStore); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func removeAccount(accountsFileName string, accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Param("id")

		if err := accountStore.Remove(accountID); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		if err := sync.Accounts(accountsFileName, accountStore); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func verifyAccount(accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		account, err := readAndValidateAccount(c.Request.Body)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		if err := client.Verify(account); err != nil {
			if err == client.ErrAuthFailed {
				abortWithClientError(c, http.StatusUnauthorized, err)
				return
			}
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		accountID := c.Param("id")
		inst := account.Institution()
		pass := account.Institution().Password()
		if pass.IsEmpty() {
			currentAccount, exists := accountStore.Find(accountID)
			if !exists && !client.IsLocalhostTestURL(inst.URL()) {
				abortWithClientError(c, http.StatusBadRequest, errors.New("Institution password is required"))
				return
			}
			if exists {
				pass.Set(currentAccount.Institution().Password())
			}
		}

		c.Status(http.StatusNoContent)
	}
}
