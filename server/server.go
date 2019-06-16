//go:generate go-bindata -pkg server -fs -prefix "../web/" ../web/

package server

import (
	"net/http"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/consts"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/rules"
	"github.com/johnstarich/sage/sync"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	syncInterval = 4 * time.Hour
	syncKey      = "syncFunc"
	loggerKey    = "logger"
	ledgerKey    = "ledger"
	accountsKey  = "accounts"
	rulesKey     = "rules"
)

func Run(addr, ledgerFileName string, ldg *ledger.Ledger, accounts []client.Account, r rules.Rules, logger *zap.Logger) error {
	runSync := func() error {
		err := sync.Sync(logger, ledgerFileName, ldg, accounts, r)
		if err == nil {
			logger.Info("Sync completed successfully")
			return nil
		}
		return errors.Wrap(err, "Error syncing ledger")
	}

	engine := gin.New()
	engine.Use(
		ginzap.Ginzap(logger, time.RFC3339, true),
		//ginzap.RecoveryWithZap(logger, true), // TODO restore recovery when https://github.com/gin-contrib/zap/pull/10 is merged
		recovery(logger, true),
	)
	engine.GET("/", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/web") })
	engine.StaticFS("/web", AssetFile())

	api := engine.Group("/api/v1")
	api.Use(
		func(c *gin.Context) {
			c.Set(syncKey, runSync)
			c.Set(loggerKey, logger)
			c.Set(ledgerKey, ldg)
			c.Set(accountsKey, accounts)
			c.Set(rulesKey, r)
		},
	)
	setupAPI(api)

	done := make(chan bool, 1)
	errs := make(chan error, 2)

	go func() {
		// give gin server time to start running. don't perform unnecessary requests if gin fails to boot
		time.Sleep(2 * time.Second)
		if err := runSync(); err != nil {
			if _, ok := err.(ledger.Error); !ok {
				// only stop sync loop if NOT a partial error
				errs <- err
				return
			}
		}
		ticker := time.NewTicker(syncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := runSync(); err != nil {
					errs <- err
					return
				}
			}
		}
	}()

	go func() {
		logger.Info("Starting server", zap.String("addr", addr))
		errs <- engine.Run(addr)
		done <- true
	}()

	return <-errs
}

func setupAPI(router gin.IRouter) {
	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{
			"version": consts.Version,
		})
	})

	router.POST("/sync", syncLedger)
	router.GET("/transactions", getTransactions)
	router.GET("/balances", getBalances)
}
