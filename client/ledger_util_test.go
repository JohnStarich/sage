package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLedgerAccountFormat(t *testing.T) {
	for _, tc := range []struct {
		description string
		format      LedgerAccountFormat
		expected    string
	}{
		{
			description: "standard account format",
			format: LedgerAccountFormat{
				AccountType: "some account type",
				Institution: "some institution",
				AccountID:   "some account",
			},
			expected: "some account type:some institution:****ount",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.format.String())
		})
	}
}
