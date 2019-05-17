package ledger

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func decFloat(f float64) *decimal.Decimal {
	dec := decimal.NewFromFloat(f)
	return &dec
}

func TestNewPostingFromString(t *testing.T) {
	for _, tc := range []struct {
		description string
		str         string
		posting     Posting
		shouldErr   bool
	}{
		{
			description: "fully specified posting",
			str:         "assets:Bank1  $ 1.25 = $ 101.25 ; hey there what's: up?",
			posting: Posting{
				Account:  "assets:Bank1",
				Amount:   decFloat(1.25),
				Balance:  decFloat(101.25),
				Comment:  "hey there",
				Currency: usd,
				Tags:     map[string]string{"what's": "up?"},
			},
		},
		{
			description: "missing tags",
			str:         "assets:Bank1  $ 1.25 = $ 101.25 ; hey there",
			posting: Posting{
				Account:  "assets:Bank1",
				Amount:   decFloat(1.25),
				Balance:  decFloat(101.25),
				Comment:  "hey there",
				Currency: usd,
			},
		},
		{
			description: "missing comment",
			str:         "assets:Bank1  $ 1.25 = $ 101.25",
			posting: Posting{
				Account:  "assets:Bank1",
				Amount:   decFloat(1.25),
				Balance:  decFloat(101.25),
				Currency: usd,
			},
		},
		{
			description: "missing balance",
			str:         "assets:Bank1  $ 1.25 ; hey there",
			posting: Posting{
				Account:  "assets:Bank1",
				Amount:   decFloat(1.25),
				Comment:  "hey there",
				Currency: usd,
			},
		},
		{
			description: "only account and amount",
			str:         "assets:Bank1  $ 1.25",
			posting: Posting{
				Account:  "assets:Bank1",
				Amount:   decFloat(1.25),
				Currency: usd,
			},
		},
		{
			description: "only account",
			str:         "assets:Bank1",
			posting: Posting{
				Account: "assets:Bank1",
			},
		},
		{
			description: "invalid amount",
			str:         "assets:Bank1  $ abc",
			shouldErr:   true,
		},
		{
			description: "invalid balance",
			str:         "assets:Bank1  $ 1.25 =",
			shouldErr:   true,
		},
		{
			description: "blank line",
			str:         "   ",
			shouldErr:   true,
		},
		{
			description: "account with space",
			str:         "assets:Bank 1",
			posting:     Posting{Account: "assets:Bank 1"},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			posting, err := NewPostingFromString(tc.str)
			if tc.shouldErr {
				assert.Error(t, err)
				return
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.posting.Amount, posting.Amount, "Amount is incorrect")
			assert.Equal(t, tc.posting.Balance, posting.Balance, "Balance is incorrect")
			assert.Equal(t, tc.posting, posting)
		})
	}
}

func TestPostingString(t *testing.T) {
	for _, tc := range []struct {
		description string
		posting     Posting
		str         string
	}{
		{
			description: "fully specified posting",
			posting: Posting{
				Account:  "assets:Bank 1",
				Amount:   decFloat(1.25),
				Balance:  decFloat(101.25),
				Comment:  "hey there",
				Currency: "$",
				Tags:     map[string]string{"what's": "up?"},
			},
			str: "assets:Bank 1  $ 1.25 = $ 101.25 ; hey there what's: up?",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.str, tc.posting.String())
		})
	}
}
