package sync

import (
	"time"

	"github.com/johnstarich/sage/client"
	"github.com/johnstarich/sage/client/direct"
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/client/web"
	sErrors "github.com/johnstarich/sage/errors"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/rules"
	"github.com/pkg/errors"
)

// Sync fetches transactions for each account and categorizes them based on rules, then writes them to disk
func Sync(ldgStore *ledger.Store, accountStore *client.AccountStore, rulesStore *rules.Store, syncFromLedgerStart bool) {
	download := downloadTxns(accountStore)
	if syncFromLedgerStart {
		ldgStore.Resync(download, rulesStore.ApplyAll)
	} else {
		ldgStore.SyncRecent(download, rulesStore.ApplyAll)
	}
}

func downloadTxns(accountStore *client.AccountStore) func(start, end time.Time) ([]ledger.Transaction, error) {
	return func(start, end time.Time) ([]ledger.Transaction, error) {
		instMap := make(map[model.Institution][]model.Account)
		var account model.Account
		err := accountStore.Iter(&account, func(id string) bool {
			inst := account.Institution()
			instMap[inst] = append(instMap[inst], account)
			return true
		})
		if err != nil {
			return nil, err
		}
		var allTxns []ledger.Transaction
		var errs sErrors.Errors
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
				txns, err := direct.Statement(connector, start, end, requestors, client.ParseOFX)
				errs.AddErr(errors.Wrapf(err, "Failed downloading transactions: %s", descriptions))
				allTxns = append(allTxns, txns...)
			}
			if connector, isConn := inst.(web.Connector); isConn {
				var descriptions []string
				var accountIDs []string
				for _, account := range accounts {
					accountIDs = append(accountIDs, account.ID())
					descriptions = append(descriptions, account.Description())
				}
				txns, err := web.Statement(connector, start, end, accountIDs, client.ParseOFX)
				if !errs.AddErr(errors.Wrapf(err, "Failed downloading transactions: %s", descriptions)) {
					// TODO remove break after beta
					break // beta: fail immediately on web connector error
				}
				allTxns = append(allTxns, txns...)
			}
		}
		return allTxns, errs.ErrOrNil()
	}
}
