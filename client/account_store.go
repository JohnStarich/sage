package client

import (
	"encoding/json"

	"github.com/johnstarich/sage/client/direct"
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/client/web"
	sErrors "github.com/johnstarich/sage/errors"
	"github.com/johnstarich/sage/plaindb"
	"github.com/pkg/errors"
)

// AccountStore enables manipulation of accounts
type AccountStore struct {
	plaindb.Bucket
}

// NewAccountStore load the accounts bucket from db
func NewAccountStore(db plaindb.DB) (*AccountStore, error) {
	bucket, err := db.Bucket("accounts", "2", &accountStoreUpgrader{})
	return &AccountStore{
		Bucket: bucket,
	}, err
}

type accountV0 struct {
	ID            string
	Description   string
	AccountType   string
	RoutingNumber string
	Institution   struct {
		Description string
		FID         string
		Org         string
		URL         string
		Username    string
		Password    string
		ClientID    string
		AppID       string
		AppVersion  string
		OFXVersion  string
	}
}

type accountStoreUpgrader struct{}

func (u *accountStoreUpgrader) Parse(dataVersion, id string, data json.RawMessage) (interface{}, error) {
	switch dataVersion {
	case "0":
		var account accountV0
		err := json.Unmarshal(data, &account)
		return account, err
	case "1", "2":
		return UnmarshalAccount(data)
	default:
		return nil, errors.Errorf("Unknown legacy version: %s", dataVersion)
	}
}

func (u *accountStoreUpgrader) Upgrade(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
	switch dataVersion {
	case "0":
		v0, ok := data.(accountV0)
		if !ok {
			return "", nil, errors.Errorf("Version mismatch with data type %T", data)
		}
		var account model.Account
		inst := direct.New(
			v0.Institution.Description,
			v0.Institution.FID,
			v0.Institution.Org,
			v0.Institution.URL,
			v0.Institution.Username,
			v0.Institution.Password,
			direct.Config{
				ClientID:   v0.Institution.ClientID,
				AppID:      v0.Institution.AppID,
				AppVersion: v0.Institution.AppVersion,
				OFXVersion: v0.Institution.OFXVersion,
			},
		)
		if v0.RoutingNumber != "" {
			// bank account
			switch direct.ParseAccountType(v0.AccountType) {
			case direct.CheckingType:
				account = direct.NewCheckingAccount(v0.ID, v0.RoutingNumber, v0.Description, inst)
			case direct.SavingsType:
				account = direct.NewSavingsAccount(v0.ID, v0.RoutingNumber, v0.Description, inst)
			default:
				return "", nil, errors.Errorf("Unrecognized bank account type: %s", v0.AccountType)
			}
		} else {
			// credit card account
			account = direct.NewCreditCard(v0.ID, v0.Description, inst)
		}
		return "1", account, nil
	case "1":
		// v2 is a no-op upgrade. switched from legacy array format to dictionary
		return "2", data, nil
	}
	return dataVersion, data, nil
}

func (u *accountStoreUpgrader) ParseLegacy(legacyData json.RawMessage) (version string, data map[string]json.RawMessage, err error) {
	if len(legacyData) == 0 {
		return "", nil, nil
	}
	var container struct {
		Version int
		Data    json.RawMessage
	}

	if err := json.Unmarshal(legacyData, &container); err != nil {
		if _, ok := err.(*json.UnmarshalTypeError); !ok {
			return "", nil, err
		}
	}

	data = make(map[string]json.RawMessage)
	switch container.Version {
	case 0:
		var v0Accounts []json.RawMessage
		if err := json.Unmarshal(legacyData, &v0Accounts); err != nil {
			return "", nil, err
		}
		for _, accountBytes := range v0Accounts {
			var account accountV0
			err := json.Unmarshal(accountBytes, &account)
			if err != nil {
				return "", nil, err
			}
			data[account.ID] = accountBytes
		}
		return "0", data, err
	case 1:
		var rawAccounts []json.RawMessage
		if err := json.Unmarshal(container.Data, &rawAccounts); err != nil {
			return "", nil, err
		}
		for _, rawAccount := range rawAccounts {
			account, err := UnmarshalAccount(rawAccount)
			if err != nil {
				return "", nil, err
			}
			data[account.ID()] = rawAccount
		}
		return "1", data, nil
	default:
		return "", nil, errors.Errorf("Unknown legacy version: %d", container.Version)
	}
}

// Update replaces the account with a matching ID, fails if the account does not exist
func (s *AccountStore) Update(id string, account model.Account) error {
	var lookup model.Account
	found, _ := s.Get(id, &lookup)
	if !found {
		return errors.Errorf("Account not found by ID: %q", id)
	}
	newID := account.ID()
	if id != newID {
		found, err := s.Get(newID, &lookup)
		if found {
			if err != nil {
				return errors.Errorf("Account already exists with that account ID: %q", newID)
			}
			return errors.Errorf("Account already exists with that account ID: %q", lookup.Description())
		}
		if err := s.Put(id, nil); err != nil {
			return err
		}
	}
	return s.Put(newID, account)
}

// Add pushes a new account into the store, fails if the account ID is already in use
func (s *AccountStore) Add(account model.Account) error {
	id := account.ID()
	var lookup model.Account
	found, _ := s.Get(id, &lookup)
	if found {
		return errors.Errorf("Account already exists with that ID: %q", id)
	}
	return s.Put(id, account)
}

// Remove deletes the account from the store by ID
func (s *AccountStore) Remove(id string) error {
	var lookup model.Account
	found, _ := s.Get(id, &lookup)
	if !found {
		return errors.Errorf("Account not found by ID: %q", id)
	}
	return s.Put(id, nil)
}

// ValidateAccount checks account for invalid data, runs validation for direct connect too
func ValidateAccount(account model.Account) error {
	var errs sErrors.Errors
	switch kind := account.(type) {
	case direct.Account:
		errs.AddErr(direct.Validate(kind))
	case web.Account:
		errs.AddErr(web.Validate(kind))
	default:
		errs.AddErr(model.ValidateAccount(account))
	}
	return errs.ErrOrNil()
}

type institutionDetector struct {
	BasicInstitution *model.BasicInstitution
	DirectConnect    *json.RawMessage
	WebConnect       *json.RawMessage
}

// UnmarshalAccount attempts to unmarshal JSON accounts from b
func UnmarshalAccount(b []byte) (model.Account, error) {
	var instDetector institutionDetector
	if err := json.Unmarshal(b, &instDetector); err != nil {
		return nil, err
	}
	switch {
	case instDetector.BasicInstitution != nil:
		var account model.BasicAccount
		if err := json.Unmarshal(b, &account); err != nil {
			return nil, err
		}
		return &account, nil
	case instDetector.DirectConnect != nil:
		return direct.UnmarshalAccount(b)
	case instDetector.WebConnect != nil:
		return web.UnmarshalAccount(b)
	default:
		return nil, errors.New("Unrecognized account type")
	}
}
