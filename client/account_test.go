package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseAccount(t *testing.T) {
	inst := &institution{}
	a := baseAccount{
		id:          "some ID",
		institution: inst,
	}

	assert.Equal(t, "some ID", a.ID())
	assert.Equal(t, "", a.Description(), "Not implemented so should be ''")
	assert.Equal(t, inst, a.Institution())
}

func TestLedgerAccountName(t *testing.T) {
	for _, tc := range []struct {
		description  string
		account      Account
		expectedName string
		expectPanic  bool
	}{
		{
			description: "unknown account type",
			account:     nil,
			expectPanic: true,
		},
		{
			description: "credit cards are liability accounts",
			account: NewCreditCard(
				"super cash back",
				institution{description: "Some Credit Card Co"},
			),
			expectedName: "liabilities:Some Credit Card Co:" + redactPrefix("super cash back"),
		},
		{
			description: "banks are asset accounts",
			account: NewSavingsAccount(
				"blah account",
				"routing no",
				institution{description: "The Boring Bank"},
			),
			expectedName: "assets:The Boring Bank:" + redactPrefix("blah account"),
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			if tc.expectPanic {
				assert.PanicsWithValue(t, "Unknown account type: <nil>", func() {
					LedgerAccountName(tc.account)
				})
				return
			}
			assert.Equal(t, tc.expectedName, LedgerAccountName(tc.account))
		})
	}
}

func TestRedactPrefix(t *testing.T) {
	for ix, tc := range []struct {
		str      string
		expected string
	}{
		{"", "****"},
		{"smol", "****smol"},
	} {
		t.Run(fmt.Sprintf("#%d - %s", ix, tc.expected), func(t *testing.T) {
			assert.Equal(t, tc.expected, redactPrefix(tc.str))
		})
	}
}
