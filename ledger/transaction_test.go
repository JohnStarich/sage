package ledger

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseDate(t *testing.T, s string) time.Time {
	date, err := time.Parse(DateFormat, s)
	require.NoError(t, err)
	return date
}

func TestReadAllTransactions(t *testing.T) {
	scanFromStr := func(s string) *bufio.Scanner {
		return bufio.NewScanner(bytes.NewBufferString(s))
	}

	for _, tc := range []struct {
		description  string
		input        string
		transactions []Transaction
		shouldErr    bool
	}{
		{
			description:  "read nothing",
			input:        "",
			transactions: nil,
		},
		{
			description: "read one txn",
			input: `
2019/01/02 some burger place ; hey there what's: up?
	expenses:food   $ 1.25
	assets:Bank 1  $ -1.25
			`,
			transactions: []Transaction{
				{
					Date:    parseDate(t, "2019/01/02"),
					Payee:   "some burger place",
					Comment: "hey there",
					Tags:    map[string]string{"what's": "up?"},
					Postings: []Posting{
						{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
						{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
					},
				},
			},
		},
		{
			description: "read two txns",
			input: `
2019/01/02 some burger place ; hey there what's: up?
	expenses:food   $ 1.25
	assets:Bank 1  $ -1.25
2019/01/03 some burger place ; hey there dude what's: up?
	expenses:food   $ 2.33
	assets:Bank 2  $ -2.33
			`,
			transactions: []Transaction{
				{
					Date:    parseDate(t, "2019/01/02"),
					Payee:   "some burger place",
					Comment: "hey there",
					Tags:    map[string]string{"what's": "up?"},
					Postings: []Posting{
						{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
						{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
					},
				},
				{
					Date:    parseDate(t, "2019/01/03"),
					Payee:   "some burger place",
					Comment: "hey there dude",
					Tags:    map[string]string{"what's": "up?"},
					Postings: []Posting{
						{Account: "expenses:food", Amount: *decFloat(2.33), Currency: usd},
						{Account: "assets:Bank 2", Amount: *decFloat(-2.33), Currency: usd},
					},
				},
			},
		},
		{
			description: "not enough postings x1",
			input: `
2019/01/02 some burger place ; hey there what's: up?
	expenses:food   $ 1.25
			`,
			shouldErr: true,
		},
		{
			description: "not enough postings x0",
			input: `
2019/01/02 some burger place ; hey there what's: up?
			`,
			shouldErr: true,
		},
		{
			description: "no comment",
			input: `
2019/01/02 some burger place
	expenses:food   $ 1.25
	assets:Bank 1  $ -1.25
			`,
			transactions: []Transaction{
				{
					Date:  parseDate(t, "2019/01/02"),
					Payee: "some burger place",
					Postings: []Posting{
						{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
						{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
					},
				},
			},
		},
		{
			description: "too many missing amounts",
			input: `
2019/01/02 some burger place
	expenses:food
	assets:Bank 1
			`,
			shouldErr: true,
		},
		{
			description: "wrong missing amount",
			input: `
2019/01/02 some burger place
	expenses:food
	assets:Bank 1  $ -1.25
			`,
			shouldErr: true,
		},
		{
			description: "correct missing amount",
			input: `
2019/01/02 some burger place
	expenses:food  $ 1.25
	assets:Bank 1
			`,
			transactions: []Transaction{
				{
					Date:  parseDate(t, "2019/01/02"),
					Payee: "some burger place",
					Postings: []Posting{
						{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
						{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
					},
				},
			},
		},
		{
			description: "unbalanced transaction",
			input: `
2019/01/02 some burger place
	expenses:food   $ 1.25
	assets:Bank 1  $ -5.00
			`,
			shouldErr: true,
		},
		{
			description: "transaction ends early",
			input: `
2019/01/02 some burger place
2019/01/03 some burger place
	expenses:food   $ 1.25
	assets:Bank 1
			`,
			shouldErr: true,
		},
		{
			description: "missing payee",
			input: `
2019/01/02
			`,
			shouldErr: true,
		},
		{
			description: "bad posting",
			input: `
2019/01/03 some burger place
	expenses:food   garbage
	assets:Bank 1
			`,
			shouldErr: true,
		},
		{
			description: "missing payee line",
			input: `
	expenses:food   garbage
	assets:Bank 1
			`,
			shouldErr: true,
		},
		{
			description: "transaction on last line",
			input: `
2019/01/02 some burger place
	expenses:food   $ 1.25
	assets:Bank 1`,
			transactions: []Transaction{
				{
					Date:  parseDate(t, "2019/01/02"),
					Payee: "some burger place",
					Postings: []Posting{
						{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
						{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
					},
				},
			},
		},
		{
			description: "bad transaction on last line",
			input: `
2019/01/02 some burger place
	expenses:food   $ 1.25`,
			shouldErr: true,
		},
		{
			description: "bad date",
			input: `
2019/January/02 some burger place
	expenses:food   $ 1.25
	assets:Bank 1
			`,
			shouldErr: true,
		},
		{
			description: "missing payee",
			input: `
2019/01/02
	expenses:food   $ 1.25
	assets:Bank 1
			`,
			transactions: []Transaction{
				{
					Date:  parseDate(t, "2019/01/02"),
					Payee: "",
					Postings: []Posting{
						{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
						{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
					},
				},
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			txns, err := readAllTransactions(scanFromStr(tc.input))
			if tc.shouldErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, txns, len(tc.transactions))
			assert.Equal(t, tc.transactions, txns)
		})
	}
}

func TestParseTags(t *testing.T) {
	for _, tc := range []struct {
		description string
		input       string
		comment     string
		tags        map[string]string
	}{
		{
			description: "empty comment",
			input:       "",
			comment:     "",
			tags:        nil,
		},
		{
			description: "simple comment",
			input:       "hey there",
			comment:     "hey there",
		},
		{
			description: "just one tag",
			input:       "key: value",
			comment:     "",
			tags:        map[string]string{"key": "value"},
		},
		{
			description: "multiple tags",
			input:       "key1: value, key2: value",
			comment:     "",
			tags:        map[string]string{"key1": "value", "key2": "value"},
		},
		{
			description: "command and multiple tags",
			input:       "hey there key1: value, key2: value",
			comment:     "hey there",
			tags:        map[string]string{"key1": "value", "key2": "value"},
		},
		{
			description: "invalid key value",
			input:       "key1: value, key2",
			comment:     "key1: value, key2",
			tags:        nil,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			comment, tags := parseTags(tc.input)
			assert.Equal(t, tc.comment, comment)
			assert.Equal(t, tc.tags, tags)
		})
	}
}

func TestTransactionString(t *testing.T) {
	prep := func(strs ...string) string {
		return strings.Join(strs, "\n") + "\n"
	}
	for _, tc := range []struct {
		description string
		txn         Transaction
		str         string
	}{
		{
			description: "all fields",
			txn: Transaction{
				Comment: "hey there",
				Date:    parseDate(t, "2019/01/05"),
				Payee:   "somebody",
				Postings: []Posting{
					{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
					{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
				},
				Tags: map[string]string{"what's": "up?"},
			},
			str: prep(
				`2019/01/05 somebody ; hey there what's: up?`,
				`    expenses:food   $ 1.25`,
				`    assets:Bank 1  $ -1.25`,
			),
		},
		{
			description: "no comment or tags",
			txn: Transaction{
				Date:  parseDate(t, "2019/01/05"),
				Payee: "somebody",
				Postings: []Posting{
					{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
					{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
				},
			},
			str: prep(
				`2019/01/05 somebody`,
				`    expenses:food   $ 1.25`,
				`    assets:Bank 1  $ -1.25`,
			),
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.str, tc.txn.String())
		})
	}
}

func TestTransactionBalanced(t *testing.T) {
	for _, tc := range []struct {
		description string
		txn         Transaction
		balanced    bool
	}{
		{
			description: "zero postings",
			txn:         Transaction{},
			balanced:    true,
		},
		{
			description: "one zero posting",
			txn: Transaction{
				Postings: []Posting{{Amount: decimal.Zero}},
			},
			balanced: true,
		},
		{
			description: "two balanced postings",
			txn: Transaction{
				Postings: []Posting{
					{Amount: *decFloat(1.25)},
					{Amount: *decFloat(-1.25)},
				},
			},
			balanced: true,
		},
		{
			description: "multiple balanced postings",
			txn: Transaction{
				Postings: []Posting{
					{Amount: *decFloat(1.25)},
					{Amount: *decFloat(6.25)},
					{Amount: *decFloat(-4)},
					{Amount: *decFloat(-3.50)},
				},
			},
			balanced: true,
		},
		{
			description: "one unbalanced posting",
			txn: Transaction{
				Postings: []Posting{{Amount: *decFloat(1)}},
			},
			balanced: false,
		},
		{
			description: "two unbalanced postings",
			txn: Transaction{
				Postings: []Posting{
					{Amount: *decFloat(1)},
					{Amount: *decFloat(2)},
				},
			},
			balanced: false,
		},
		{
			description: "multiple unbalanced postings",
			txn: Transaction{
				Postings: []Posting{
					{Amount: *decFloat(1)},
					{Amount: *decFloat(2)},
					{Amount: *decFloat(3)},
					{Amount: *decFloat(4)},
					{Amount: *decFloat(-100)},
				},
			},
			balanced: false,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.balanced, tc.txn.Balanced())
		})
	}
}

func TestTransactionValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, Transaction{
			Postings: []Posting{
				{Amount: *decFloat(1.25)},
				{Amount: *decFloat(-1.25)},
			},
		}.Validate())
	})

	// require 2 minimum postings to more easily verify assertions and categorize transactions
	t.Run("too few postings", func(t *testing.T) {
		assert.EqualError(t, Transaction{}.Validate(), "Transactions must have a minimum of 2 postings")
	})

	t.Run("unbalanced postings", func(t *testing.T) {
		err := Transaction{
			Postings: []Posting{
				{Amount: *decFloat(1)},
				{Amount: *decFloat(2)},
			},
		}.Validate()
		require.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "Transaction is not balanced - postings do not sum to zero:"))
	})
}
