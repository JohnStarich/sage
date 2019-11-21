package direct

import (
	"encoding/json"
	"testing"

	"github.com/johnstarich/sage/client/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectAccountUnmarshal(t *testing.T) {
	var unmarshaledAccount directAccount
	assert.Error(t, json.Unmarshal([]byte(`{"AccountID": true}`), &unmarshaledAccount))

	account := `{
	"AccountID": "some ID",
	"AccountDescription": "some desc",
	"DirectConnect": {
		"InstDescription": "some inst",
		"InstFID": "",
		"InstOrg": "",
		"ConnectorURL": "",
		"ConnectorUsername": "",
		"ConnectorConfig": {
			"AppID": "",
			"AppVersion": "",
			"OFXVersion": ""
		}
	}
}`
	err := json.Unmarshal([]byte(account), &unmarshaledAccount)
	require.NoError(t, err)

	assert.Equal(t, directAccount{
		AccountID:          "some ID",
		AccountDescription: "some desc",
		DirectConnect: &directConnect{
			BasicInstitution: model.BasicInstitution{
				InstDescription: "some inst",
			},
		},
	}, unmarshaledAccount)
}

func TestUnmarshalConnector(t *testing.T) {
	directConnector := `{
		"InstDescription": "some inst",
		"InstFID": "",
		"InstOrg": "",
		"ConnectorURL": "",
		"ConnectorUsername": "",
		"ConnectorConfig": {
			"AppID": "",
			"AppVersion": "",
			"OFXVersion": ""
		}
	}`
	connector, err := UnmarshalConnector([]byte(directConnector))
	require.NoError(t, err)

	assert.Equal(t, &directConnect{
		BasicInstitution: model.BasicInstitution{
			InstDescription: "some inst",
		},
	}, connector)
}

func TestValidate(t *testing.T) {
	type fakeBank struct {
		bankAccount
	}
	for _, tc := range []struct {
		description   string
		account       Account
		expectedErr   []string
		unexpectedErr []string
	}{
		{
			description: "bankAccount",
			account:     &bankAccount{},
			expectedErr: []string{
				"Account ID must not be empty",
				"Routing number must not be empty",
				`Account type must be "CHECKING" or "SAVINGS"`,
			},
		},
		{
			description: "Bank",
			account:     &fakeBank{},
			expectedErr: []string{
				"Account ID must not be empty",
				"Routing number must not be empty",
			},
			unexpectedErr: []string{
				`Account type must be "CHECKING" or "SAVINGS"`,
			},
		},
		{
			description: "CreditCard",
			account:     &CreditCard{},
			expectedErr: []string{
				"Account ID must not be empty",
			},
		},
		{
			description: "Connector institution",
			account: &CreditCard{
				directAccount: directAccount{
					DirectConnect: &directConnect{},
				},
			},
			expectedErr: []string{
				"Account ID must not be empty",
				"Institution OFX version must not be empty",
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			err := Validate(tc.account)
			if len(tc.expectedErr) > 0 || len(tc.unexpectedErr) > 0 {
				require.Error(t, err)
				for _, expectedErr := range tc.expectedErr {
					assert.Contains(t, err.Error(), expectedErr)
				}
				for _, unexpectedErr := range tc.unexpectedErr {
					assert.NotContains(t, err.Error(), unexpectedErr)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUnmarshalAccount(t *testing.T) {
	for _, tc := range []struct {
		description   string
		data          string
		expectAccount Account
		expectErr     bool
	}{
		{
			description: "bad JSON",
			data:        `garbage`,
			expectErr:   true,
		},
		{
			description: "bank",
			data:        `{"RoutingNumber": "1234"}`,
			expectAccount: &bankAccount{
				RoutingNumber: "1234",
				directAccount: directAccount{
					DirectConnect: (*directConnect)(nil),
				},
			},
		},
		{
			description: "credit card",
			data:        `{}`,
			expectAccount: &CreditCard{
				directAccount: directAccount{
					DirectConnect: (*directConnect)(nil),
				},
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			account, err := UnmarshalAccount([]byte(tc.data))
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectAccount, account)
		})
	}
}
