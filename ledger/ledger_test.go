package ledger

import (
	"bytes"
	"strings"
	"testing"
	"time"

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
			transactions: []Transaction{},
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
				require.Error(t, err)
				assert.Equal(t, duplicateTransactionError(tc.duplicateID).Error(), err.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.transactions, dereferenceTransactions(ldg.transactions))
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
		txn := &Transaction{
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
		}
		assert.Equal(t, &Ledger{
			transactions: Transactions{txn},
			idSet:        map[string]*Transaction{"A": txn, "B": txn, "C": txn},
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
					{Account: "equity:Opening Balances", Amount: *decFloat(-2.25), Tags: makeIDTag("Opening-Balance")},
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
					{Account: "equity:Opening Balances", Amount: *decFloat(-1.25), Tags: makeIDTag("Opening-Balance")},
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
					{Account: "equity:Opening Balances", Amount: *decFloat(1.25), Tags: makeIDTag("Opening-Balance")},
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
					{Account: "equity:Opening Balances", Amount: *decFloat(-13), Tags: makeIDTag("Opening-Balance")},
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
					{Account: "revenues", Amount: *decFloat(-1)},
				}},
				{Postings: []Posting{
					{Account: "account 3", Amount: *decFloat(-3), Balance: decFloat(-1.50)},
					{Account: "revenues", Amount: *decFloat(3)},
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
					{Account: "equity:Opening Balances", Amount: *decFloat(-13), Tags: makeIDTag("Opening-Balance")},
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
					{Account: "equity:Opening Balances", Amount: *decFloat(-1), Tags: makeIDTag("Opening-Balance")},
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

func TestLastTransactionTime(t *testing.T) {
	now := time.Now()
	ldg, err := New([]Transaction{
		{Payee: "some payee", Date: now.Add(-1 * time.Hour)},
		{Payee: "some other payee", Date: now},
	})
	require.NoError(t, err)
	assert.Equal(t, now, ldg.LastTransactionTime())

	ldg, err = New(nil)
	require.NoError(t, err)
	assert.Zero(t, ldg.LastTransactionTime())
}

func TestAddTransactions(t *testing.T) {
	somePostings := []Posting{
		{Account: "some bank"},
		{Account: "some business"},
	}
	txn1 := &Transaction{Payee: "woot woot", Postings: somePostings, Tags: makeIDTag("a")}
	txn2 := &Transaction{Payee: "the dough", Postings: somePostings, Tags: makeIDTag("b")}
	brokenTxn := &Transaction{Payee: "broken transaction", Postings: nil, Tags: makeIDTag("c")}
	for _, tc := range []struct {
		description  string
		txns         Transactions
		newTxns      Transactions
		expectedTxns Transactions
		expectedErr  bool
	}{
		{description: "no transactions"},
		{
			description:  "add to empty ledger",
			newTxns:      Transactions{txn1, txn2},
			expectedTxns: Transactions{txn1, txn2},
		},
		{
			description:  "ignore duplicates from old to new txns",
			txns:         Transactions{txn1},
			newTxns:      Transactions{txn1},
			expectedTxns: Transactions{txn1},
		},
		{
			description:  "ignore duplicates in new txns",
			txns:         Transactions{txn1},
			newTxns:      Transactions{txn1, txn1},
			expectedTxns: Transactions{txn1},
		},
		{
			description:  "validate new transactions before adding",
			txns:         Transactions{txn1},
			newTxns:      Transactions{brokenTxn},
			expectedTxns: Transactions{txn1},
			expectedErr:  true,
		},
		{
			description:  "add txns up to first failure",
			txns:         Transactions{txn1},
			newTxns:      Transactions{txn2, brokenTxn},
			expectedTxns: Transactions{txn1, txn2},
			expectedErr:  true,
		},
		{
			description:  "no validate error if txns started invalid",
			txns:         Transactions{brokenTxn},
			newTxns:      Transactions{},
			expectedTxns: Transactions{brokenTxn},
			expectedErr:  true,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			ldg, err := New(dereferenceTransactions(tc.txns))
			require.NoError(t, err)

			err = ldg.AddTransactions(dereferenceTransactions(tc.newTxns))
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedTxns, ldg.transactions)
			idSet, _, _ := makeIDSet(ldg.transactions)
			assert.Equal(t, idSet, ldg.idSet)
		})
	}
}
