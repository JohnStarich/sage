package ledger

import (
	"sort"

	"github.com/johnstarich/sage/math"
)

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
func (l *Ledger) Query(search string, page, results int) QueryResult {
	if page < 1 || results < 1 {
		panic("Page and results must >= 1")
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	txns := l.transactions
	if len(txns) > 0 {
		for _, p := range l.transactions[0].Postings {
			if p.IsOpeningBalance() {
				txns = l.transactions[1:]
				break
			}
		}
	}

	size := len(txns)
	if search != "" {
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
		searchTxns := make([]*Transaction, 0, results)
		start, end := paginateFromEnd(page, results, size)
		for _, score := range txnScores[start:end] {
			if len(searchTxns) == results {
				break
			}
			searchTxns = append(searchTxns, score.Transaction)
		}
		txns = searchTxns
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
