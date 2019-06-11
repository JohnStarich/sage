package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/rules"
	"github.com/johnstarich/sage/sync"
	"github.com/pkg/errors"
)

const (
	syncInterval = 4 * time.Hour
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

func start(server bool, ledgerFileName string, ldg *ledger.Ledger, accounts []client.Account, r rules.Rules) error {
	for {
		if err := sync.Sync(ldg, accounts, r); err != nil {
			fmt.Fprintln(os.Stderr, errors.Wrap(err, "Error syncing ledger"))
		}

		if err := ioutil.WriteFile(ledgerFileName, []byte(ldg.String()), os.ModePerm); err != nil {
			return errors.Wrap(err, "Error writing updated ledger to disk")
		}
		if !server {
			return nil
		}
		time.Sleep(syncInterval)
	}
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
	ofxClientFileName := flagSet.String("ofxclient", "", "Required: Path to an ofxclient ini file, includes connection information for institutions")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return true, err
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

	accounts, err := client.AccountsFromOFXClientINI(*ofxClientFileName)
	if err != nil {
		return false, err
	}
	if len(accounts) == 0 {
		return false, errors.New("No accounts found in client ini file")
	}
	return false, start(*enableServer, *ledgerFileName, ldg, accounts, r)
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
