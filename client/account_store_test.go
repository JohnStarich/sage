package client

import (
	"strings"
	"testing"

	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/plaindb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccountStore(t *testing.T) {
	db := plaindb.NewMockDB(plaindb.MockConfig{})
	store, err := NewAccountStore(db)
	require.NoError(t, err)
	bucket, err := db.Bucket("accounts", "2", &accountStoreUpgrader{})
	require.NoError(t, err)
	assert.Equal(t, bucket, store.Bucket)
}

func TestAccountStoreUpgradeV0(t *testing.T) {
	for _, tc := range []struct {
		description string
		v0          string
		v2          string
	}{
		{
			description: "empty accounts file",
			v0:          ``,
			v2: `
{
	"Version": "2",
	"Data": {}
}`,
		},
		{
			description: "checking account",
			v0: `[
{
	"Description": "Super bank",
	"ID": "123456789",
	"AccountType": "CHECKING",
	"RoutingNumber": "1234567890",
	"Institution": {
		"Description": "123456789",
		"FID": "123456",
		"Org": "test org",
		"URL": "http://localhost:8000/",
		"Username": "1234567890",
		"Password": "hey there",
		"AppID": "QWIN",
		"AppVersion": "2500",
		"OFXVersion": "202"
	}
}
]`,
			v2: `
{
	"Version": "2",
	"Data": {
		"123456789": {
			"AccountID": "123456789",
			"AccountDescription": "Super bank",
			"DirectConnect": {
				"InstDescription": "123456789",
				"InstFID": "123456",
				"InstOrg": "test org",
				"ConnectorURL": "http://localhost:8000/",
				"ConnectorUsername": "1234567890",
				"ConnectorPassword": "hey there",
				"ConnectorConfig": {
					"AppID": "QWIN",
					"AppVersion": "2500",
					"OFXVersion": "202"
				}
			},
			"BankAccountType": "CHECKING",
			"RoutingNumber": "1234567890"
		}
	}
}`,
		},
		{
			description: "savings account",
			v0: `[
{
	"Description": "Super bank",
	"ID": "123456789",
	"AccountType": "SAVINGS",
	"RoutingNumber": "1234567890",
	"Institution": {
		"Description": "123456789",
		"FID": "123456",
		"Org": "test org",
		"URL": "http://localhost:8000/",
		"Username": "1234567890",
		"Password": "hey there",
		"AppID": "QWIN",
		"AppVersion": "2500",
		"OFXVersion": "202"
	}
}
]`,
			v2: `
{
	"Version": "2",
	"Data": {
		"123456789": {
			"AccountID": "123456789",
			"AccountDescription": "Super bank",
			"DirectConnect": {
				"InstDescription": "123456789",
				"InstFID": "123456",
				"InstOrg": "test org",
				"ConnectorURL": "http://localhost:8000/",
				"ConnectorUsername": "1234567890",
				"ConnectorPassword": "hey there",
				"ConnectorConfig": {
					"AppID": "QWIN",
					"AppVersion": "2500",
					"OFXVersion": "202"
				}
			},
			"BankAccountType": "SAVINGS",
			"RoutingNumber": "1234567890"
		}
	}
}`,
		},
		{
			description: "credit card account",
			v0: `[
{
	"Description": "Bro Card",
	"ID": "1234",
	"Institution": {
		"Description": "Bro Cards for All",
		"FID": "1234",
		"Org": "BRO",
		"URL": "http://localhost:8000/",
		"Username": "brotato",
		"Password": "sup",
		"AppID": "QWIN",
		"AppVersion": "2500",
		"OFXVersion": "102"
	}
}
]`,
			v2: `
{
	"Version": "2",
	"Data": {
		"1234": {
			"AccountID": "1234",
			"AccountDescription": "Bro Card",
			"DirectConnect": {
				"InstDescription": "Bro Cards for All",
				"InstFID": "1234",
				"InstOrg": "BRO",
				"ConnectorURL": "http://localhost:8000/",
				"ConnectorUsername": "brotato",
				"ConnectorPassword": "sup",
				"ConnectorConfig": {
					"AppID": "QWIN",
					"AppVersion": "2500",
					"OFXVersion": "102"
				}
			}
		}
	}
}`,
		},
		{
			description: "multiple accounts",
			v0: `[
{
	"Description": "Super bank",
	"ID": "123456789",
	"AccountType": "SAVINGS",
	"RoutingNumber": "1234567890",
	"Institution": {
		"Description": "123456789",
		"FID": "123456",
		"Org": "test org",
		"URL": "http://localhost:8000/",
		"Username": "1234567890",
		"Password": "hey there",
		"AppID": "QWIN",
		"AppVersion": "2500",
		"OFXVersion": "202"
	}
},
{
	"Description": "Bro Card",
	"ID": "1234",
	"Institution": {
		"Description": "Bro Cards for All",
		"FID": "1234",
		"Org": "BRO",
		"URL": "http://localhost:8000/",
		"Username": "brotato",
		"Password": "sup",
		"AppID": "QWIN",
		"AppVersion": "2500",
		"OFXVersion": "102"
	}
}
]`,
			v2: `
{
	"Version": "2",
	"Data": {
		"1234": {
			"AccountID": "1234",
			"AccountDescription": "Bro Card",
			"DirectConnect": {
				"InstDescription": "Bro Cards for All",
				"InstFID": "1234",
				"InstOrg": "BRO",
				"ConnectorURL": "http://localhost:8000/",
				"ConnectorUsername": "brotato",
				"ConnectorPassword": "sup",
				"ConnectorConfig": {
					"AppID": "QWIN",
					"AppVersion": "2500",
					"OFXVersion": "102"
				}
			}
		},
		"123456789": {
			"AccountID": "123456789",
			"AccountDescription": "Super bank",
			"DirectConnect": {
				"InstDescription": "123456789",
				"InstFID": "123456",
				"InstOrg": "test org",
				"ConnectorURL": "http://localhost:8000/",
				"ConnectorUsername": "1234567890",
				"ConnectorPassword": "hey there",
				"ConnectorConfig": {
					"AppID": "QWIN",
					"AppVersion": "2500",
					"OFXVersion": "202"
				}
			},
			"BankAccountType": "SAVINGS",
			"RoutingNumber": "1234567890"
		}
	}
}`,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			db := plaindb.NewMockDB(plaindb.MockConfig{
				FileReader: func(path string) ([]byte, error) {
					return []byte(tc.v0), nil
				},
			})
			bucket, err := db.Bucket("accounts", "2", &accountStoreUpgrader{})
			require.NoError(t, err, "Error type: %T", err)
			store := &AccountStore{Bucket: bucket}

			expected := strings.ReplaceAll(strings.TrimSpace(tc.v2), "\t", "    ")
			output := strings.TrimSpace(db.Dump(store.Bucket))
			assert.Equal(t, expected, output)
		})
	}
}

func TestAccountStoreUpgradeV1(t *testing.T) {
	for _, tc := range []struct {
		description string
		v1          string
		v2          string
	}{
		{
			description: "savings account",
			v1: `
{
	"Version": 1,
	"Data": [
		{
			"AccountID": "123456789",
			"AccountDescription": "Super bank",
			"DirectConnect": {
				"InstDescription": "123456789",
				"InstFID": "123456",
				"InstOrg": "test org",
				"ConnectorURL": "http://localhost:8000/",
				"ConnectorUsername": "1234567890",
				"ConnectorPassword": "hey there",
				"ConnectorConfig": {
					"AppID": "QWIN",
					"AppVersion": "2500",
					"OFXVersion": "202"
				}
			},
			"BankAccountType": "SAVINGS",
			"RoutingNumber": "1234567890"
		}
	]
}`,
			v2: `
{
	"Version": "2",
	"Data": {
		"123456789": {
			"AccountID": "123456789",
			"AccountDescription": "Super bank",
			"DirectConnect": {
				"InstDescription": "123456789",
				"InstFID": "123456",
				"InstOrg": "test org",
				"ConnectorURL": "http://localhost:8000/",
				"ConnectorUsername": "1234567890",
				"ConnectorPassword": "hey there",
				"ConnectorConfig": {
					"AppID": "QWIN",
					"AppVersion": "2500",
					"OFXVersion": "202"
				}
			},
			"BankAccountType": "SAVINGS",
			"RoutingNumber": "1234567890"
		}
	}
}`,
		},
		{
			description: "OFX import account",
			v1: `
{
	"Version": 1,
	"Data": [
		{
			"AccountDescription": "Super bank",
			"AccountID": "123456",
			"AccountType": "assets",
			"BasicInstitution": {
				"InstDescription": "123456789",
				"InstFID": "123456",
				"InstOrg": "test org"
			}
		}
	]
}`,
			v2: `
{
	"Version": "2",
	"Data": {
		"123456": {
			"AccountDescription": "Super bank",
			"AccountID": "123456",
			"AccountType": "assets",
			"BasicInstitution": {
				"InstDescription": "123456789",
				"InstFID": "123456",
				"InstOrg": "test org"
			}
		}
	}
}`,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			db := plaindb.NewMockDB(plaindb.MockConfig{
				FileReader: func(path string) ([]byte, error) {
					return []byte(tc.v1), nil
				},
			})
			bucket, err := db.Bucket("accounts", "2", &accountStoreUpgrader{})
			require.NoError(t, err, "Error type: %T", err)
			store := &AccountStore{Bucket: bucket}

			expected := strings.ReplaceAll(strings.TrimSpace(tc.v2), "\t", "    ")
			output := strings.TrimSpace(db.Dump(store.Bucket))
			assert.Equal(t, expected, output)
		})
	}
}

func TestAccountStoreUpdate(t *testing.T) {
	setup := func() *AccountStore {
		db := plaindb.NewMockDB(plaindb.MockConfig{})
		store, err := NewAccountStore(db)
		require.NoError(t, err)
		return store
	}

	t.Run("update nothing", func(t *testing.T) {
		store := setup()
		err := store.Update("blah", nil)
		require.Error(t, err, "Can't update when no records")
		assert.Equal(t, err.Error(), `Account not found by ID: "blah"`)
	})

	t.Run("update field in-place", func(t *testing.T) {
		store := setup()
		require.NoError(t, store.Bucket.Put("1234", &model.BasicAccount{AccountID: "1234"}))
		err := store.Update("1234", &model.BasicAccount{AccountID: "1234", AccountDescription: "hi"})
		assert.NoError(t, err)
		var savedAccount *model.BasicAccount
		_, err = store.Bucket.Get("1234", &savedAccount)
		assert.NoError(t, err)
		assert.Equal(t, "hi", savedAccount.AccountDescription)
	})

	t.Run("update to different ID", func(t *testing.T) {
		store := setup()
		require.NoError(t, store.Bucket.Put("1234", &model.BasicAccount{AccountID: "1234"}))
		err := store.Update("1234", &model.BasicAccount{AccountID: "5678", AccountDescription: "hi"})
		assert.NoError(t, err)
		var savedAccount *model.BasicAccount
		_, err = store.Bucket.Get("5678", &savedAccount)
		assert.NoError(t, err)
		assert.Equal(t, "hi", savedAccount.AccountDescription)
	})

	t.Run("fail update to existing ID", func(t *testing.T) {
		store := setup()
		require.NoError(t, store.Bucket.Put("1234", &model.BasicAccount{AccountID: "1234"}))
		require.NoError(t, store.Bucket.Put("5678", &model.BasicAccount{AccountID: "5678", AccountDescription: "5 description"}))
		err := store.Update("1234", &model.BasicAccount{AccountID: "5678", AccountDescription: "hi"})
		require.Error(t, err)
		assert.Equal(t, err.Error(), `Account already exists with that account ID: "5 description"`)
	})

	t.Run("fail update with bad DB data", func(t *testing.T) {
		store := setup()
		require.NoError(t, store.Bucket.Put("1234", &model.BasicAccount{AccountID: "1234"}))
		require.NoError(t, store.Bucket.Put("5678", "bad data"))
		err := store.Update("1234", &model.BasicAccount{AccountID: "5678", AccountDescription: "hi"})
		require.Error(t, err)
		assert.Equal(t, `Account already exists with that account ID: "5678"`, err.Error())
	})
}

func TestAccountStoreAdd(t *testing.T) {
	db := plaindb.NewMockDB(plaindb.MockConfig{})
	store, err := NewAccountStore(db)
	require.NoError(t, err)

	err = store.Add(&model.BasicAccount{AccountID: "1234"})
	assert.NoError(t, err)
	err = store.Add(&model.BasicAccount{AccountID: "1234"})
	require.Error(t, err)
	assert.Equal(t, `Account already exists with that ID: "1234"`, err.Error())
}

func TestAccountStoreRemove(t *testing.T) {
	db := plaindb.NewMockDB(plaindb.MockConfig{})
	store, err := NewAccountStore(db)
	require.NoError(t, err)

	require.NoError(t, store.Bucket.Put("1234", &model.BasicAccount{AccountID: "1234"}))
	err = store.Remove("1234")
	assert.NoError(t, err)
	err = store.Remove("1234")
	require.Error(t, err)
	assert.Equal(t, `Account not found by ID: "1234"`, err.Error())
}
