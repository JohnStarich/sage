package rules

import (
	"fmt"
	"strings"

	"github.com/johnstarich/sage/ledger"
)

type Rule interface {
	Match(ledger.Transaction) bool
	Apply(*ledger.Transaction)
}

type Rules []Rule

func (r Rules) Apply(txn *ledger.Transaction) {
	for _, rule := range r {
		if rule.Match(*txn) {
			rule.Apply(txn)
		}
	}
}

func (r Rules) String() string {
	var buf strings.Builder
	for _, rule := range r {
		buf.WriteString(fmt.Sprintf("%s\n", rule))
	}
	return buf.String()
}
