//go:generate mkdir -p ../web/build
//go:generate go run github.com/go-bindata/go-bindata/go-bindata -pkg server -fs -prefix "../web/build" ../web/build/...
//go:generate go fmt ./bindata.go

package server

import (
	"net/http"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/plaindb"
	"github.com/johnstarich/sage/redactor"
	"github.com/johnstarich/sage/rules"
	"github.com/johnstarich/sage/sync"
	"github.com/johnstarich/sage/vcs"
	"go.uber.org/zap"
)

const (
	syncInterval = 4 * time.Hour
	loggerKey    = "logger"
)

// Options contains options for configuring the Sage HTTP server
type Options struct {
	Address  string
	AutoSync bool
	Password redactor.String
}

// Run starts the server
func Run(
	db plaindb.DB,
	ldgStore *ledger.Store,
	accountStore *client.AccountStore,
	rulesFile vcs.File, rulesStore *rules.Store,
	logger *zap.Logger,
	options Options,
) error {
	engine := gin.New()
	engine.Use(
		ginzap.Ginzap(logger, time.RFC3339, true),
		ginzap.RecoveryWithZap(logger, true),
		func(c *gin.Context) {
			c.Set(loggerKey, logger)
		},
	)
	engine.GET("/", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/web") })

	web := engine.Group("/web")
	web.StaticFS("/", newDefaultRouteFS(
		"/index.html",
		AssetFile(),
		"/static/",
	))

	engine.GET("/api/v1/getVersion", getVersion(http.DefaultClient, "api.github.com", "JohnStarich/sage", logger)) // add version route without auth

	api := engine.Group("/api/v1")
	if len(options.Password) > 0 {
		auth := newAuthenticator(options.Password)
		engine.POST("/api/authz", signIn(auth))
		api.Use(requireAuth(auth))
	}
	setupAPI(api, db, ldgStore, accountStore, rulesFile, rulesStore)

	done := make(chan bool, 1)
	errs := make(chan error, 2)

	logger.Info("Starting server", zap.String("addr", options.Address))
	if !options.AutoSync {
		return engine.Run(options.Address)
	}

	go func() {
		// give gin server time to start running. don't perform unnecessary requests if gin fails to boot
		time.Sleep(2 * time.Second)
		runSync := func() {
			sync.Sync(ldgStore, accountStore, rulesStore, false)
		}
		runSync()
		ticker := time.NewTicker(syncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_, _, err := ldgStore.SyncStatus()
				if err == nil {
					// only auto-sync if last sync succeeded
					runSync()
				}
			}
		}
	}()

	go func() {
		errs <- engine.Run(options.Address)
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
	ldgStore *ledger.Store,
	accountStore *client.AccountStore,
	rulesFile vcs.File,
	rulesStore *rules.Store,
) {
	router.GET("/getLedgerSyncStatus", getLedgerSyncStatus(ldgStore))
	router.POST("/submitSyncPrompt", submitSyncPrompt(ldgStore))
	router.POST("/syncLedger", syncLedger(ldgStore, accountStore, rulesStore))
	router.POST("/importOFX", importOFXFile(ldgStore, accountStore, rulesStore))
	router.POST("/renameLedgerAccount", renameLedgerAccount(ldgStore))
	router.GET("/renameSuggestions", renameSuggestions(accountStore))

	router.GET("/getBalances", getBalances(ldgStore, accountStore))
	router.POST("/updateOpeningBalance", updateOpeningBalance(ldgStore, accountStore))
	router.GET("/getCategories", getExpenseAndRevenueAccounts(ldgStore, rulesStore))

	router.GET("/getAccounts", getAccounts(accountStore))
	router.GET("/getAccount", getAccount(accountStore))
	router.POST("/updateAccount", updateAccount(accountStore, ldgStore))
	router.POST("/addAccount", addAccount(accountStore))
	router.GET("/deleteAccount", removeAccount(accountStore))

	router.GET("/web/getDriverNames", getWebConnectDrivers())

	router.GET("/direct/getDrivers", getDirectConnectDrivers())
	router.POST("/direct/verifyAccount", verifyAccount(accountStore))
	router.POST("/direct/fetchAccounts", fetchDirectConnectAccounts())

	router.GET("/getTransactions", getTransactions(ldgStore, accountStore))
	router.POST("/updateTransaction", updateTransaction(ldgStore))
	router.POST("/updateTransactions", updateTransactions(ldgStore))
	router.POST("/reimportTransactions", reimportTransactions(ldgStore, rulesStore))

	router.GET("/getRules", getRules(rulesStore, ldgStore))
	router.GET("/getRule", getRule(rulesStore))
	router.POST("/updateRules", updateRules(rulesFile, rulesStore))
	router.POST("/updateRule", updateRule(rulesFile, rulesStore))
	router.POST("/addRule", addRule(rulesFile, rulesStore))
	router.POST("/deleteRule", deleteRule(rulesFile, rulesStore))

	router.GET("/getBudgets", getBudgets(db, ldgStore))
	router.GET("/getBudget", getBudget(db, ldgStore))
	router.POST("/updateBudget", updateBudget(db))
	router.GET("/deleteBudget", deleteBudget(db))
	router.GET("/getEverythingElseBudget", getEverythingElseBudgetDetails(db, ldgStore))
}
