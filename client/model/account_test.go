package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	a := BasicAccount{
		AccountDescription: "some description",
		AccountID:          "some ID",
		AccountType:        "some type",
		BasicInstitution:   inst,
	}

	assert.Equal(t, "some ID", a.ID())
	assert.Equal(t, "some description", a.Description())
	assert.Equal(t, inst, a.Institution())
	assert.Equal(t, "some type", a.Type())
}

func TestRedactPrefix(t *testing.T) {
	for ix, tc := range []struct {
		str      string
		expected string
	}{
		{"", ""},
		{"smol", "****smol"},
	} {
		t.Run(fmt.Sprintf("#%d - %s", ix, tc.expected), func(t *testing.T) {
			assert.Equal(t, tc.expected, redactPrefix(tc.str))
		})
	}
}

func TestValidatePartialAccount(t *testing.T) {
	errs := ValidatePartialAccount(&BasicAccount{AccountID: "", AccountDescription: ""})
	require.Error(t, errs)
	message := errs.Error()
	assert.Contains(t, message, "Account description must not be empty")
	assert.Contains(t, message, "Account ID must not be empty")
}

func TestValidateAccount(t *testing.T) {
	for _, tc := range []struct {
		description string
		account     BasicAccount
		errors      []string
	}{
		{
			description: "all empty",
			errors: []string{
				`Account ID must not be empty`,
				`Account type must not be empty`,
				`Institution name must not be empty`,
			},
		},
		{
			description: "bad account type",
			account:     BasicAccount{AccountType: "not normal"},
			errors:      []string{`Account type must be "assets" or "liabilities": "not normal"`},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			errs := ValidateAccount(&tc.account)
			require.Error(t, errs)
			message := errs.Error()
			for _, errMessage := range tc.errors {
				assert.Contains(t, message, errMessage)
			}
		})
	}
}

func TestValidateInstitution(t *testing.T) {
	errs := ValidateInstitution(nil)
	require.Error(t, errs)
	assert.Equal(t, "Institution must not be empty", errs.Error())

	errs = ValidateInstitution(BasicInstitution{})
	require.Error(t, errs)
	message := errs.Error()
	assert.Contains(t, message, "Institution name must not be empty")
	assert.Contains(t, message, "Institution FID must not be empty")
	assert.Contains(t, message, "Institution org must not be empty")
}

func TestLedgerFormat(t *testing.T) {
	format := LedgerFormat(&BasicAccount{
		AccountID:        "1234",
		AccountType:      "assets",
		BasicInstitution: BasicInstitution{InstOrg: "some org"},
	})
	assert.Equal(t, &LedgerAccountFormat{
		AccountType: "assets",
		Institution: "some org",
		AccountID:   "1234",
	}, format)
	assert.Equal(t, `assets:some org:****1234`, format.String())

	assert.Equal(t, "", (&LedgerAccountFormat{}).String())
}

func TestParseLedgerFormat(t *testing.T) {
	for _, tc := range []struct {
		account string
		format  LedgerAccountFormat
		err     string
	}{
		{
			account: "",
			err:     `Account string must have at least 2 colon separated components: ""`,
		},
		{
			account: "one component",
			err:     `Account string must have at least 2 colon separated components: "one component"`,
		},
		{
			account: ":empty first component",
			err:     `First component in account string must not be empty: ":empty first component"`,
		},
		{
			account: "a:b",
			format:  LedgerAccountFormat{AccountType: "a", Remaining: "b"},
		},
		{
			account: "assets:some inst",
			format:  LedgerAccountFormat{AccountType: "assets", Remaining: "some inst"},
		},
		{
			account: "liabilities:some inst",
			format:  LedgerAccountFormat{AccountType: "liabilities", Remaining: "some inst"},
		},
		{
			account: "assets:some inst:****ount",
			format:  LedgerAccountFormat{AccountType: "assets", Institution: "some inst", AccountID: "****ount"},
		},
	} {
		t.Run(tc.account, func(t *testing.T) {
			format, err := ParseLedgerFormat(tc.account)
			if tc.err != "" {
				require.Error(t, err)
				assert.Equal(t, err.Error(), tc.err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, &tc.format, format)
			assert.Equal(t, tc.account, format.String())
		})
	}
}

func TestLedgerAccountName(t *testing.T) {
	account := &BasicAccount{AccountID: "1234", AccountType: "some type", BasicInstitution: BasicInstitution{InstOrg: "some org"}}
	assert.Equal(t, "some type:some org:****1234", LedgerAccountName(account))
}
