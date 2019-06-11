package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountsFromOFXClientINI(t *testing.T) {
	accounts, err := AccountsFromOFXClientINI("testdata/does_not_exist")
	assert.Error(t, err, "Should not load non-existent file")

	accounts, err = AccountsFromOFXClientINI("testdata/ofxclient.ini")
	require.NoError(t, err)
	assert.Equal(t, []Account{
		NewCreditCard(
			"123456789",
			NewInstitution("Some Cool Credit Card", "1234", "SOME", "https://ofx.neato.url/", "wowen", "some password", Config{
				AppID:      "QWIN",
				AppVersion: "2500",
				OFXVersion: "102",
			}),
		),
	}, accounts)
}

func TestAccountsFromOFXClientConfig(t *testing.T) {
	someDescription := "Some cool place"
	someFID := "some FID"
	someOrg := "some org"
	someURL := "some URL"
	someUsername := "some username"
	somePassword := "some password"
	someConfig := Config{
		AppID:      "some app ID",
		AppVersion: "some app version",
		OFXVersion: "some OFX version",
	}
	someAccountNumber := "123456789"
	someRoutingNumber := "987654321"
	for _, tc := range []struct {
		description      string
		config           credConfig
		expectedAccounts []Account
		expectedErr      string
	}{
		{
			description: "checking account",
			config: credConfig{
				{
					"institution.description":             someDescription,
					"institution.id":                      someFID,
					"institution.org":                     someOrg,
					"institution.url":                     someURL,
					"institution.username":                someUsername,
					"institution.password":                somePassword,
					"institution.client_args.app_id":      someConfig.AppID,
					"institution.client_args.app_version": someConfig.AppVersion,
					"institution.client_args.ofx_version": someConfig.OFXVersion,
					"account_type":                        checkingType,
					"number":                              someAccountNumber,
					"routing_number":                      someRoutingNumber,
				},
			},
			expectedAccounts: []Account{
				NewCheckingAccount(
					someAccountNumber,
					someRoutingNumber,
					NewInstitution(someDescription, someFID, someOrg, someURL, someUsername, somePassword, someConfig),
				),
			},
		},
		{
			description: "savings account",
			config: credConfig{
				{
					"institution.description":             someDescription,
					"institution.id":                      someFID,
					"institution.org":                     someOrg,
					"institution.url":                     someURL,
					"institution.username":                someUsername,
					"institution.password":                somePassword,
					"institution.client_args.app_id":      someConfig.AppID,
					"institution.client_args.app_version": someConfig.AppVersion,
					"institution.client_args.ofx_version": someConfig.OFXVersion,
					"account_type":                        savingsType,
					"number":                              someAccountNumber,
					"routing_number":                      someRoutingNumber,
				},
			},
			expectedAccounts: []Account{
				NewSavingsAccount(
					someAccountNumber,
					someRoutingNumber,
					NewInstitution(someDescription, someFID, someOrg, someURL, someUsername, somePassword, someConfig),
				),
			},
		},
		{
			description: "credit card account",
			config: credConfig{
				{
					"institution.description":             someDescription,
					"institution.id":                      someFID,
					"institution.org":                     someOrg,
					"institution.url":                     someURL,
					"institution.username":                someUsername,
					"institution.password":                somePassword,
					"institution.client_args.app_id":      someConfig.AppID,
					"institution.client_args.app_version": someConfig.AppVersion,
					"institution.client_args.ofx_version": someConfig.OFXVersion,
					"number":                              someAccountNumber,
				},
			},
			expectedAccounts: []Account{
				NewCreditCard(
					someAccountNumber,
					NewInstitution(someDescription, someFID, someOrg, someURL, someUsername, somePassword, someConfig),
				),
			},
		},
		{
			description: "missing some fields",
			config: credConfig{
				{
					"TYPO GOES HERE":                      someDescription,
					"institution.id":                      someFID,
					"institution.org":                     someOrg,
					"institution.url":                     someURL,
					"institution.username":                someUsername,
					"institution.password":                somePassword,
					"institution.client_args.app_id":      someConfig.AppID,
					"institution.client_args.app_version": someConfig.AppVersion,
					"institution.client_args.ofx_version": someConfig.OFXVersion,
					"number":                              someAccountNumber,
				},
			},
			expectedErr: "Failed to parse ofxclient.ini: \nMissing required field 'institution.description' for ofxclient account #1 ':'",
		},
		{
			description: "missing some fields",
			config: credConfig{
				{
					"institution.description":             someDescription,
					"institution.id":                      someFID,
					"institution.org":                     someOrg,
					"institution.url":                     someURL,
					"institution.username":                someUsername,
					"institution.password":                somePassword,
					"institution.client_args.app_id":      someConfig.AppID,
					"institution.client_args.app_version": someConfig.AppVersion,
					"institution.client_args.ofx_version": someConfig.OFXVersion,
					"number":                              someAccountNumber,
					"account_type":                        "nah man",
				},
			},
			expectedErr: "Failed to parse ofxclient.ini: \nUnknown account type 'nah man' for ofxclient account #1 'Some cool place:'",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			accounts, err := accountsFromOFXClientConfig(tc.config)
			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Equal(t, tc.expectedErr, err.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedAccounts, accounts)
		})
	}
}
