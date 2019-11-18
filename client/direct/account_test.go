package direct

import (
	"encoding/json"
	"testing"

	"github.com/johnstarich/sage/client/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectAccountUnmarshal(t *testing.T) {
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
	var unmarshaledAccount directAccount
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
