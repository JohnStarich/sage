package rules

import (
	"encoding/json"
	"sync"

	"github.com/johnstarich/sage/ledger"
	"github.com/pkg/errors"
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

// ApplyAll transforms the given transactions based on the current rules and the default rules.
// Custom rules take precedence to default rules.
func (s *Store) ApplyAll(txns []ledger.Transaction) {
	for i := range txns {
		Default.Apply(&txns[i])
	}
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

// Accounts returns account names (account2) for any CSV rules
func (s *Store) Accounts() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	accounts := make([]string, 0, len(s.rules))
	for _, rule := range s.rules {
		if csv, ok := rule.(csvRule); ok && csv.Account2 != "" {
			accounts = append(accounts, csv.Account2)
		}
	}
	return accounts
}

// Get returns the rule at 'index'
func (s *Store) Get(index int) (Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.rules) {
		return nil, errors.New("Rule not found")
	}
	return s.rules[index], nil
}

// Update updates the rule at 'index' with 'rule'
func (s *Store) Update(index int, rule Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index >= len(s.rules) {
		return errors.New("Rule not found")
	}
	s.rules[index] = rule
	return nil
}

// Remove removes the rule at 'index'
func (s *Store) Remove(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index >= len(s.rules) {
		return errors.New("Rule not found")
	}
	newRules := make(Rules, 0, len(s.rules)-1)
	newRules = append(newRules, s.rules[:index]...)
	newRules = append(newRules, s.rules[index+1:]...)
	s.rules = newRules
	return nil
}

// Add appends a new rule
func (s *Store) Add(rule Rule) (newRuleIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = append(s.rules, rule)
	return len(s.rules) - 1
}

// Matches returns matching rules
func (s *Store) Matches(txn *ledger.Transaction) map[int]Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rules.Matches(txn)
}
