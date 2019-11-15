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
	"github.com/johnstarich/sage/plaindb"
	"github.com/johnstarich/sage/rules"
	"github.com/johnstarich/sage/sync"
	"github.com/johnstarich/sage/vcs"
	"go.uber.org/zap"
)

const (
	syncInterval = 4 * time.Hour
	loggerKey    = "logger"
)

// Run starts the server
func Run(
	autoSync bool,
	addr string,
	db plaindb.DB,
	ledgerFile vcs.File, ldg *ledger.Ledger,
	accountStore *client.AccountStore,
	rulesFile vcs.File, rulesStore *rules.Store,
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
	setupAPI(api, db, ledgerFile, ldg, accountStore, rulesFile, rulesStore)

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
			return sync.Sync(logger, ledgerFile, ldg, accountStore, rulesStore, false)
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

	var lastError error
	for {
		select {
		case err := <-errs:
			lastError = err
			logger.Error("Sync loop errored", zap.Error(err))
		case <-done:
			return lastError
		}
	}
}

func setupAPI(
	router gin.IRouter,
	db plaindb.DB,
	ledgerFile vcs.File,
	ldg *ledger.Ledger,
	accountStore *client.AccountStore,
	rulesFile vcs.File,
	rulesStore *rules.Store,
) {
	router.GET("/getVersion", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{
			"version": consts.Version,
		})
	})

	router.POST("/syncLedger", syncLedger(ledgerFile, ldg, accountStore, rulesStore))
	router.POST("/importOFX", importOFXFile(ledgerFile, ldg, accountStore, rulesStore))
	router.POST("/renameLedgerAccount", renameLedgerAccount(ledgerFile, ldg))
	router.GET("/renameSuggestions", renameSuggestions(ldg, accountStore))

	router.GET("/getBalances", getBalances(ldg, accountStore))
	router.POST("/updateOpeningBalance", updateOpeningBalance(ledgerFile, ldg, accountStore))
	router.GET("/getCategories", getExpenseAndRevenueAccounts(ldg, rulesStore))

	router.GET("/getAccounts", getAccounts(accountStore))
	router.GET("/getAccount", getAccount(accountStore))
	router.POST("/updateAccount", updateAccount(accountStore, ledgerFile, ldg))
	router.POST("/addAccount", addAccount(accountStore))
	router.GET("/deleteAccount", removeAccount(accountStore))

	router.GET("/web/getDriverNames", getWebConnectDrivers())

	router.POST("/direct/verifyAccount", verifyAccount(accountStore))
	router.POST("/direct/fetchAccounts", fetchDirectConnectAccounts())

	router.GET("/getTransactions", getTransactions(ldg, accountStore))
	router.POST("/updateTransaction", updateTransaction(ledgerFile, ldg))

	router.GET("/getRules", getRules(rulesStore))
	router.POST("/updateRules", updateRules(rulesFile, rulesStore))

	router.GET("/getBudgets", getBudgets(db, ldg))
	router.GET("/getBudget", getBudget(db, ldg))
	router.POST("/addBudget", addBudget(db))
	router.POST("/updateBudget", updateBudget(db))
	router.GET("/deleteBudget", deleteBudget(db))
	router.GET("/getEverythingElseBudget", getEverythingElseBudgetDetails(db, ldg))
}
