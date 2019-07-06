package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

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
	ledgerFile, err := os.Open(fileName)
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
	rulesFile, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "Error opening '%s'", fileName)
	}
	defer rulesFile.Close()
	r, err := rules.NewCSVRulesFromReader(rulesFile)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func loadAccounts(fileName string) (*client.AccountStore, error) {
	accountsFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer accountsFile.Close()
	return client.NewAccountStoreFromReader(accountsFile)
}

func start(isServer bool, ledgerFileName string, ldg *ledger.Ledger, accountsFileName string, accountStore *client.AccountStore, r rules.Rules) error {
	logger, err := zap.NewProduction()
	if os.Getenv("DEVELOPMENT") == "true" {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		return err
	}

	if !isServer {
		return sync.Sync(logger, ledgerFileName, ldg, accountStore, r)
	}
	gin.SetMode(gin.ReleaseMode)
	return server.Run("0.0.0.0:8080", ledgerFileName, ldg, accountsFileName, accountStore, r, logger)
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
	enableServer := flagSet.Bool("server", false, "Syncs on an interval until terminated")
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

	r, err := loadRules(*rulesFileName)
	if err != nil {
		return false, err
	}

	ldg, err := loadLedger(*ledgerFileName)
	if err != nil {
		return false, err
	}

	accountStore, err := loadAccounts(*accountsFileName)
	if err != nil {
		return false, err
	}
	return false, start(*enableServer, *ledgerFileName, ldg, *accountsFileName, accountStore, r)
}

func main() {
	usageErr, err := handleErrors()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if usageErr {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
