package ledger

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseDate(t *testing.T, s string) time.Time {
	date, err := time.Parse(dateFormat, s)
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
						Posting{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
						Posting{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
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
						Posting{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
						Posting{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
					},
				},
				{
					Date:    parseDate(t, "2019/01/03"),
					Payee:   "some burger place",
					Comment: "hey there dude",
					Tags:    map[string]string{"what's": "up?"},
					Postings: []Posting{
						Posting{Account: "expenses:food", Amount: *decFloat(2.33), Currency: usd},
						Posting{Account: "assets:Bank 2", Amount: *decFloat(-2.33), Currency: usd},
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
						Posting{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
						Posting{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
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
						Posting{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
						Posting{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
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
						Posting{Account: "expenses:food", Amount: *decFloat(1.25), Currency: usd},
						Posting{Account: "assets:Bank 1", Amount: *decFloat(-1.25), Currency: usd},
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

func TestMax(t *testing.T) {
	assert.Equal(t, 1, max(0, 1))
	assert.Equal(t, 1, max(1, 0))
	assert.Equal(t, 1, max(1, 1))
	assert.Equal(t, -1, max(-2, -1))
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
