package ledger

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	oneDay   = 24 * time.Hour
	oneMonth = 30 * oneDay
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
			description: "two valid txns",
			txns: []Transaction{
				{Postings: []Posting{
					{Amount: *decFloat(1.25)},
					{Amount: *decFloat(-1.25)},
				}},
				{Postings: []Posting{
					{Amount: *decFloat(2.50)},
					{Amount: *decFloat(-2.50)},
				}},
			},
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

func TestFirstTransactionTime(t *testing.T) {
	end := time.Now()
	start := end.Add(-1 * time.Hour)
	ldg, err := New([]Transaction{
		{Payee: "some payee", Date: start},
		{Payee: "some other payee", Date: end},
	})
	require.NoError(t, err)
	assert.Equal(t, start, ldg.FirstTransactionTime())

	ldg, err = New(nil)
	require.NoError(t, err)
	assert.Zero(t, ldg.FirstTransactionTime())
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

func TestUpdateOpeningBalance(t *testing.T) {
	makeOpeningBalance := func(amount float64) Posting {
		return Posting{
			Account:  "equity:Opening Balance",
			Amount:   *decFloat(amount),
			Currency: usd,
			Tags:     makeIDTag(OpeningBalanceID),
		}
	}
	makeOpeningTxn := func(date string, postings ...Posting) *Transaction {
		return &Transaction{
			Date:     parseDate(t, date),
			Payee:    "* Opening Balance",
			Postings: postings,
		}
	}
	makeAsset := func(name string, amount float64) Posting {
		return Posting{
			Account:  "assets:" + name,
			Amount:   *decFloat(amount),
			Currency: usd,
		}
	}
	makeExpenseTxn := func(date, name, assetName string, amount float64) *Transaction {
		return &Transaction{
			Date:  parseDate(t, date),
			Payee: name,
			Postings: []Posting{
				{
					Account:  "expenses:" + name,
					Amount:   *decFloat(amount),
					Currency: usd,
				},
				makeAsset(assetName, -amount),
			},
		}
	}
	for _, tc := range []struct {
		description  string
		txns         Transactions
		openingTxn   *Transaction
		expectedErr  bool
		expectedTxns Transactions
	}{
		{
			description: "no txns",
			openingTxn: makeOpeningTxn("2019/01/01",
				makeAsset("Bank 1", 1.25),
				makeOpeningBalance(-1.25),
			),
			expectedTxns: Transactions{
				makeOpeningTxn("2019/01/01",
					makeAsset("Bank 1", 1.25),
					makeOpeningBalance(-1.25),
				),
			},
		},
		{
			description: "identical opening txn",
			openingTxn: makeOpeningTxn("2019/01/01",
				makeAsset("Bank 1", 1.25),
				makeOpeningBalance(-1.25),
			),
			expectedTxns: Transactions{
				makeOpeningTxn("2019/01/01",
					makeAsset("Bank 1", 1.25),
					makeOpeningBalance(-1.25),
				),
			},
		},
		{
			description: "additional opening balance",
			openingTxn: makeOpeningTxn("2019/01/01",
				makeAsset("Bank 1", 1.25),
				makeAsset("Bank 2", 2.50),
				makeOpeningBalance(-3.75),
			),
			txns: Transactions{
				makeOpeningTxn("2019/01/01",
					makeAsset("Bank 2", 2.50),
					makeOpeningBalance(-2.50),
				),
			},
			expectedTxns: Transactions{
				makeOpeningTxn("2019/01/01",
					makeAsset("Bank 1", 1.25),
					makeAsset("Bank 2", 2.50),
					makeOpeningBalance(-3.75),
				),
			},
		},
		{
			description: "different date",
			openingTxn: makeOpeningTxn("2019/01/02",
				makeAsset("Bank 1", 1.25),
				makeOpeningBalance(-1.25),
			),
			txns: Transactions{
				makeOpeningTxn("2019/01/01",
					makeAsset("Bank 1", 1.25),
					makeOpeningBalance(-1.25),
				),
			},
			expectedTxns: Transactions{
				makeOpeningTxn("2019/01/02",
					makeAsset("Bank 1", 1.25),
					makeOpeningBalance(-1.25),
				),
			},
		},
		{
			description: "existing txns",
			openingTxn: makeOpeningTxn("2019/01/01",
				makeAsset("Bank 1", 2.00),
				makeOpeningBalance(-2.00),
			),
			txns: Transactions{
				makeExpenseTxn("2019/01/02", "Fast Food", "Bank 1", 1.00),
			},
			expectedTxns: Transactions{
				makeOpeningTxn("2019/01/01",
					makeAsset("Bank 1", 2.00),
					makeOpeningBalance(-2.00),
				),
				makeExpenseTxn("2019/01/02", "Fast Food", "Bank 1", 1.00),
			},
		},
		{
			description: "opening after txns",
			openingTxn: makeOpeningTxn("2019/01/02",
				makeAsset("Bank 1", 2.00),
				makeOpeningBalance(-2.00),
			),
			txns: Transactions{
				makeExpenseTxn("2019/01/01", "Fast Food", "Bank 1", 1.00),
			},
			expectedTxns: Transactions{
				makeExpenseTxn("2019/01/01", "Fast Food", "Bank 1", 1.00),
				makeOpeningTxn("2019/01/02",
					makeAsset("Bank 1", 2.00),
					makeOpeningBalance(-2.00),
				),
			},
		},
		{
			description: "update opening after txns",
			openingTxn: makeOpeningTxn("2019/01/01",
				makeAsset("Bank 1", 2.00),
				makeOpeningBalance(-2.00),
			),
			txns: Transactions{
				makeExpenseTxn("2019/01/02", "Fast Food", "Bank 1", 1.00),
				makeOpeningTxn("2019/01/03",
					makeAsset("Bank 1", 2.00),
					makeOpeningBalance(-2.00),
				),
			},
			expectedTxns: Transactions{
				makeOpeningTxn("2019/01/01",
					makeAsset("Bank 1", 2.00),
					makeOpeningBalance(-2.00),
				),
				makeExpenseTxn("2019/01/02", "Fast Food", "Bank 1", 1.00),
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			ldg, err := New(dereferenceTransactions(tc.txns))
			require.NoError(t, err)
			err = ldg.UpdateOpeningBalance(*tc.openingTxn)
			if tc.expectedErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTxns, ldg.transactions)
		})
	}
}

func TestUpdateTransaction(t *testing.T) {
	for _, tc := range []struct {
		description string
		txns        []Transaction
		id          string
		txn         Transaction
		expectErr   bool
	}{
		{
			description: "not found",
			id:          "non-existent",
			txn:         Transaction{Comment: "Something"},
			expectErr:   true,
		},
		{
			description: "first txn",
			txns: []Transaction{
				{
					Comment: "Other thing",
					Postings: []Posting{
						{Account: "assets:Super Bank:****1234", Tags: makeIDTag("some-txn")},
						{Account: "expenses:uncategorized"},
					},
				},
			},
			id: "some-txn",
			txn: Transaction{
				Comment: "Something",
				Postings: []Posting{
					{Account: "assets:Super Bank:****1234", Tags: makeIDTag("some-txn")},
					{Account: "expenses:travel"},
				},
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			ldg, err := New(tc.txns)
			require.NoError(t, err)

			err = ldg.UpdateTransaction(tc.id, tc.txn)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// look it up in idSet
			compareUpdate(t, *ldg.idSet[tc.id], tc.txn)
			found := false
			// look it up in transactions
			for _, txn := range ldg.transactions {
				if txn.Postings[0].ID() == tc.id {
					compareUpdate(t, *txn, tc.txn)
					assert.True(t, txn == ldg.idSet[tc.id], "The pointers must be identical")
					found = true
				}
			}
			assert.True(t, found)
		})
	}
}

func compareUpdate(t *testing.T, original, update Transaction) {
	if update.Payee == "" {
		original.Payee = ""
	}
	if update.Comment == "" {
		original.Comment = ""
	}
	if update.Date.IsZero() {
		original.Date = time.Time{}
	}
	if len(update.Postings) == 0 {
		update.Postings = nil
		original.Postings = nil
	}
	if len(update.Tags) == 0 {
		update.Tags = nil
		original.Tags = nil
	}
	assert.Equal(t, update, original)
}

func TestMakeIDSet(t *testing.T) {
	for _, tc := range []struct {
		description string
		txns        []Transaction
	}{
		{
			description: "one txn",
			txns: []Transaction{
				{
					Comment: "Other thing",
					Postings: []Posting{
						{Account: "assets:Super Bank:****1234", Tags: makeIDTag("some-txn")},
						{Account: "expenses:uncategorized"},
					},
				},
			},
		},
		{
			description: "two txns",
			txns: []Transaction{
				{
					Payee: "Other thing",
					Postings: []Posting{
						{Account: "assets:Super Bank:****1234", Tags: makeIDTag("some-txn")},
						{Account: "expenses:uncategorized"},
					},
				},
				{
					Payee: "the dough",
					Postings: []Posting{
						{Account: "assets:Super Bank:****1234", Tags: makeIDTag("dough-txn")},
						{Account: "expenses:uncategorized"},
					},
				},
			},
		},
		{
			description: "duplicate txns",
			txns: []Transaction{
				{
					Payee: "Other thing",
					Tags:  makeIDTag("some-txn"),
				},
				{
					Payee: "Other thing",
					Tags:  makeIDTag("some-txn"),
				},
			},
		},
		{
			description: "duplicate txn postings",
			txns: []Transaction{
				{
					Payee: "Other thing",
					Postings: []Posting{
						{Account: "assets:Super Bank:****1234", Tags: makeIDTag("some-txn")},
						{Account: "expenses:uncategorized"},
					},
				},
				{
					Payee: "Other thing",
					Postings: []Posting{
						{Account: "assets:Super Bank:****1234", Tags: makeIDTag("some-txn")},
						{Account: "expenses:uncategorized"},
					},
				},
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			txns := makeTransactionPtrs(tc.txns)
			idSet, uniqueTxns, _ := makeIDSet(txns)
			assert.Len(t, uniqueTxns, len(idSet))
			for _, txn := range uniqueTxns {
				id := txn.ID()
				if id == "" {
					id = txn.Postings[0].ID()
				}

				assert.True(t, txn == idSet[id], "Pointers must be identical to unique transactions\n%p != %p", txn, idSet[id])

				found := false
				for _, originalTxn := range txns {
					originalID := originalTxn.ID()
					if originalID == "" {
						originalID = originalTxn.Postings[0].ID()
					}

					if originalID == id {
						assert.True(t, originalTxn == txn, "First original txn match must be identical to unique txn pointer\n%p != %p", txn, originalTxn)
						found = true
						break
					}
				}
				assert.True(t, found, "First matching original transaction ptr must exist in uniqueTxns")
			}
		})
	}
}

func TestRenameAccount(t *testing.T) {
	makeTxn := func(account, fid, accountID, randomStr string) Transaction {
		return Transaction{
			Payee: "some payee",
			Postings: []Posting{
				{Account: account, Tags: makeIDTag(fid + "-" + accountID + "-" + randomStr)},
				{Account: "expenses"},
			},
		}
	}
	for _, tc := range []struct {
		description      string
		txns             []Transaction
		oldName, newName string
		oldID, newID     string
		expectCount      int
		expectTxns       []Transaction
	}{
		{
			description: "no txns",
		},
		{
			description: "change names",
			oldName:     "ye olde bank",
			newName:     "spiffy bank",
			txns: []Transaction{
				makeTxn("ye olde bank", "7101", "my-account", "1"),
				makeTxn("something else", "7101", "my-account", "2"),
			},
			expectCount: 1,
			expectTxns: []Transaction{
				makeTxn("spiffy bank", "7101", "my-account", "1"),
				makeTxn("something else", "7101", "my-account", "2"),
			},
		},
		{
			description: "change names and IDs",
			oldName:     "ye olde bank",
			newName:     "spiffy bank",
			oldID:       "7101-my-account-",
			newID:       "9625-my-other-account-",
			txns: []Transaction{
				makeTxn("ye olde bank", "7101", "my-account", "1"),
				makeTxn("something else", "7101", "my-account", "2"),
			},
			expectCount: 1,
			expectTxns: []Transaction{
				makeTxn("spiffy bank", "9625", "my-other-account", "1"),
				makeTxn("something else", "7101", "my-account", "2"),
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			ldg, err := New(tc.txns)
			require.NoError(t, err)

			count := ldg.RenameAccount(tc.oldName, tc.newName, tc.oldID, tc.newID)
			assert.Equal(t, tc.expectCount, count)
			expectLdg, err := New(tc.expectTxns)
			require.NoError(t, err)
			assert.Equal(t, expectLdg, ldg)
		})
	}
}

func TestBalances(t *testing.T) {
	var date time.Time
	makeTxn := func(account string, num float64, increment time.Duration) Transaction {
		date = date.Add(increment)
		return Transaction{
			Date:  date,
			Payee: "some payee",
			Postings: []Posting{
				{Account: "assets:something", Amount: *decFloat(-num)},
				{Account: account, Amount: *decFloat(num)},
			},
		}
	}
	ldg, err := New([]Transaction{
		makeTxn("ball", 1.0, oneDay),
		makeTxn("food:fish", 1.5, oneDay),
		makeTxn("food:taco", 3, oneDay),

		makeTxn("food:taco", 40, oneMonth),
		makeTxn("food:potato", 5, oneDay),
	})
	require.NoError(t, err)

	start, end, balances := ldg.Balances()
	assert.Equal(t, time.Time{}.AddDate(0, 0, 1), *start)
	assert.Equal(t, time.Time{}.AddDate(0, 1, 3), *end)

	floatBalances := make(map[string][]float64, len(balances))
	for key, values := range balances {
		for _, value := range values {
			floatValue, exact := value.Float64()
			require.True(t, exact)
			floatBalances[key] = append(floatBalances[key], floatValue)
		}
	}
	assert.Equal(t, map[string][]float64{
		"ball":             {1, 1},
		"food:fish":        {1.5, 1.5},
		"food:taco":        {3, 43},
		"food:potato":      {0, 5},
		"assets:something": {-5.5, -50.5},
	}, floatBalances)
}

func TestAccountBalance(t *testing.T) {
	var date time.Time
	makeTxn := func(account string, num float64, increment time.Duration) Transaction {
		date = date.Add(increment)
		return Transaction{
			Date:  date,
			Payee: "some payee",
			Postings: []Posting{
				{Account: "assets:some bank", Amount: *decFloat(-num)},
				{Account: account, Amount: *decFloat(num)},
			},
		}
	}
	ldg, err := New([]Transaction{
		makeTxn("food:Groceries", 10, oneDay),

		makeTxn("gas", 25, oneMonth),
		makeTxn("food:Restaurants", 20, oneDay),
		makeTxn("food", 5, oneDay),
	})
	require.NoError(t, err)

	// all txns
	balDecimal := ldg.AccountBalance("food", time.Time{}, date)
	bal, exact := balDecimal.Float64()
	require.True(t, exact)
	assert.EqualValues(t, 35, bal)

	// skip first day
	balDecimal = ldg.AccountBalance("food", time.Time{}.AddDate(0, 0, 2), date)
	bal, exact = balDecimal.Float64()
	require.True(t, exact)
	assert.EqualValues(t, 25, bal)
}

func TestLeftOverAccountBalances(t *testing.T) {
	makeTxn := func(account string, num float64) Transaction {
		return Transaction{
			Postings: []Posting{
				{Account: account, Amount: *decFloat(num)},
			},
		}
	}
	ldg, err := New([]Transaction{
		makeTxn("food:groceries", 1),
		makeTxn("food:restaurants", 2),
		makeTxn("food", 3),
		makeTxn("shopping:electronics", 150),
		makeTxn("shopping", 50),
		makeTxn("shopping:gifts", 320),
	})
	require.NoError(t, err)

	t.Run("exclude specific category", func(t *testing.T) {
		balancesDecimal := ldg.LeftOverAccountBalances(time.Time{}, time.Time{}, "food:groceries")
		balances := make(map[string]float64, len(balancesDecimal))
		for key, value := range balancesDecimal {
			var exact bool
			balances[key], exact = value.Float64()
			require.True(t, exact)
		}
		assert.Equal(t, map[string]float64{
			"food:restaurants":     2,
			"food":                 3,
			"shopping":             50,
			"shopping:electronics": 150,
			"shopping:gifts":       320,
		}, balances)
	})

	t.Run("exclude entire prefix", func(t *testing.T) {
		balancesDecimal := ldg.LeftOverAccountBalances(time.Time{}, time.Time{}, "shopping")
		balances := make(map[string]float64, len(balancesDecimal))
		for key, value := range balancesDecimal {
			var exact bool
			balances[key], exact = value.Float64()
			require.True(t, exact)
		}
		assert.Equal(t, map[string]float64{
			"food:groceries":   1,
			"food:restaurants": 2,
			"food":             3,
		}, balances)
	})
}

func TestTransaction(t *testing.T) {
	someTxn := Transaction{
		Postings: []Posting{
			{Account: "some account", Amount: *decFloat(1)},
			{Account: "expenses", Amount: *decFloat(1), Tags: makeIDTag("some-id")},
		},
	}
	ldg, err := New([]Transaction{
		someTxn,
	})
	require.NoError(t, err)
	_, found := ldg.Transaction("non-existent txn")
	assert.False(t, found)

	txn, found := ldg.Transaction("some-id")
	assert.True(t, found)
	assert.Equal(t, someTxn, txn)
}
