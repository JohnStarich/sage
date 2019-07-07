package rules

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/johnstarich/sage/ledger"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	usd          = "$"
	someAccount1 = "some account 1"
	someAccount2 = "some account 2"
	someComment  = "some comment"
)

func TestNewCSVRule(t *testing.T) {
	conditions := []string{"a", "b", "c"}
	rule, err := NewCSVRule(someAccount1, someAccount2, someComment, conditions...)
	assert.NoError(t, err)
	assert.Equal(t, csvRule{
		Conditions: conditions,
		matchLine:  regexp.MustCompile("(?i)a|b|c"),
		Account1:   someAccount1,
		Account2:   someAccount2,
		comment:    someComment,
	}, rule)

	_, err = NewCSVRule(someAccount1, someAccount2, someComment, ".**")
	assert.Error(t, err, "Invalid regex expected")
}

func TestLedgerMatchLine(t *testing.T) {
	date, err := time.Parse(ledger.DateFormat, "2019/01/02")
	require.NoError(t, err)
	amt1 := decimal.NewFromFloat(1.25)
	amt2 := decimal.NewFromFloat(2)

	txn := ledger.Transaction{
		Date:  date,
		Payee: `a "sandwich"`,
		Postings: []ledger.Posting{
			{Account: someAccount1, Amount: amt1, Balance: &amt2, Currency: usd},
			{Account: someAccount2, Amount: amt1.Neg(), Currency: usd},
		},
	}
	assert.Equal(t, `2019/01/02,"a \"sandwich\"",$,1.25,2`, ledgerMatchLine(txn))
}

func TestCSVRuleMatch(t *testing.T) {
	date, err := time.Parse(ledger.DateFormat, "2019/01/02")
	require.NoError(t, err)
	amt1 := decimal.NewFromFloat(7.35)
	amt2 := decimal.NewFromFloat(8)
	txn1 := ledger.Transaction{
		Date:  date,
		Payee: "with a balance",
		Postings: []ledger.Posting{
			{Account: someAccount1, Amount: amt1, Balance: &amt2, Currency: usd},
			{Account: someAccount2, Amount: amt1.Neg(), Currency: usd},
		},
	}
	txn2 := ledger.Transaction{
		Date:  date,
		Payee: "no balance",
		Postings: []ledger.Posting{
			{Account: someAccount1, Amount: amt1, Currency: usd},
			{Account: someAccount2, Amount: amt1.Neg(), Currency: usd},
		},
	}

	for _, tc := range []struct {
		description string
		conditions  []string
		txn         ledger.Transaction
		shouldMatch bool
	}{
		{
			description: "unconditional rule",
			conditions:  nil,
			txn:         txn1,
			shouldMatch: true,
		},
		{
			description: "match condition",
			conditions:  []string{txn1.Payee},
			txn:         txn1,
			shouldMatch: true,
		},
		{
			description: "match condition case insensitive",
			conditions:  []string{strings.ToUpper(txn1.Payee)},
			txn:         txn1,
			shouldMatch: true,
		},
		{
			description: "match first condition",
			conditions:  []string{txn1.Payee, "something not in txn"},
			txn:         txn1,
			shouldMatch: true,
		},
		{
			description: "match second condition",
			conditions:  []string{"something not in txn", txn1.Payee},
			txn:         txn1,
			shouldMatch: true,
		},
		{
			description: "don't match missing balance",
			conditions:  []string{"8"},
			txn:         txn2,
			shouldMatch: false,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			rule, err := NewCSVRule("", "", "", tc.conditions...)
			require.NoError(t, err)

			if tc.shouldMatch {
				assert.True(t, rule.Match(tc.txn))
			} else {
				assert.False(t, rule.Match(tc.txn))
			}
		})
	}
}

func TestCSVRuleApply(t *testing.T) {
	rule, err := NewCSVRule(someAccount1, someAccount2, "something %comment")
	require.NoError(t, err)

	amt1 := decimal.NewFromFloat(7.35)
	amt2 := decimal.NewFromFloat(8)
	txn := ledger.Transaction{
		Postings: []ledger.Posting{
			{Account: "", Amount: amt1, Balance: &amt2, Currency: usd, Comment: "cool"},
			{Account: "", Amount: amt1.Neg(), Currency: usd},
		},
	}

	require.Zero(t, txn.Postings[0].Account)
	require.Zero(t, txn.Postings[1].Account)
	require.Equal(t, "cool", txn.Postings[0].Comment)
	rule.Apply(&txn)
	assert.Equal(t, someAccount1, txn.Postings[0].Account)
	assert.Equal(t, someAccount2, txn.Postings[1].Account)
	assert.Equal(t, "something cool", txn.Postings[0].Comment)
}

func TestCSVRuleString(t *testing.T) {
	for _, tc := range []struct {
		description string
		rule        csvRule
		result      string
	}{
		{
			description: "one field",
			rule: csvRule{
				Account1: "some account 1",
			},
			result: `
account1 some account 1
			`,
		},
		{
			description: "every field",
			rule: csvRule{
				Account1:   "some account 1",
				Account2:   "some account 2",
				comment:    "some comment",
				Conditions: []string{"a", "b"},
			},
			result: `
if
a
b
  account1 some account 1
  account2 some account 2
  comment some comment
			`,
		},
		{
			description: "unconditional rule",
			rule: csvRule{
				Account1: "some account 1",
				Account2: "some account 2",
				comment:  "some comment",
			},
			result: `
account1 some account 1
account2 some account 2
comment some comment
			`,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, strings.TrimSpace(tc.result)+"\n", tc.rule.String())
		})
	}
}

func TestNewCSVRulesFromReader(t *testing.T) {
	for _, tc := range []struct {
		description string
		input       string
		rules       []Rule
		err         bool
		errMessage  string
	}{
		{description: "empty input"},
		{
			description: "unconditional rule",
			input: `
account1 some account
			`,
			rules: []Rule{csvRule{Account1: "some account"}},
		},
		{
			description: "one condition",
			input: `
if
match me
  account1 some account
			`,
			rules: []Rule{csvRule{
				Account1:   "some account",
				Conditions: []string{"match me"},
			}},
		},
		{
			description: "two rules",
			input: `
if
match me
  account1 some account
if
match me
  account1 some account
			`,
			rules: []Rule{
				csvRule{
					Account1:   "some account",
					Conditions: []string{"match me"},
				},
				csvRule{
					Account1:   "some account",
					Conditions: []string{"match me"},
				},
			},
		},
		{
			description: "multiple condition",
			input: `
if
match me
me too!
  account1 some account
			`,
			rules: []Rule{csvRule{
				Account1:   "some account",
				Conditions: []string{"match me", "me too!"},
			}},
		},
		{
			description: "inline condition",
			input: `
if match me
  account1 some account
			`,
			rules: []Rule{csvRule{
				Account1:   "some account",
				Conditions: []string{"match me"},
			}},
		},
		{
			description: "every field",
			input: `
if
match me
  account1 some account 1
  account2 some account 2
  comment some comment
			`,
			rules: []Rule{csvRule{
				Account1:   "some account 1",
				Account2:   "some account 2",
				comment:    "some comment",
				Conditions: []string{"match me"},
			}},
		},
		{
			description: "invalid condition",
			input: `
if
match me .**
  account1 some account 1
			`,
			err: true,
		},
		{
			description: "two if's in a row",
			input: `
if
if
  account1 some account 1
						`,
			rules: []Rule{csvRule{
				Account1:   "some account 1",
				Conditions: []string{"if"},
			}},
		},
		{
			description: "missing body",
			input: `
if
			`,
			err:        true,
			errMessage: "If statements must have a condition and expression",
		},
		{
			description: "invalid condition then start a new if",
			input: `
if
bad expression .**
  account1 some account
if
match me
  comment some comment
			`,
			err: true,
		},
		{
			description: "if statement with no conditions",
			input: `
if
  account1 some account
			`,
			err:        true,
			errMessage: "Started expressions but no conditions were found",
		},
		{
			description: "expression with only the key",
			input: `
account1
			`,
			err:        true,
			errMessage: "Rule transform line must have both key and value: 'account1'",
		},
		{
			description: "unknown expression type",
			input: `
account3 some account
			`,
			err:        true,
			errMessage: "Unrecognized rule key: 'account3'",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			buf := bytes.NewBufferString(tc.input)
			rules, err := NewCSVRulesFromReader(buf)
			if tc.err {
				t.Logf("Result: %+v", rules)
				require.Error(t, err)
				if tc.errMessage != "" {
					assert.Equal(t, tc.errMessage, err.Error())
				}
				return
			}
			require.NoError(t, err)
			for i, rule := range rules {
				cRule := rule.(csvRule)
				assert.NotNil(t, cRule.matchLine)
				cRule.matchLine = nil
				rules[i] = cRule
			}
			assert.Equal(t, tc.rules, []Rule(rules))
		})
	}
}
