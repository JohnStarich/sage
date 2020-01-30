package rules

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/johnstarich/sage/ledger"
)

// Rule can match a transaction and apply a transformation to it
type Rule interface {
	Match(ledger.Transaction) bool
	Apply(*ledger.Transaction)
}

// Rules enables transformation of transactions across multiple, sequential rules
type Rules []Rule

// Apply runs a match and subsequent apply for each rule on the given transaction
func (r Rules) Apply(txn *ledger.Transaction) {
	for _, rule := range r {
		if rule.Match(*txn) {
			rule.Apply(txn)
		}
	}
}

// Matches returns matching rules
func (r Rules) Matches(txn *ledger.Transaction) map[int]Rule {
	matchingRules := make(map[int]Rule)
	for ix, rule := range r {
		if rule.Match(*txn) {
			matchingRules[ix] = rule
		}
	}
	return matchingRules
}

// UnmarshalJSON parses the given bytes into rules
func (r *Rules) UnmarshalJSON(b []byte) error {
	var rules []csvRule
	if err := json.Unmarshal(b, &rules); err != nil {
		return err
	}
	*r = make(Rules, len(rules))
	for i := range rules {
		(*r)[i] = rules[i]
	}
	return nil
}

func (r Rules) String() string {
	var buf strings.Builder
	for _, rule := range r {
		buf.WriteString(fmt.Sprintf("%s\n", rule))
	}
	return buf.String()
}
