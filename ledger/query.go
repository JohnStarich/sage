package ledger

import (
	"sort"
	"time"

	"github.com/johnstarich/sage/math"
)

// QueryOptions contains all available options to query the ledger
type QueryOptions struct {
	Search   string    `form:"search"`
	Start    time.Time `form:"start"`
	End      time.Time `form:"end"`
	Accounts []string  `form:"accounts[]"`
}

// QueryResult is a paginated search result containing relevant transactions
type QueryResult struct {
	Count        int
	Page         int
	Results      int
	Transactions []Transaction
}

type transactionScore struct {
	Score int
	*Transaction
}

// Query searches the ledger and paginates the results
func (l *Ledger) Query(options QueryOptions, page, results int) QueryResult {
	if page < 1 || results < 1 {
		panic("Page and results must >= 1")
	}
	if options.End.IsZero() {
		// default End time is in the future
		options.End = time.Now().AddDate(0, 0, 1)
	}

	l.mu.RLock()
	defer l.mu.RUnlock()
	txns := make(Transactions, 0, len(l.transactions))
	if openingBalTxn := l.idSet[OpeningBalanceID]; openingBalTxn != nil {
		// skip opening balance for queries
		for _, txn := range l.transactions {
			if txn != openingBalTxn && matchesOptions(txn, options) {
				txns = append(txns, txn)
			}
		}
	} else {
		for _, txn := range l.transactions {
			if matchesOptions(txn, options) {
				txns = append(txns, txn)
			}
		}
	}

	size := len(txns)
	if options.Search != "" {
		txns, size = searchTxns(options.Search, txns, page, results)
	} else {
		start, end := paginateFromEnd(page, results, size)
		txns = txns[start:end]
	}

	return QueryResult{
		Count:        size,
		Page:         page,
		Results:      results,
		Transactions: dereferenceTransactions(txns),
	}
}

func matchesOptions(txn *Transaction, options QueryOptions) bool {
	if txn.Date.Before(options.Start) || txn.Date.After(options.End) {
		return false
	}
	if len(options.Accounts) > 0 {
		found := false
		txnAccount := txn.Postings[len(txn.Postings)-1].Account
		for _, account := range options.Accounts {
			if account == txnAccount {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// assumes all parameters are > 0
func paginateFromEnd(page, results, size int) (start, end int) {
	if size == 0 {
		return
	}

	start = math.MaxInt(size-page*results, 0)
	end = math.MinInt(size-(page-1)*results, size)
	end = math.MaxInt(end, 0)
	return
}

func searchTxns(search string, txns Transactions, page, results int) (searchTxns Transactions, size int) {
	// this search is wildly inefficient, hopefully we move onto a DB with proper indexes

	// map initial txn index to score
	txnScores := make([]transactionScore, 0, len(txns))
	for _, txn := range txns {
		score := txn.matches(search)
		if score > 0 {
			txnScores = append(txnScores, transactionScore{
				Score:       score,
				Transaction: txn,
			})
		}
	}
	size = len(txnScores)
	sort.Slice(txnScores, func(a, b int) bool {
		return txnScores[a].Score < txnScores[b].Score
	})
	searchTxns = make([]*Transaction, 0, results)
	start, end := paginateFromEnd(page, results, size)
	for _, score := range txnScores[start:end] {
		if len(searchTxns) == results {
			break
		}
		searchTxns = append(searchTxns, score.Transaction)
	}
	return searchTxns, size
}
