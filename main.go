package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/consts"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/rules"
	"github.com/johnstarich/sage/server"
	"github.com/johnstarich/sage/sync"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func loadLedger(fileName string) (*ledger.Ledger, error) {
	ledgerFile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, errors.Wrapf(err, "Error opening '%s'", fileName)
	}
	defer ledgerFile.Close()
	ldg, err := ledger.NewFromReader(ledgerFile)
	if err != nil {
		return nil, err
	}
	return ldg, nil
}

func loadRules(fileName string) (rules.Rules, error) {
	rulesFile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, errors.Wrapf(err, "Error opening rules file '%s'", fileName)
	}
	defer rulesFile.Close()
	r, err := rules.NewCSVRulesFromReader(rulesFile)
	return r, errors.Wrapf(err, "Error reading rules from file '%s'", fileName)
}

func loadAccounts(fileName string) (*client.AccountStore, error) {
	accountsFile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, errors.Wrapf(err, "Error opening accounts file '%s'", fileName)
	}
	defer accountsFile.Close()
	accountStore, err := client.NewAccountStoreFromReader(accountsFile)
	return accountStore, errors.Wrap(err, "Error reading accounts from file")
}

func start(
	isServer bool, autoSync bool, port uint16,
	ledgerFileName string, ldg *ledger.Ledger,
	accountsFileName string, accountStore *client.AccountStore,
	rulesFileName string, rulesStore *rules.Store,
) error {
	logger, err := zap.NewProduction()
	if os.Getenv("DEVELOPMENT") == "true" {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		return err
	}

	if !isServer {
		return sync.Sync(logger, ledgerFileName, ldg, accountStore, rulesStore, false)
	}
	gin.SetMode(gin.ReleaseMode)
	return server.Run(autoSync, fmt.Sprintf("0.0.0.0:%d", port), ledgerFileName, ldg, accountsFileName, accountStore, rulesFileName, rulesStore, logger)
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

func handleErrors() (usageErr bool, err error) {
	flagSet := flag.NewFlagSet("sage", flag.ContinueOnError)
	isServer := flagSet.Bool("server", false, "Starts the Sage http server and sync on an interval until terminated")
	serverPort := flagSet.Uint("port", 0, "Sets the port the server listens on. Defaults to 8080. Implies -server")
	noSyncLoop := flagSet.Bool("no-auto-sync", false, "Disables ledger auto-sync")
	rulesFileName := flagSet.String("rules", "", "Required: Path to an hledger CSV import rules file")
	ledgerFileName := flagSet.String("ledger", "", "Required: Path to a ledger file")
	accountsFileName := flagSet.String("accounts", "", "Required: Path to an accounts file, includes connection information for institutions")
	requestVersion := flagSet.Bool("version", false, "Print the version and exit")
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
		if port <= 0 || uint(port) != *serverPort {
			return true, errors.Errorf("Port number must be a positive 16-bit integer: %d", *serverPort)
		}
	}

	r, err := loadRules(*rulesFileName)
	if err != nil {
		return false, err
	}
	rulesStore := rules.NewStore(r)

	ldg, err := loadLedger(*ledgerFileName)
	if err != nil {
		return false, err
	}

	accountStore, err := loadAccounts(*accountsFileName)
	if err != nil {
		return false, err
	}
	return false, start(*isServer, !*noSyncLoop, port, *ledgerFileName, ldg, *accountsFileName, accountStore, *rulesFileName, rulesStore)
}

func main() {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c)
		for {
			s := <-c
			switch s {
			case os.Interrupt, syscall.SIGTERM, syscall.SIGUSR2:
				sync.Shutdown(0)
			case os.Kill:
				sync.Shutdown(1)
			default:
				fmt.Println(`{"level":"info","msg":"Handling signal: ` + s.String() + `"}`)
			}
		}
	}()
	usageErr, err := handleErrors()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if usageErr {
			sync.Shutdown(2)
		}
		sync.Shutdown(1)
	}
}
