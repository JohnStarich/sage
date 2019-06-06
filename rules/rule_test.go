package rules

import (
	"testing"
	"time"

	"github.com/johnstarich/sage/ledger"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireRule(rule Rule, err error) Rule {
	if err != nil {
		panic(err)
	}
	return rule
}

func TestRulesApply(t *testing.T) {
	rules := Rules{
		requireRule(NewCSVRule(
			"", "revenue", "",
			`,[0-9\.]+,`,
		)),
		requireRule(NewCSVRule(
			"", "expenses", "",
			`,-[0-9\.]+,`,
		)),
		requireRule(NewCSVRule(
			"cool bank", "", "",
			`"cool place to eat"`,
		)),
		requireRule(NewCSVRule(
			"", "", "hey there!",
			"2019/01/02",
		)),
	}
	someDate, err := time.Parse(ledger.DateFormat, "2019/01/02")
	require.NoError(t, err)

	for _, tc := range []struct {
		description string
		txn         ledger.Transaction
		expectedTxn ledger.Transaction
	}{
		{
			description: "revenue",
			txn: ledger.Transaction{
				Postings: []ledger.Posting{
					{Account: "assets:my bank", Amount: decimal.NewFromFloat(1.25)},
					{Amount: decimal.NewFromFloat(-1.25)},
				},
			},
			expectedTxn: ledger.Transaction{
				Postings: []ledger.Posting{
					{Account: "assets:my bank", Amount: decimal.NewFromFloat(1.25)},
					{Account: "revenue", Amount: decimal.NewFromFloat(-1.25)},
				},
			},
		},
		{
			description: "expenses",
			txn: ledger.Transaction{
				Postings: []ledger.Posting{
					{Account: "assets:my bank", Amount: decimal.NewFromFloat(-1.25)},
					{Amount: decimal.NewFromFloat(1.25)},
				},
			},
			expectedTxn: ledger.Transaction{
				Postings: []ledger.Posting{
					{Account: "assets:my bank", Amount: decimal.NewFromFloat(-1.25)},
					{Account: "expenses", Amount: decimal.NewFromFloat(1.25)},
				},
			},
		},
		{
			description: "account1",
			txn: ledger.Transaction{
				Payee: "cool place to eat",
				Postings: []ledger.Posting{
					{Account: "assets:my bank", Amount: decimal.NewFromFloat(-1.25)},
					{Amount: decimal.NewFromFloat(1.25)},
				},
			},
			expectedTxn: ledger.Transaction{
				Payee: "cool place to eat",
				Postings: []ledger.Posting{
					{Account: "cool bank", Amount: decimal.NewFromFloat(-1.25)},
					{Account: "expenses", Amount: decimal.NewFromFloat(1.25)},
				},
			},
		},
		{
			description: "comment",
			txn: ledger.Transaction{
				Date: someDate,
				Postings: []ledger.Posting{
					{Account: "assets:my bank", Amount: decimal.NewFromFloat(-1.25)},
					{Amount: decimal.NewFromFloat(1.25)},
				},
			},
			expectedTxn: ledger.Transaction{
				Date: someDate,
				Postings: []ledger.Posting{
					{Account: "assets:my bank", Amount: decimal.NewFromFloat(-1.25), Comment: "hey there!"},
					{Account: "expenses", Amount: decimal.NewFromFloat(1.25)},
				},
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			rules.Apply(&tc.txn)
			assert.Equal(t, tc.expectedTxn, tc.txn)
		})
	}
}

func TestRulesString(t *testing.T) {
	rules := Rules{
		requireRule(NewCSVRule("some account 1", "", "")),
		requireRule(NewCSVRule("", "", "some comment", "hey there")),
	}
	assert.Equal(t, `account1 some account 1

if
hey there
  comment some comment

`, rules.String())
}
