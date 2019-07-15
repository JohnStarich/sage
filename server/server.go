//go:generate go run github.com/go-bindata/go-bindata/go-bindata -pkg server -fs -prefix "../web/build" ../web/build/...
//go:generate go fmt ./bindata.go

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
	"go.uber.org/zap"
)

const (
	syncInterval = 4 * time.Hour
	fullSyncKey  = "fullSyncFunc"
	fileSyncKey  = "fileSyncFunc"
	loggerKey    = "logger"
	ledgerKey    = "ledger"
	accountsKey  = "accounts"
	rulesKey     = "rules"
)

// Run starts the server
func Run(
	autoSync bool,
	addr string,
	ledgerFileName string, ldg *ledger.Ledger,
	accountsFileName string, accountStore *client.AccountStore,
	rulesFileName string, rulesStore *rules.Store,
	logger *zap.Logger,
) error {
	engine := gin.New()
	engine.Use(
		ginzap.Ginzap(logger, time.RFC3339, true),
		//ginzap.RecoveryWithZap(logger, true), // TODO restore recovery when https://github.com/gin-contrib/zap/pull/10 is merged
		recovery(logger, true),
	)
	engine.GET("/", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/web") })
	engine.StaticFS("/web", newDefaultRouteFS(
		"/index.html",
		AssetFile(),
		"/static/",
	))

	api := engine.Group("/api/v1")
	api.Use(
		func(c *gin.Context) {
			c.Set(loggerKey, logger)
		},
	)
	setupAPI(api, ledgerFileName, ldg, accountsFileName, accountStore, rulesFileName, rulesStore)

	done := make(chan bool, 1)
	errs := make(chan error, 2)

	logger.Info("Starting server", zap.String("addr", addr))
	if !autoSync {
		return engine.Run(addr)
	}

	go func() {
		// give gin server time to start running. don't perform unnecessary requests if gin fails to boot
		time.Sleep(2 * time.Second)
		runSync := func() error {
			return sync.Sync(logger, ledgerFileName, ldg, accountStore, rulesStore)
		}
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
		errs <- engine.Run(addr)
		done <- true
	}()

	return <-errs
}

func setupAPI(
	router gin.IRouter,
	ledgerFileName string,
	ldg *ledger.Ledger,
	accountsFileName string,
	accountStore *client.AccountStore,
	rulesFileName string,
	rulesStore *rules.Store,
) {
	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{
			"version": consts.Version,
		})
	})

	router.POST("/sync", syncLedger(ledgerFileName, ldg, accountStore, rulesStore))
	router.GET("/balances", getBalances(ldg, accountStore))
	router.GET("/categories", getExpenseAndRevenueAccounts(ldg, rulesStore))

	router.GET("/accounts", getAccounts(accountStore))
	router.GET("/accounts/:id", getAccount(accountStore))
	router.PUT("/accounts/:id", updateAccount(accountsFileName, accountStore, ledgerFileName, ldg))
	router.POST("/accounts", addAccount(accountsFileName, accountStore))
	router.DELETE("/accounts/:id", removeAccount(accountsFileName, accountStore))
	router.POST("/accounts/:id/verify", verifyAccount(accountStore))

	router.GET("/transactions", getTransactions(ldg, accountStore))
	router.PATCH("/transactions/:id", updateTransaction(ledgerFileName, ldg))

	router.GET("/rules", getRules(rulesStore))
	router.PUT("/rules", updateRules(rulesFileName, rulesStore))
}
