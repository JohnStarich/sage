package rules

import (
	"encoding/json"
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

func TestUnmarshalJSON(t *testing.T) {
	var r Rules
	err := r.UnmarshalJSON([]byte("not JSON"))
	assert.Error(t, err)

	err = json.Unmarshal([]byte(`
	[
		{"Conditions": ["burgers"], "Account2": "some expenses"}
	]
	`), &r)
	assert.NoError(t, err)
	assert.Equal(t, Rules{
		requireRule(NewCSVRule("", "some expenses", "", "burgers")),
	}, r)
}

func TestMatches(t *testing.T) {
	r := Rules{
		requireRule(NewCSVRule("", "some expenses", "", "burgers")),
		requireRule(NewCSVRule("", "some sandwich expenses", "", "sandwiches")),
		requireRule(NewCSVRule("", "some expenses", "", "more burgers")),
	}
	results := r.Matches(&ledger.Transaction{
		Payee:    "more burgers",
		Postings: []ledger.Posting{{}, {}},
	})
	assert.Equal(t, map[int]Rule{
		0: r[0],
		2: r[2],
	}, results)
}
