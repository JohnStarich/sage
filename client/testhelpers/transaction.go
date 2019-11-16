package testhelpers

import (
	"testing"

	"github.com/johnstarich/sage/ledger"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// AssertEqualTransactions carefully compares postings, with special handling for amounts and balances
func AssertEqualTransactions(t *testing.T, expected, actual ledger.Transaction) bool {
	failed := false
	for i := range expected.Postings {
		if expected.Postings[i].Balance != actual.Postings[i].Balance {
			switch {
			case expected.Postings[i].Balance == nil:
				failed = failed || !assert.Nil(t, actual.Postings[i].Balance)
			case actual.Postings[i].Balance == nil:
				failed = failed || !assert.NotNil(t, actual.Postings[i].Balance)
			default:
				failed = failed || !assert.Equal(t,
					expected.Postings[i].Balance.String(),
					actual.Postings[i].Balance.String(),
					"Balances not equal for posting index #%d", i,
				)
			}
		}
		failed = failed || !assert.Equal(t,
			expected.Postings[i].Amount.String(),
			actual.Postings[i].Amount.String(),
			"Amounts not equal for posting index #%d", i,
		)
		expected.Postings[i].Balance = nil
		actual.Postings[i].Balance = nil
		expected.Postings[i].Amount = decimal.Zero
		actual.Postings[i].Amount = decimal.Zero
	}
	failed = failed || !assert.Equal(t, expected, actual)
	return !failed
}
