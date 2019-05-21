package ledger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeIDTag(s string) map[string]string {
	return map[string]string{idTag: s}
}

func TestNew(t *testing.T) {
	for _, tc := range []struct {
		description  string
		transactions []Transaction
		duplicateID  string
	}{
		{
			description: "happy path",
			transactions: []Transaction{
				{
					Tags: makeIDTag("1"), Postings: []Posting{
						{Tags: makeIDTag("2")},
						{Tags: makeIDTag("3")},
					},
				},
				{
					Tags: makeIDTag("4"), Postings: []Posting{
						{Tags: makeIDTag("5")},
						{Tags: makeIDTag("6")},
					},
				},
			},
		},
		{
			description:  "no transactions",
			transactions: nil,
		},
		{
			description: "duplicate transaction IDs",
			transactions: []Transaction{
				{Tags: makeIDTag("1")},
				{Tags: makeIDTag("1")},
			},
			duplicateID: "1",
		},
		{
			description: "duplicate transaction/posting IDs",
			transactions: []Transaction{
				{Tags: makeIDTag("1")},
				{Tags: makeIDTag("2"), Postings: []Posting{
					{Tags: makeIDTag("2")},
				}},
			},
			duplicateID: "2",
		},
		{
			description: "duplicate posting IDs",
			transactions: []Transaction{
				{Tags: makeIDTag("1")},
				{Tags: makeIDTag("2"), Postings: []Posting{
					{Tags: makeIDTag("3")},
					{Tags: makeIDTag("3")},
				}},
			},
			duplicateID: "3",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			ldg, err := New(tc.transactions)
			if tc.duplicateID != "" {
				assert.Error(t, err)
				assert.Equal(t, duplicateTransactionError(tc.duplicateID), err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.transactions, ldg.transactions)
		})
	}
}

func TestNewFromReader(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		buf := bytes.NewBufferString(`
2019/01/02 some burger place ; id: A
	expenses:food   $ 1.25 ; id: B
	assets:Bank 1 ; id: C
		`)
		ldg, err := NewFromReader(buf)
		require.NoError(t, err)
		assert.Equal(t, &Ledger{
			transactions: []Transaction{
				{
					Date:  parseDate(t, "2019/01/02"),
					Payee: "some burger place",
					Tags:  makeIDTag("A"),
					Postings: []Posting{
						{
							Account:  "expenses:food",
							Amount:   *decFloat(1.25),
							Currency: usd,
							Tags:     makeIDTag("B"),
						},
						{
							Account:  "assets:Bank 1",
							Amount:   *decFloat(-1.25),
							Currency: usd,
							Tags:     makeIDTag("C"),
						},
					},
				},
			},
			idSet: map[string]bool{"A": true, "B": true, "C": true},
		}, ldg)
	})

	t.Run("bad transaction", func(t *testing.T) {
		buf := bytes.NewBufferString(`
2019/01/02 some burger place
	expenses:food   $ 1.25
		`)
		_, err := NewFromReader(buf)
		assert.Error(t, err)
	})
}

func TestLedgerString(t *testing.T) {
	ldg, err := New([]Transaction{
		{
			Date:  parseDate(t, "2019/01/02"),
			Payee: "some burger place",
			Tags:  makeIDTag("A"),
			Postings: []Posting{
				{
					Account:  "expenses:food",
					Amount:   *decFloat(1.25),
					Currency: usd,
					Tags:     makeIDTag("B"),
				},
				{
					Account:  "assets:Bank 1",
					Amount:   *decFloat(-1.25),
					Currency: usd,
					Tags:     makeIDTag("C"),
				},
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, strings.TrimSpace(`
2019/01/02 some burger place ; id: A
    expenses:food   $ 1.25 ; id: B
    assets:Bank 1  $ -1.25 ; id: C
	`)+"\n\n", ldg.String())
}

func TestLedgerValidate(t *testing.T) {
	for _, tc := range []struct {
		description string
		txns        []Transaction
		expectedErr string
	}{
		{
			description: "zero txns",
			txns:        []Transaction{},
		},
		{
			description: "one valid txn",
			txns: []Transaction{
				{Postings: []Posting{
					{Amount: *decFloat(1.25)},
					{Amount: *decFloat(-1.25)},
				}},
			},
		},
		{
			description: "opening balance account and one txn valid",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(2.25)},
					{Account: "equity:Opening Balances", Amount: *decFloat(-2.25)},
				}},
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-1.25), Balance: decFloat(1)},
					{Account: "expenses", Amount: *decFloat(1.25)},
				}},
			},
		},
		{
			description: "opening balance ID and one txn valid",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(2.25)},
					{Account: "equity:open", Amount: *decFloat(-2.25), Tags: makeIDTag("Opening-Balance")},
				}},
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-1.25), Balance: decFloat(1)},
					{Account: "expenses", Amount: *decFloat(1.25)},
				}},
			},
		},
		{
			description: "opening balance and one txn invalid",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(2.25)},
					{Account: "equity:Opening Balances", Amount: *decFloat(-2.25)},
				}},
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-1.25), Balance: decFloat(-5)},
					{Account: "expenses", Amount: *decFloat(1.25)},
				}},
			},
			expectedErr: "Failed balance assertion for account 'account 1': difference = 6",
		},
		{
			description: "invalid opening balance txn",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(2.25)},
					{Account: "equity:Opening Balances", Amount: *decFloat(-1.25)},
				}},
			},
			expectedErr: "Transaction is not balanced - postings do not sum to zero:",
		},
		{
			description: "valid auto-opening balance with txn",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-1.25), Balance: decFloat(5)},
					{Account: "expenses", Amount: *decFloat(1.25)},
				}},
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-2.00), Balance: decFloat(3)},
					{Account: "expenses", Amount: *decFloat(2.00)},
				}},
			},
		},
		{
			description: "invalid auto-opening balance with txn",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-1.25), Balance: decFloat(5)},
					{Account: "expenses", Amount: *decFloat(1.25)},
				}},
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-2.00), Balance: decFloat(4)},
					{Account: "expenses", Amount: *decFloat(2.00)},
				}},
			},
			expectedErr: "Failed balance assertion for account 'account 1' (opening balances were auto-generated): difference = -1",
		},
		{
			description: "invalid auto-opening balance missing first balance",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-1.25)},
					{Account: "expenses", Amount: *decFloat(1.25)},
				}},
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-2.00), Balance: decFloat(2)},
					{Account: "expenses", Amount: *decFloat(2.00)},
				}},
			},
			expectedErr: "Failed balance assertion for account 'account 1' (opening balances were auto-generated): difference = -4",
		},
		{
			description: "invalid opening balance missing first balance",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "some other account", Amount: *decFloat(-1.25)},
					{Account: "equity:Opening Balances", Amount: *decFloat(1.25)},
				}},
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-2.00), Balance: decFloat(2)},
					{Account: "expenses", Amount: *decFloat(2.00)},
				}},
			},
			expectedErr: "Balance assertion found for account 'account 1', but no opening balance detected:",
		},
		{
			description: "valid ledger",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(5.25)},
					{Account: "account 2", Amount: *decFloat(5.25)},
					{Account: "account 3", Amount: *decFloat(2.50)},
					{Account: "equity:Opening Balances", Amount: *decFloat(-13)},
				}},
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-2.25), Balance: decFloat(3)},
					{Account: "expenses", Amount: *decFloat(2.25)},
				}},
				{Postings: []Posting{
					{Account: "account 2", Amount: *decFloat(-1), Balance: decFloat(4.25)},
					{Account: "account 3", Amount: *decFloat(-1), Balance: decFloat(1.50)},
					{Account: "expenses", Amount: *decFloat(2)},
				}},
				{Postings: []Posting{
					{Account: "account 2", Amount: *decFloat(-1), Balance: decFloat(3.25)},
					{Account: "expenses", Amount: *decFloat(1)},
				}},
				{Postings: []Posting{
					{Account: "account 2", Amount: *decFloat(1), Balance: decFloat(4.25)},
					{Account: "revenue", Amount: *decFloat(-1)},
				}},
				{Postings: []Posting{
					{Account: "account 3", Amount: *decFloat(-3), Balance: decFloat(-1.50)},
					{Account: "revenue", Amount: *decFloat(3)},
				}},
			},
		},
		{
			description: "invalid ledger - last txn balance",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(5.25)},
					{Account: "account 2", Amount: *decFloat(5.25)},
					{Account: "account 3", Amount: *decFloat(2.50)},
					{Account: "equity:Opening Balances", Amount: *decFloat(-13)},
				}},
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-2.25), Balance: decFloat(3)},
					{Account: "expenses", Amount: *decFloat(2.25)},
				}},
				{Postings: []Posting{
					{Account: "account 2", Amount: *decFloat(-1), Balance: decFloat(4.25)},
					{Account: "account 3", Amount: *decFloat(-1), Balance: decFloat(1.00)},
					{Account: "expenses", Amount: *decFloat(2)},
				}},
			},
			expectedErr: "Failed balance assertion for account 'account 3': difference = 0.5",
		},
		{
			description: "unbalanced txn",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(1)},
					{Account: "equity:Opening Balances", Amount: *decFloat(-1)},
				}},
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(-1.00), Balance: decFloat(0)},
					{Account: "expenses", Amount: *decFloat(2.00)},
				}},
			},
			expectedErr: "Transaction is not balanced - postings do not sum to zero:",
		},
		{
			description: "bad opening balance equity posting uses auto-open",
			txns: []Transaction{
				{Postings: []Posting{
					{Account: "account 1", Amount: *decFloat(1.25)},
					{Account: "equity:not an open", Amount: *decFloat(-1.25)},
				}},
				{Postings: []Posting{
					{Account: "account 2", Amount: *decFloat(-2.00), Balance: decFloat(2)}, // asserting account 2 after a bad "open" line for account 1 is fine. need to assume this is a custom notation
					{Account: "expenses", Amount: *decFloat(2.00)},
				}},
			},
			expectedErr: "",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			ldg, ldgErr := New(tc.txns)
			require.NoError(t, ldgErr)
			err := ldg.Validate()
			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
