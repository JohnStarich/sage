package ledger

import (
	"sort"

	"github.com/johnstarich/sage/math"
)

type QueryResult struct {
	Count   int
	Page    int
	Results int
	Transactions
}

type transactionScore struct {
	Score int
	Transaction
}

func (l *Ledger) Query(search string, page, results int) QueryResult {
	txns := l.transactions
	if page < 1 || results < 1 {
		panic("Page and results must >= 1")
	}
	if len(txns) > 0 {
		for _, p := range l.transactions[0].Postings {
			if p.isOpeningBalance() {
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
		searchTxns := make([]Transaction, 0, results)
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
		Transactions: txns,
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
