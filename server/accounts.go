package server

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/client/directconnect"
	"github.com/johnstarich/sage/client/model"
	sErrors "github.com/johnstarich/sage/errors"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/sync"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func validateAccount(account model.Account) error {
	var errs sErrors.Errors
	errs.ErrIf(account.ID() == "", "Account ID is required")
	errs.ErrIf(account.Description() == "", "Account description is required")
	inst := account.Institution()
	if errs.ErrIf(inst == nil, "Institution is required") {
		return errs
	}

	errs.ErrIf(inst.Description() == "", "Institution description is required")
	errs.ErrIf(inst.FID() == "", "Institution FID is required")
	errs.ErrIf(inst.Org() == "", "Institution Org is required")

	switch impl := account.(type) {
	case directconnect.Bank:
		errs.ErrIf(impl.BankID() == "", "Bank ID is required")
	}

	if connector, ok := inst.(directconnect.Connector); ok {
		errs.ErrIf(connector.URL() == "", "Institution URL is required")
		errs.ErrIf(connector.Username() == "", "Institution username is required")
		u, err := url.Parse(connector.URL())
		if err != nil {
			errs.AddErr(errors.Wrap(err, "Institution URL is malformed"))
		} else {
			errs.ErrIf(u.Scheme != "https" && u.Hostname() != "localhost", "Institution URL is required to use HTTPS")
		}
	}

	return errs.ErrOrNil()
}

func abortWithClientError(c *gin.Context, status int, err error) {
	logger := c.MustGet(loggerKey).(*zap.Logger)
	logger.WithOptions(zap.AddCallerSkip(1))
	if status/100 == 5 {
		logger.Error("Aborting with server error", zap.Error(err))
	} else {
		logger.Info("Aborting with client error", zap.String("error", err.Error()))
	}
	c.AbortWithStatusJSON(status, map[string]string{
		"Error": err.Error(),
	})
}

func readAndValidateAccount(r io.ReadCloser) (model.Account, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	account, err := directconnect.UnmarshalAccount(b)
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

		{
			connector, ok := account.Institution().(directconnect.Connector)
			currentConnector, currentOK := currentAccount.Institution().(directconnect.Connector)
			if ok && currentOK {
				if pass := connector.Password(); pass == "" {
					// if no password provided, use existing password
					connector.SetPassword(currentConnector.Password())
				}
			}
		}

		if err := accountStore.Update(accountID, account); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		oldAccountName := model.LedgerAccountName(currentAccount)
		newAccountName := model.LedgerAccountName(account)
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

		connector, isConn := account.Institution().(directconnect.Connector)
		if !isConn {
			abortWithClientError(c, http.StatusBadRequest, errors.New("Cannot verify account: no direct connect details"))
			return
		}
		requestor, isReq := account.(directconnect.Requestor)
		if !isReq {
			abortWithClientError(c, http.StatusBadRequest, errors.Errorf("Cannot verify account: account is invalid type: %T", account))
			return
		}
		if err := directconnect.Verify(connector, requestor); err != nil {
			if err == directconnect.ErrAuthFailed {
				abortWithClientError(c, http.StatusUnauthorized, err)
				return
			}
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		accountID := c.Param("id")
		pass := connector.Password()
		if pass == "" {
			currentAccount, exists := accountStore.Find(accountID)
			errPasswordRequired := errors.New("Institution password is required")
			isLocal := directconnect.IsLocalhostTestURL(connector.URL())
			if !isLocal {
				if !exists {
					abortWithClientError(c, http.StatusBadRequest, errPasswordRequired)
					return
				}
				currentConnector, isConn := currentAccount.Institution().(directconnect.Connector)
				if isConn {
					pass = currentConnector.Password()
				}
				if pass == "" {
					abortWithClientError(c, http.StatusBadRequest, errPasswordRequired)
					return
				}
			}
		}

		c.Status(http.StatusNoContent)
	}
}
