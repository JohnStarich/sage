package server

import (
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/client/direct"
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/sync"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

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

func readAndValidateAccount(r io.ReadCloser, accountStore *client.AccountStore) (model.Account, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	account, err := client.UnmarshalAccount(b)
	if err != nil {
		return nil, err
	}

	if connector, ok := account.Institution().(direct.Connector); ok && connector.Password() == "" {
		currentAccount, found := accountStore.Find(account.ID())
		if found {
			currentConn, currentOK := currentAccount.Institution().(direct.Connector)
			if currentOK {
				connector.SetPassword(currentConn.Password())
			}
		}
	}

	err = client.ValidateAccount(account)
	return account, err
}

func readAndValidateDirectConnector(r io.ReadCloser) (direct.Connector, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	connector, err := direct.UnmarshalConnector(b)
	if err != nil {
		return nil, err
	}
	return connector, direct.ValidateConnector(connector)
}

func getAccount(accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Query("id")
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
		account, err := readAndValidateAccount(c.Request.Body, accountStore)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		currentAccount, exists := accountStore.Find(account.ID())
		if !exists {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		if err := accountStore.Update(account.ID(), account); err != nil {
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
		account, err := readAndValidateAccount(c.Request.Body, accountStore)
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
		accountID := c.Query("id")

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
		account, err := readAndValidateAccount(c.Request.Body, accountStore)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		connector, isConn := account.Institution().(direct.Connector)
		if !isConn {
			abortWithClientError(c, http.StatusBadRequest, errors.New("Cannot verify account: no direct connect details"))
			return
		}
		requestor, isReq := account.(direct.Requestor)
		if !isReq {
			abortWithClientError(c, http.StatusBadRequest, errors.Errorf("Cannot verify account: account is invalid type: %T", account))
			return
		}
		if err := direct.Verify(connector, requestor); err != nil {
			if err == direct.ErrAuthFailed {
				abortWithClientError(c, http.StatusUnauthorized, err)
				return
			}
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		pass := connector.Password()
		if pass == "" {
			currentAccount, exists := accountStore.Find(account.ID())
			errPasswordRequired := errors.New("Institution password is required")
			isLocal := direct.IsLocalhostTestURL(connector.URL())
			if !isLocal {
				if !exists {
					abortWithClientError(c, http.StatusBadRequest, errPasswordRequired)
					return
				}
				currentConnector, isConn := currentAccount.Institution().(direct.Connector)
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

func fetchDirectConnectAccounts() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := c.MustGet(loggerKey).(*zap.Logger)

		connector, err := readAndValidateDirectConnector(c.Request.Body)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		accounts, err := direct.Accounts(connector, logger)
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, accounts)
	}
}
