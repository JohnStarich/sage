package rules

import (
	"regexp"
	"testing"

	"github.com/johnstarich/sage/ledger"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCategoryMatch(t *testing.T) {
	for _, tc := range []struct {
		description string
		category    category
		txn         ledger.Transaction
		expectMatch bool
	}{
		{
			description: "catch-all",
			category: category{
				Zero:     true,
				Positive: true,
				Negative: true,
			},
			txn: ledger.Transaction{
				Postings: []ledger.Posting{{}},
			},
			expectMatch: true,
		},
		{
			description: "payee regex match",
			category: category{
				PayeeContains: regexp.MustCompile(`hi there`),
				Zero:          true,
			},
			txn: ledger.Transaction{
				Payee:    "why, hi there",
				Postings: []ledger.Posting{{}},
			},
			expectMatch: true,
		},
		{
			description: "payee regex NOT match",
			category: category{
				PayeeContains: regexp.MustCompile(`hi there`),
				Zero:          true,
			},
			txn: ledger.Transaction{
				Payee:    "why, hello there",
				Postings: []ledger.Posting{{}},
			},
			expectMatch: false,
		},
		{
			description: "amount is zero",
			category: category{
				Zero: true,
			},
			txn: ledger.Transaction{
				Postings: []ledger.Posting{{}},
			},
			expectMatch: true,
		},
		{
			description: "amount is NOT zero",
			category: category{
				Positive: true,
				Negative: true,
			},
			txn: ledger.Transaction{
				Postings: []ledger.Posting{{}},
			},
			expectMatch: false,
		},
		{
			description: "amount is positive",
			category: category{
				Positive: true,
			},
			txn: ledger.Transaction{
				Postings: []ledger.Posting{{Amount: decimal.New(1, 1)}},
			},
			expectMatch: true,
		},
		{
			description: "amount is NOT positive",
			category: category{
				Zero:     true,
				Negative: true,
			},
			txn: ledger.Transaction{
				Postings: []ledger.Posting{{Amount: decimal.New(1, 1)}},
			},
			expectMatch: false,
		},
		{
			description: "amount is negative",
			category: category{
				Negative: true,
			},
			txn: ledger.Transaction{
				Postings: []ledger.Posting{{Amount: decimal.New(-1, 1)}},
			},
			expectMatch: true,
		},
		{
			description: "amount is NOT negative",
			category: category{
				Zero:     true,
				Positive: true,
			},
			txn: ledger.Transaction{
				Postings: []ledger.Posting{{Amount: decimal.New(-1, 1)}},
			},
			expectMatch: false,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expectMatch, tc.category.Match(tc.txn))
		})
	}
}

func TestApply(t *testing.T) {
	txn := ledger.Transaction{
		Postings: []ledger.Posting{
			{},
		},
	}
	category{Category: "some category"}.Apply(&txn)
	assert.Equal(t, "some category", txn.Postings[0].Account)
}
