package sync

import (
	"fmt"
	"time"

	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/rules"
	"github.com/pkg/errors"
)

const (
	days = 24 * time.Hour
)

func Sync(ldg *ledger.Ledger, accounts []client.Account, r rules.Rules) error {
	return sync(ldg, r, downloadTxns(accounts))
}

func sync(ldg *ledger.Ledger, r rules.Rules, download func(start, end time.Time) ([]ledger.Transaction, error)) error {
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
		{
			fmt.Printf("Downloading txns... (%s - %s)\n", downloadedTime, end)
		}
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
