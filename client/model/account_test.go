package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLedgerAccountFormat(t *testing.T) {
	// NOTE: since this skips NewLedgerFormat, redactPrefix isn't being run
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

func TestBaseAccount(t *testing.T) {
	inst := BasicInstitution{}
	a := basicAccount{
		AccountDescription: "some description",
		AccountID:          "some ID",
		BasicInstitution:   inst,
	}

	assert.Equal(t, "some ID", a.ID())
	assert.Equal(t, "some description", a.Description())
	assert.Equal(t, inst, a.Institution())
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
