package rules

import (
	"testing"

	"github.com/johnstarich/sage/ledger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	rules := Rules{csvRule{comment: "hi"}}
	assert.Equal(t, &Store{rules: rules}, NewStore(rules))
}

func TestMarhsalJSON(t *testing.T) {
	store := NewStore(Rules{
		csvRule{Conditions: []string{"Hank's burgers"}, Account2: "expenses:burgers"},
	})

	data, err := store.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `[{"Conditions":["Hank's burgers"],"Account2":"expenses:burgers"}]`, string(data))
}

func TestStoreApply(t *testing.T) {
	rule, err := NewCSVRule("", "expenses:burgers", "", "Hank's burgers")
	require.NoError(t, err)
	store := NewStore(Rules{rule})
	txn := ledger.Transaction{
		Payee: "Hank's burgers",
		Postings: []ledger.Posting{
			{Account: "assets:Some Bank"},
			{Account: "uncategorized"},
		},
	}
	store.Apply(&txn)
	assert.Equal(t, "expenses:burgers", txn.Postings[1].Account)
}

func TestStoreApplyAll(t *testing.T) {
	rule, err := NewCSVRule("", "expenses:burgers", "", "Hank's burgers")
	require.NoError(t, err)
	store := NewStore(Rules{rule})
	txns := []ledger.Transaction{
		{
			Payee: "Hank's burgers",
			Postings: []ledger.Posting{
				{Account: "assets:Some Bank"},
				{Account: "uncategorized"},
			},
		},
	}
	store.ApplyAll(txns)
	assert.Equal(t, "expenses:burgers", txns[0].Postings[1].Account)
}

func TestStoreString(t *testing.T) {
	rule, err := NewCSVRule("", "expenses:burgers", "", "Hank's burgers")
	require.NoError(t, err)
	rules := Rules{rule}
	store := NewStore(rules)
	assert.Equal(t, rules.String(), store.String())
}

func TestStoreReplace(t *testing.T) {
	rule, err := NewCSVRule("", "expenses:burgers", "", "Hank's burgers")
	require.NoError(t, err)
	rules := Rules{rule}
	store := NewStore(rules)
	store.Replace(Rules{})
	assert.Equal(t, Rules{}, store.rules)
}

func TestStoreAccounts(t *testing.T) {
	rule, err := NewCSVRule("", "expenses:burgers", "", "Hank's burgers")
	require.NoError(t, err)
	rules := Rules{rule}
	store := NewStore(rules)
	assert.Equal(t, []string{"expenses:burgers"}, store.Accounts())
}
