package client

import (
	"encoding/json"
	"sort"
	"sync"

	"github.com/pkg/errors"
)

// AccountStore enables manipulation of accounts in memory
type AccountStore struct {
	accounts map[string]Account
	mu       sync.RWMutex
}

// NewAccountStore creates an account store from the given accounts, must not contain duplicate account IDs
func NewAccountStore(accounts []Account) (*AccountStore, error) {
	accountMap := make(map[string]Account)
	for _, account := range accounts {
		id := account.ID()
		if _, exists := accountMap[id]; exists {
			return nil, errors.New("Duplicate account ID: " + id)
		}
		accountMap[id] = account
	}
	return &AccountStore{accounts: accountMap}, nil
}

// Find returns the account with the given ID if it exists, otherwise found is false
func (s *AccountStore) Find(id string) (account Account, found bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, found = s.accounts[id]
	return
}

// Update replaces the account with a matching ID, fails if the account does not exist
func (s *AccountStore) Update(id string, account Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.accounts[id]; !exists {
		return errors.New("Account not found by ID: " + id)
	}
	s.accounts[id] = account
	return nil
}

// Add pushes a new account into the store, fails if the account ID is already in use
func (s *AccountStore) Add(account Account) error {
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
func (s *AccountStore) Iterate(f func(Account) (keepGoing bool)) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, account := range s.accounts {
		if !f(account) {
			return false
		}
	}
	return true
}

// MarshalJSON marshals into a sorted list of accounts
func (s *AccountStore) MarshalJSON() ([]byte, error) {
	return s.marshalJSON(false)
}

func (s *AccountStore) marshalJSON(encodePasswords bool) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	accountIDs := make([]string, 0, len(s.accounts))
	for id := range s.accounts {
		accountIDs = append(accountIDs, id)
	}
	sort.Strings(accountIDs)
	accounts := make([]json.RawMessage, 0, len(s.accounts))
	for _, id := range accountIDs {
		var data json.RawMessage
		var err error
		if impl, ok := s.accounts[id].(PasswordMarshaler); encodePasswords && ok {
			data, err = impl.MarshalWithPassword()
		} else {
			data, err = json.Marshal(s.accounts[id])
		}
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, data)
	}
	return json.Marshal(accounts)
}

func (s *AccountStore) MarshalWithPassword() ([]byte, error) {
	return s.marshalJSON(true)
}
