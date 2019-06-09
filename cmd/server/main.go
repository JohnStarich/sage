package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/johnstarich/sage"
	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/rules"
)

func loadLedger(fileName string) (*ledger.Ledger, error) {
	ledgerFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	defer rulesFile.Close()
	r, err := rules.NewCSVRulesFromReader(rulesFile)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func main() {
	if len(os.Args) < 4 {
		os.Exit(2)
	}
	rulesFile, ledgerFile, ofxClientFile := os.Args[1], os.Args[2], os.Args[3]

	r, err := loadRules(rulesFile)
	if err != nil {
		panic(err)
	}

	ldg, err := loadLedger(ledgerFile)
	if err != nil {
		panic(err)
	}

	accounts, err := client.AccountsFromOFXClientIni(ofxClientFile)
	if err != nil {
		panic(err)
	}
	if len(accounts) == 0 {
		panic("No accounts found in client ini file")
	}

	if err := sage.Sync(ldg, accounts, r); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	if err := ioutil.WriteFile(ledgerFile, []byte(ldg.String()), os.ModePerm); err != nil {
		panic(err)
	}
}
