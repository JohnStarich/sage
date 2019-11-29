package server

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/client/direct"
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/client/web"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/sync"
	"github.com/johnstarich/sage/vcs"
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

func readAndValidateAccount(r io.Reader, accountStore *client.AccountStore) (originalAccountID string, account model.Account, err error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", nil, err
	}
	var original struct {
		PreviousAccountID string
	}
	if err := json.Unmarshal(b, &original); err != nil {
		return "", nil, err
	}

	account, err = client.UnmarshalAccount(b)
	if err != nil {
		return "", nil, err
	}
	originalAccountID = account.ID()
	if original.PreviousAccountID != "" {
		originalAccountID = original.PreviousAccountID
	}

	if connector, ok := account.Institution().(direct.Connector); ok && connector.Password() == "" {
		var currentAccount model.Account
		found, err := accountStore.Get(originalAccountID, &currentAccount)
		if err != nil {
			return "", nil, err
		}
		if found {
			currentConn, currentOK := currentAccount.Institution().(direct.Connector)
			if currentOK {
				connector.SetPassword(currentConn.Password())
			}
		}
	} else if connector, ok := account.Institution().(web.PasswordConnector); ok && connector.Password() == "" {
		// TODO combine these implementations?
		var currentAccount model.Account
		found, err := accountStore.Get(originalAccountID, &currentAccount)
		if err != nil {
			return "", nil, err
		}
		if found {
			currentConn, currentOK := currentAccount.Institution().(web.PasswordConnector)
			if currentOK {
				connector.SetPassword(currentConn.Password())
			}
		}
	}

	err = client.ValidateAccount(account)
	return originalAccountID, account, err
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
		var account model.Account
		exists, err := accountStore.Get(accountID, &account)
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
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
		var accounts []model.Account
		var account model.Account
		err := accountStore.Iter(&account, func(id string) bool {
			accounts = append(accounts, account)
			return true
		})
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, map[string]interface{}{
			"Accounts": accounts,
		})
	}
}

func updateAccount(accountStore *client.AccountStore, ledgerFile vcs.File, ldg *ledger.Ledger) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, account, err := readAndValidateAccount(c.Request.Body, accountStore)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		var currentAccount model.Account
		exists, err := accountStore.Get(accountID, &currentAccount)
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
		if !exists {
			abortWithClientError(c, http.StatusNotFound, errors.Errorf("Account not found with ID: %q", accountID))
			return
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
			if err := sync.LedgerFile(ldg, ledgerFile); err != nil {
				abortWithClientError(c, http.StatusInternalServerError, err)
				return
			}
		}
	}
}

func addAccount(accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, account, err := readAndValidateAccount(c.Request.Body, accountStore)
		if err != nil {
			abortWithClientError(c, http.StatusBadRequest, err)
			return
		}

		if err := accountStore.Add(account); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func removeAccount(accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Query("id")

		if err := accountStore.Remove(accountID); err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func verifyAccount(accountStore *client.AccountStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, account, err := readAndValidateAccount(c.Request.Body, accountStore)
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
		if err := direct.Verify(connector, requestor, client.ParseOFX); err != nil {
			if err == direct.ErrAuthFailed {
				abortWithClientError(c, http.StatusUnauthorized, err)
				return
			}
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}

		pass := connector.Password()
		if pass != "" {
			c.Status(http.StatusNoContent)
			return
		}

		// attempt to pull previous password value
		var currentAccount model.Account
		exists, err := accountStore.Get(account.ID(), &currentAccount)
		if err != nil {
			abortWithClientError(c, http.StatusInternalServerError, err)
			return
		}
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

func getWebConnectDrivers() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"DriverNames": web.Drivers(),
		})
	}
}

func getDirectConnectDrivers() gin.HandlerFunc {
	return func(c *gin.Context) {
		results := direct.Search(c.Query("search"))

		type driverResult struct {
			ID          string
			Description string
		}
		response := make([]driverResult, 0, len(results))
		for _, result := range results {
			response = append(response, driverResult{
				ID:          result.ID(),
				Description: result.Description(),
			})
		}
		c.JSON(http.StatusOK, map[string]interface{}{
			"Drivers": response,
		})
	}
}
