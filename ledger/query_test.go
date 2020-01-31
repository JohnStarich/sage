package ledger

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuery(t *testing.T) {
	for _, tc := range []struct {
		description   string
		txns          []Transaction
		options       QueryOptions
		page, results int
		expect        QueryResult
	}{
		{
			description: "no txns",
			page:        1,
			results:     1,
			expect: QueryResult{
				Page:         1,
				Results:      1,
				Transactions: []Transaction{},
			},
		},
		{
			description: "search one txn",
			txns: []Transaction{
				{Payee: "hello there"},
				{Payee: "hi there"},
			},
			options: QueryOptions{Search: "hello"},
			page:    1,
			results: 10,
			expect: QueryResult{
				Count:        1,
				Page:         1,
				Results:      10,
				Transactions: []Transaction{{Payee: "hello there"}},
			},
		},
		{
			description: "paginate search",
			txns: []Transaction{
				{
					Payee: "Opening balance",
					Postings: []Posting{
						{Account: "equity:Opening Balances", Tags: makeIDTag(OpeningBalanceID)},
					},
				},
				{Payee: "hello there"},
				{Payee: "hi there"},
			},
			options: QueryOptions{Search: "there"},
			page:    1,
			results: 1,
			expect: QueryResult{
				Count:   2,
				Page:    1,
				Results: 1,
				Transactions: []Transaction{
					{Payee: "hi there"}, // sorted results means 'hi' is last (we're paginating from the end backwards)
				},
			},
		},
		{
			description: "filter dates",
			txns: []Transaction{
				{Date: parseDate(t, "2020/01/01"), Payee: "hello there"},
				{Date: parseDate(t, "2020/01/02"), Payee: "hi there"},
				{Date: parseDate(t, "2020/01/03"), Payee: "goodbye"},
				{Date: parseDate(t, "2020/01/04"), Payee: "see ya later"},
			},
			options: QueryOptions{
				Start: parseDate(t, "2020/01/02"),
				End:   parseDate(t, "2020/01/03"),
			},
			page:    1,
			results: 10,
			expect: QueryResult{
				Count:   2,
				Page:    1,
				Results: 10,
				Transactions: []Transaction{
					{Date: parseDate(t, "2020/01/02"), Payee: "hi there"},
					{Date: parseDate(t, "2020/01/03"), Payee: "goodbye"},
				},
			},
		},
		{
			description: "filter accounts",
			txns: []Transaction{
				{Postings: []Posting{{}, {Account: "uncategorized"}}},
				{Postings: []Posting{{}, {Account: "expenses:uncategorized"}}},
				{Postings: []Posting{{}, {Account: "some real category"}}},
			},
			options: QueryOptions{
				Accounts: []string{"uncategorized", "expenses:uncategorized"},
			},
			page:    1,
			results: 10,
			expect: QueryResult{
				Count:   2,
				Page:    1,
				Results: 10,
				Transactions: []Transaction{
					{Postings: []Posting{{}, {Account: "uncategorized"}}},
					{Postings: []Posting{{}, {Account: "expenses:uncategorized"}}},
				},
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			ldg, err := New(tc.txns)
			require.NoError(t, err)
			assert.Equal(t, tc.expect, ldg.Query(tc.options, tc.page, tc.results))
		})
	}
}

func TestPaginateFromEnd(t *testing.T) {
	for _, tc := range []struct {
		page, results, size int
		start, end          int
	}{
		{page: 1, results: 10, size: 0, start: 0, end: 0},
		{page: 1, results: 10, size: 10, start: 0, end: 10},
		{page: 1, results: 5, size: 10, start: 5, end: 10},
		{page: 2, results: 5, size: 10, start: 0, end: 5},

		{page: 1, results: 10, size: 5, start: 0, end: 5},
		{page: 2, results: 2, size: 3, start: 0, end: 1},

		{page: 3, results: 1, size: 1, start: 0, end: 0},
	} {
		t.Run(fmt.Sprintf("page %d results %d size %d", tc.page, tc.results, tc.size), func(t *testing.T) {
			start, end := paginateFromEnd(tc.page, tc.results, tc.size)
			require.Truef(t, tc.start >= 0, "Test case start must be greater than or equal to 0: %d", tc.start)
			require.Truef(t, tc.end >= tc.start, "Test case end must be greater than or equal to start: end=%d, start=%d", tc.end, tc.start)
			require.Truef(t, tc.end <= tc.size, "Test case end must be less than or equal to size: end=%d, size=%d", tc.end, tc.size)

			assert.Equal(t, tc.start, start, "Incorrect start")
			assert.Equal(t, tc.end, end, "Incorrect end")
		})
	}
}
