package client

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"sync"

	"github.com/johnstarich/sage/client/direct"
	"github.com/johnstarich/sage/client/model"
	sErrors "github.com/johnstarich/sage/errors"
	"github.com/johnstarich/sage/redactor"
	"github.com/pkg/errors"
)

// AccountStore enables manipulation of accounts in memory
type AccountStore struct {
	accounts map[string]model.Account
	mu       sync.RWMutex
}

// NewAccountStore creates an account store from the given accounts, must not contain duplicate account IDs
func NewAccountStore(accounts []model.Account) (*AccountStore, error) {
	accountMap, err := newAccountsFromSlice(accounts)
	return &AccountStore{accounts: accountMap}, err
}

func newAccountsFromSlice(accounts []model.Account) (map[string]model.Account, error) {
	accountMap := make(map[string]model.Account)
	for _, account := range accounts {
		id := account.ID()
		if _, exists := accountMap[id]; exists {
			return nil, errors.New("Duplicate account ID: " + id)
		}
		accountMap[id] = account
	}
	return accountMap, nil
}

type accountStoreContainer struct {
	Version int
	Data    json.RawMessage
}

// NewAccountStoreFromReader returns a new account store loaded from the provided JSON-encoded reader
func NewAccountStoreFromReader(r io.Reader) (*AccountStore, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return NewAccountStore(nil)
	}
	var container accountStoreContainer
	if err := json.Unmarshal(data, &container); err != nil {
		if _, ok := err.(*json.UnmarshalTypeError); !ok {
			return nil, err
		}
	}

	switch container.Version {
	case 0:
		return decodeAccountsV0(data)
	case 1:
		var store AccountStore
		err := json.Unmarshal(container.Data, &store)
		return &store, err
	}
	return nil, errors.Errorf("Unknown accounts file spec: version %d", container.Version)
}

func decodeAccountsV0(b []byte) (*AccountStore, error) {
	type AccountV0 struct {
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

	var v0Accounts []AccountV0
	if err := json.Unmarshal(b, &v0Accounts); err != nil {
		return nil, err
	}

	var accounts []model.Account
	for _, v0 := range v0Accounts {
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
				return nil, errors.Errorf("Unrecognized bank account type: %s", v0.AccountType)
			}
		} else {
			// credit card account
			account = direct.NewCreditCard(v0.ID, v0.Description, inst)
		}
		accounts = append(accounts, account)
	}
	return NewAccountStore(accounts)
}

// Find returns the account with the given ID if it exists, otherwise found is false
func (s *AccountStore) Find(id string) (account model.Account, found bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, found = s.accounts[id]
	return
}

// FindLedger returns the account with the given ledger account string if it exists, otherwise found is false
func (s *AccountStore) FindLedger(format *model.LedgerAccountFormat) (account model.Account, found bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, account := range s.accounts {
		if format.Institution == account.Institution().Org() && len(format.AccountID) > model.RedactPrefixLength && strings.HasSuffix(account.ID(), format.AccountID[model.RedactPrefixLength:]) {
			return account, true
		}
	}
	return nil, false
}

// Update replaces the account with a matching ID, fails if the account does not exist
func (s *AccountStore) Update(id string, account model.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.accounts[id]; !exists {
		return errors.New("Account not found by ID: " + id)
	}
	newID := account.ID()
	if id != newID {
		if existingAccount, exists := s.accounts[newID]; exists {
			return errors.New("Account already exists with that account ID: " + existingAccount.Description())
		}
		delete(s.accounts, id)
	}
	s.accounts[newID] = account
	return nil
}

// Add pushes a new account into the store, fails if the account ID is already in use
func (s *AccountStore) Add(account model.Account) error {
	id := account.ID()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.accounts[id]; exists {
		return errors.New("Account already exists with that ID: " + id)
	}
	s.accounts[id] = account
	return nil
}

// Remove deletes the account from the store by ID
func (s *AccountStore) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.accounts[id]; !exists {
		return errors.New("Account not found by ID: " + id)
	}
	delete(s.accounts, id)
	return nil
}

// Iterate ranges over the accounts in the store, running f on each one until it returns false
// Returns the last return value from f
func (s *AccountStore) Iterate(f func(model.Account) (keepGoing bool)) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, account := range s.accounts {
		if !f(account) {
			return false
		}
	}
	return true
}

// ValidateAccount checks account for invalid data, runs validation for direct connect too
func ValidateAccount(account model.Account) error {
	var errs sErrors.Errors
	if dcAccount, ok := account.(direct.Account); ok {
		errs.AddErr(direct.Validate(dcAccount))
	} else {
		errs.AddErr(model.ValidateAccount(account))
	}
	return errs.ErrOrNil()
}

type institutionDetector struct {
	BasicInstitution *model.BasicInstitution
	DirectConnect    *json.RawMessage
}

// UnmarshalAccount attempts to unmarshal JSON accounts from b and validate the result
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
		return &account, model.ValidateAccount(&account)
	case instDetector.DirectConnect != nil:
		account, err := direct.UnmarshalAccount(b)
		if err != nil {
			return nil, err
		}

		if err := ValidateAccount(account); err != nil {
			return nil, err
		}
		return account, nil
	default:
		return nil, errors.New("Unrecognized account type")
	}
}

// UnmarshalJSON unmarshals from a list of accounts
func (s *AccountStore) UnmarshalJSON(b []byte) error {
	var rawAccounts []json.RawMessage
	if err := json.Unmarshal(b, &rawAccounts); err != nil {
		return err
	}
	var accounts []model.Account
	for _, rawAccount := range rawAccounts {
		account, err := UnmarshalAccount(rawAccount)
		if err != nil {
			return err
		}
		accounts = append(accounts, account)
	}
	accountMap, err := newAccountsFromSlice(accounts)
	if err != nil {
		return err
	}
	s.accounts = accountMap
	return nil
}

// MarshalJSON marshals into a sorted list of accounts
func (s *AccountStore) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.sortedAccounts())
}

func (s *AccountStore) sortedAccounts() []model.Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	accountIDs := make([]string, 0, len(s.accounts))
	for id := range s.accounts {
		accountIDs = append(accountIDs, id)
	}
	sort.Strings(accountIDs)
	accounts := make([]model.Account, 0, len(s.accounts))
	for _, id := range accountIDs {
		accounts = append(accounts, s.accounts[id])
	}
	return accounts
}

// WriteTo marshals into a sorted list of accounts with their passwords and writes to 'w'.
// Only use this when persisting the accounts, never pass this back through an API call.
// Writes the current file format version into the file to enable format upgrade.
func (s *AccountStore) WriteTo(w io.Writer) error {
	type accountStoreJSON struct {
		Version int
		Data    interface{}
	}
	encoder := redactor.NewEncoder(w)
	encoder.SetIndent("", "    ")
	return encoder.Encode(accountStoreJSON{
		Version: 1,
		Data:    s.sortedAccounts(),
	})
}
