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

func TestUpdate(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		store := NewStore(Rules{
			csvRule{
				Conditions: []string{"some condition"},
				Account2:   "some account",
			},
		})
		someRule := csvRule{
			Conditions: []string{"some other condition"},
			Account2:   "some other account",
		}
		err := store.Update(0, someRule)
		require.NoError(t, err)
		assert.Equal(t, store.rules[0], someRule)
	})

	t.Run("not found", func(t *testing.T) {
		store := NewStore(Rules{})
		err := store.Update(0, csvRule{})
		require.Error(t, err)
		assert.Equal(t, "Rule not found", err.Error())
	})
}

func TestRemove(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		store := NewStore(Rules{
			csvRule{Conditions: []string{"some condition"}, Account2: "some account"},
			csvRule{Conditions: []string{"some second condition"}, Account2: "some second account"},
			csvRule{Conditions: []string{"some third condition"}, Account2: "some third account"},
		})
		err := store.Remove(1)
		require.NoError(t, err)
		assert.Equal(t, Rules{
			csvRule{Conditions: []string{"some condition"}, Account2: "some account"},
			csvRule{Conditions: []string{"some third condition"}, Account2: "some third account"},
		}, store.rules)
	})

	t.Run("not found", func(t *testing.T) {
		store := NewStore(Rules{})
		err := store.Remove(0)
		require.Error(t, err)
		assert.Equal(t, "Rule not found", err.Error())
	})
}

func TestAdd(t *testing.T) {
	store := NewStore(Rules{
		csvRule{Conditions: []string{"some condition"}, Account2: "some account"},
	})
	someRule := csvRule{
		Conditions: []string{"some other condition"},
		Account2:   "some other account",
	}
	ix := store.Add(someRule)
	assert.Equal(t, 1, ix)
	assert.Equal(t, Rules{
		csvRule{Conditions: []string{"some condition"}, Account2: "some account"},
		csvRule{Conditions: []string{"some other condition"}, Account2: "some other account"},
	}, store.rules)
}

func TestGet(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		someRule := csvRule{Conditions: []string{"some other condition"}, Account2: "some other account"}
		store := NewStore(Rules{someRule})
		rule, err := store.Get(0)
		require.NoError(t, err)
		assert.Equal(t, someRule, rule)
	})

	t.Run("not found", func(t *testing.T) {
		store := NewStore(nil)
		_, err := store.Get(0)
		require.Error(t, err)
		assert.Equal(t, "Rule not found", err.Error())
	})
}
