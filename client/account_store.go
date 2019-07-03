package client

import (
	"encoding/json"
	"sort"

	"github.com/pkg/errors"
)

// AccountStore enables manipulation of accounts in memory
type AccountStore struct {
	accounts map[string]Account
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
	return &AccountStore{accountMap}, nil
}

// Find returns the account with the given ID if it exists, otherwise found is false
func (s *AccountStore) Find(id string) (account Account, found bool) {
	account, found = s.accounts[id]
	return
}

// Update replaces the account with a matching ID, fails if the account does not exist
func (s *AccountStore) Update(id string, account Account) error {
	if _, exists := s.accounts[id]; !exists {
		return errors.New("Account not found by ID: " + id)
	}
	s.accounts[id] = account
	return nil
}

// Add pushes a new account into the store, fails if the account ID is already in use
func (s *AccountStore) Add(account Account) error {
	id := account.ID()
	if _, exists := s.accounts[id]; exists {
		return errors.New("Account already exists with that ID: " + id)
	}
	s.accounts[id] = account
	return nil
}

// Remove deletes the account from the store by ID
func (s *AccountStore) Remove(id string) error {
	if _, exists := s.accounts[id]; !exists {
		return errors.New("Account not found by ID: " + id)
	}
	delete(s.accounts, id)
	return nil
}

// Iterate ranges over the accounts in the store, running f on each one until it returns false
// Returns the last return value from f
func (s *AccountStore) Iterate(f func(Account) (keepGoing bool)) bool {
	for _, account := range s.accounts {
		if !f(account) {
			return false
		}
	}
	return true
}

// MarshalJSON marshals into a sorted list of accounts
func (s *AccountStore) MarshalJSON() ([]byte, error) {
	accountIDs := make([]string, 0, len(s.accounts))
	for id := range s.accounts {
		accountIDs = append(accountIDs, id)
	}
	sort.Strings(accountIDs)
	accounts := make([]Account, 0, len(s.accounts))
	for _, id := range accountIDs {
		accounts = append(accounts, s.accounts[id])
	}
	return json.Marshal(accounts)
}
