package sync

import (
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/rules"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	days = 24 * time.Hour
)

var (
	mu sync.Mutex // basic protection against concurrent sync operations
)

func Sync(logger *zap.Logger, ledgerFileName string, ldg *ledger.Ledger, accounts []client.Account, r rules.Rules) error {
	mu.Lock()
	defer mu.Unlock()
	ledgerErr := Ledger(logger, ldg, accounts, r)
	if ledgerErr != nil {
		if _, ok := ledgerErr.(ledger.Error); !ok {
			return ledgerErr
		}
	}
	if err := File(ldg, ledgerFileName); err != nil {
		return err
	}
	return ledgerErr
}

func Ledger(logger *zap.Logger, ldg *ledger.Ledger, accounts []client.Account, r rules.Rules) error {
	return ledgerSync(logger, ldg, r, downloadTxns(accounts))
}

func ledgerSync(logger *zap.Logger, ldg *ledger.Ledger, r rules.Rules, download func(start, end time.Time) ([]ledger.Transaction, error)) error {
	if err := ldg.Validate(); err != nil {
		return errors.Wrap(err, "Existing ledger is not valid")
	}

	now := time.Now()
	// TODO use smart first date selection on a per-account basis
	const syncBuffer = 2 * days
	duration := now.Sub(ldg.LastTransactionTime())
	duration -= syncBuffer

	const maxDownloadDuration = 30 * days
	beforeStart := now.Add(-duration)

	var allTxns []ledger.Transaction
	downloadedTime := beforeStart
	for downloadedTime.Before(now) {
		end := min(now, downloadedTime.Add(maxDownloadDuration))
		logger.Info("Downloading txns...", zap.Time("start", downloadedTime), zap.Time("end", end))
		txns, err := download(downloadedTime, end)
		if err != nil {
			return err
		}
		allTxns = append(allTxns, txns...)
		downloadedTime = end
	}

	// throw out extra transactions that were included by the institution responses
	filteredTxns := make([]ledger.Transaction, 0, len(allTxns))
	for _, t := range allTxns {
		if beforeStart.After(t.Date) {
			continue
		}
		filteredTxns = append(filteredTxns, t)
	}
	allTxns = filteredTxns

	for i := range allTxns {
		r.Apply(&allTxns[i])
	}

	return ldg.AddTransactions(allTxns)
}

func File(ldg *ledger.Ledger, fileName string) error {
	err := ioutil.WriteFile(fileName, []byte(ldg.String()), os.ModePerm)
	return errors.Wrap(err, "Error writing ledger to disk")
}

func downloadTxns(accounts []client.Account) func(start, end time.Time) ([]ledger.Transaction, error) {
	return func(start, end time.Time) ([]ledger.Transaction, error) {
		var txns []ledger.Transaction
		for _, account := range accounts {
			accountTxns, err := client.Transactions(account, start, end)
			if err != nil {
				return nil, err
			}
			txns = append(txns, accountTxns...)
		}
		return txns, nil
	}
}

func min(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
