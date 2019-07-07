package rules

import (
	"encoding/json"
	"sync"

	"github.com/johnstarich/sage/ledger"
)

// Store enables manipulation of rules in memory
type Store struct {
	rules Rules
	mu    sync.RWMutex
}

// NewStore creates a rules store from the given rules
func NewStore(rules Rules) *Store {
	return &Store{rules: rules}
}

// MarshalJSON returns JSON-encoded rules
func (s *Store) MarshalJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s.rules)
}

// Apply transforms the given transaction based on the current rules
func (s *Store) Apply(txn *ledger.Transaction) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.rules.Apply(txn)
}

// ApplyAll transforms the given transactions based on the current rules
func (s *Store) ApplyAll(txns []ledger.Transaction) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range txns {
		s.rules.Apply(&txns[i])
	}
}

func (s *Store) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rules.String()
}

// Replace replaces the current rules with newRules
func (s *Store) Replace(newRules Rules) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = newRules
}
