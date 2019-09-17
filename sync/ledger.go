package sync

import (
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/client/direct"
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/rules"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	days = 24 * time.Hour
)

var (
	ledgerMu sync.Mutex // basic protection against concurrent sync operations
)

// Sync runs a Ledger sync, followed by a LedgerFile sync
// If a partial failure occurs during Ledger sync, runs LedgerFile sync anyway
func Sync(logger *zap.Logger, ledgerFileName string, ldg *ledger.Ledger, accountStore *client.AccountStore, rulesStore *rules.Store, syncFromLedgerStart bool) error {
	ledgerErr := Ledger(logger, ldg, accountStore, rulesStore, syncFromLedgerStart)
	if ledgerErr != nil {
		if _, ok := ledgerErr.(ledger.Error); !ok {
			return ledgerErr
		}
	}
	if err := LedgerFile(ldg, ledgerFileName); err != nil {
		return err
	}
	return ledgerErr
}

// Ledger fetches transactions for each account and categorizes them based on rules
func Ledger(logger *zap.Logger, ldg *ledger.Ledger, accountStore *client.AccountStore, rulesStore *rules.Store, syncFromLedgerStart bool) error {
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	return ledgerSync(logger, ldg, rulesStore, downloadTxns(accountStore), syncFromLedgerStart)
}

func ledgerSync(
	logger *zap.Logger,
	ldg *ledger.Ledger,
	rulesStore *rules.Store,
	download func(start, end time.Time) ([]ledger.Transaction, error),
	syncFromLedgerStart bool,
) error {
	if err := ldg.Validate(); err != nil {
		return errors.Wrap(err, "Existing ledger is not valid")
	}

	now := time.Now()
	// TODO use smart first date selection on a per-account basis
	const syncBuffer = 2 * days
	var lastTxnTime time.Time
	if syncFromLedgerStart {
		lastTxnTime = ldg.FirstTransactionTime()
	} else {
		lastTxnTime = ldg.LastTransactionTime()
	}
	if lastTxnTime.IsZero() {
		lastTxnTime = now.Add(-30 * days)
	}
	duration := now.Sub(lastTxnTime)
	duration += syncBuffer

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
	logger.Info("Download succeeded!")

	// throw out extra transactions that were included by the institution responses
	filteredTxns := make([]ledger.Transaction, 0, len(allTxns))
	for _, t := range allTxns {
		if t.Date.Before(lastTxnTime) {
			continue
		}
		filteredTxns = append(filteredTxns, t)
	}
	allTxns = filteredTxns

	rulesStore.ApplyAll(allTxns)

	if err := ldg.AddTransactions(allTxns); err != nil {
		logger.Warn("Failed to add transactions to ledger", zap.Error(err))
		return err
	}
	logger.Info("Ledger successfully updated")
	return nil
}

// LedgerFile writes the given ledger to disk in "ledger" format
func LedgerFile(ldg *ledger.Ledger, fileName string) error {
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	err := ioutil.WriteFile(fileName, []byte(ldg.String()), os.ModePerm)
	return errors.Wrap(err, "Error writing ledger to disk")
}

func downloadTxns(accountStore *client.AccountStore) func(start, end time.Time) ([]ledger.Transaction, error) {
	return func(start, end time.Time) ([]ledger.Transaction, error) {
		instMap := make(map[model.Institution][]model.Account)
		accountStore.Iterate(func(account model.Account) bool {
			inst := account.Institution()
			instMap[inst] = append(instMap[inst], account)
			return true
		})
		var allTxns []ledger.Transaction
		for inst, accounts := range instMap {
			if connector, isConn := inst.(direct.Connector); isConn {
				var descriptions []string
				var requestors []direct.Requestor
				for _, account := range accounts {
					if requestor, isRequestor := account.(direct.Requestor); isRequestor {
						requestors = append(requestors, requestor)
						descriptions = append(descriptions, account.Description())
					}
				}
				txns, err := direct.Statement(connector, start, end, requestors)
				if err != nil {
					return nil, errors.Wrapf(err, "Failed downloading transactions: %s", descriptions)
				}
				allTxns = append(allTxns, txns...)
			}
		}
		return allTxns, nil
	}
}

func min(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
