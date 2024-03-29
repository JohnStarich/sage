package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	_ "github.com/johnstarich/sage/client/direct/drivers"
	_ "github.com/johnstarich/sage/client/web/drivers"
	"github.com/johnstarich/sage/consts"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/plaindb"
	"github.com/johnstarich/sage/redactor"
	"github.com/johnstarich/sage/rules"
	"github.com/johnstarich/sage/server"
	"github.com/johnstarich/sage/sync"
	"github.com/johnstarich/sage/vcs"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func loadRules(fileName string) (rules.Rules, error) {
	rulesFile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, errors.Wrapf(err, "Error opening rules file '%s'", fileName)
	}
	defer rulesFile.Close()
	r, err := rules.NewCSVRulesFromReader(rulesFile)
	return r, errors.Wrapf(err, "Error reading rules from file '%s'", fileName)
}

func getLogger() (*zap.Logger, error) {
	if os.Getenv("DEVELOPMENT") == "true" {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}

func start(
	isServer bool,
	db plaindb.DB,
	ldgStore *ledger.Store,
	accountStore *client.AccountStore,
	rulesFile vcs.File, rulesStore *rules.Store,
	logger *zap.Logger,
	options server.Options,
) error {
	if !isServer {
		sync.Sync(ldgStore, accountStore, rulesStore, false)
		for {
			// TODO add CLI prompt support
			syncing, _, err := ldgStore.SyncStatus()
			if !syncing {
				return err
			}
			time.Sleep(time.Second)
		}
	}
	gin.SetMode(gin.ReleaseMode)
	err := server.Run(db, ldgStore, accountStore, rulesFile, rulesStore, logger, options)
	if err != nil {
		logger.Error("Server run failed", zap.Error(err))
	}
	return err
}

func usage(flagSet *flag.FlagSet) string {
	oldOutput := flagSet.Output()
	buf := bytes.NewBuffer(nil)
	flagSet.SetOutput(buf)
	flagSet.Usage()
	flagSet.SetOutput(oldOutput)
	return buf.String()
}

func requireFlags(flagSet *flag.FlagSet) (err error) {
	setFlags := make(map[string]bool)
	flagSet.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})
	var missingFlags []string
	flagSet.VisitAll(func(f *flag.Flag) {
		if strings.HasPrefix(f.Usage, "Required: ") && !setFlags[f.Name] {
			missingFlags = append(missingFlags, f.Name)
		}
	})
	if len(missingFlags) > 0 {
		return errors.Errorf("Missing required flags: %s", missingFlags)
	}
	return nil
}

func handleErrors(db *plaindb.DB) (usageErr bool, err error) {
	flagSet := flag.NewFlagSet("sage", flag.ContinueOnError)
	isServer := flagSet.Bool("server", false, "Starts the Sage http server and sync on an interval until terminated")
	serverPort := flagSet.Uint("port", 0, "Sets the port the server listens on. Defaults to 8080. Implies -server")
	noSyncLoop := flagSet.Bool("no-auto-sync", false, "Disables ledger auto-sync")
	rulesFileName := flagSet.String("rules", "", "Required: Path to an hledger CSV import rules file")
	ledgerFileName := flagSet.String("ledger", "", "Required: Path to a ledger file")
	dbDirName := flagSet.String("data", "", "Required: Path to a database directory")
	requestVersion := flagSet.Bool("version", false, "Print the version and exit")
	serverPassword := flagSet.String("password", "", "A password to lock the web UI and API")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return true, err
	}
	if *requestVersion {
		fmt.Println(consts.Version)
		return false, nil
	}

	if err := requireFlags(flagSet); err != nil {
		return true, errors.Errorf("%s\n%s", err.Error(), usage(flagSet))
	}

	*isServer = *isServer || *serverPort != 0
	if *serverPort == 0 {
		*serverPort = 8080
	}
	var port uint16
	if *isServer {
		port = uint16(*serverPort)
		if uint(port) != *serverPort {
			return true, errors.Errorf("Port number must be a positive 16-bit integer: %d", *serverPort)
		}
	}

	var repo vcs.Repository
	*db, err = plaindb.Open(*dbDirName, plaindb.VersionControl(&repo))
	if err != nil {
		return false, err
	}

	accountStore, err := client.NewAccountStore(*db)
	if err != nil {
		return false, err
	}

	logger, err := getLogger()
	if err != nil {
		return false, err
	}

	ldgStore, err := ledger.NewStore(repo.File(*ledgerFileName), logger)
	if err != nil {
		return false, err
	}

	r, err := loadRules(*rulesFileName)
	if err != nil {
		return false, err
	}
	rulesStore := rules.NewStore(r)
	rulesFile := repo.File(*rulesFileName)

	return false, start(*isServer, *db, ldgStore, accountStore, rulesFile, rulesStore, logger, server.Options{
		Address:  fmt.Sprintf("0.0.0.0:%d", port),
		AutoSync: !*noSyncLoop,
		Password: redactor.String(*serverPassword),
	})
}

func main() {
	var db plaindb.DB

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c)
		for {
			s := <-c
			fmt.Println(`{"level":"info","msg":"Handling signal: ` + s.String() + `"}`)
			switch s {
			case os.Interrupt:
				sync.Shutdown(db, 0)
			case os.Kill:
				sync.Shutdown(db, 1)
			}
		}
	}()
	usageErr, err := handleErrors(&db)
	if err != nil && err != flag.ErrHelp {
		fmt.Fprintln(os.Stderr, err)
		if usageErr {
			sync.Shutdown(db, 2)
		}
		sync.Shutdown(db, 1)
	}
}
