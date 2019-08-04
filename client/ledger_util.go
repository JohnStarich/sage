package client

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// LedgerAccountFormat represents an account's structured name for a ledger account format
type LedgerAccountFormat struct {
	AccountType string
	Institution string
	AccountID   string
	Remaining   string
}

// LedgerFormat parses the account and returns a ledger account format
func LedgerFormat(a Account) *LedgerAccountFormat {
	var accountType string
	switch a.(type) {
	case *CreditCard:
		accountType = LiabilityAccount
	case Bank:
		accountType = AssetAccount
	default:
		panic(fmt.Sprintf("Unknown account type: %T", a))
	}

	return &LedgerAccountFormat{
		AccountType: accountType,
		Institution: a.Institution().Org(),
		AccountID:   a.ID(),
	}
}

// ParseLedgerFormat parses the given account string as a ledger account
func ParseLedgerFormat(account string) (*LedgerAccountFormat, error) {
	format := &LedgerAccountFormat{}
	components := strings.Split(account, ":")
	if len(components) == 0 {
		return nil, errors.New("Account string must have at least 2 colon separated components: " + account)
	}
	format.AccountType = components[0]
	if format.AccountType == "" {
		return nil, errors.New("First component in account string must not be empty: " + account)
	}
	switch format.AccountType {
	case AssetAccount, LiabilityAccount:
		if len(components) < 3 {
			// require accountType:institution:accountNumber format
			return format, nil
		}
		format.Institution, format.AccountID = components[1], strings.Join(components[2:], ":")
	default:
		format.Remaining = strings.Join(components[1:], ":")
	}
	return format, nil
}

func (l *LedgerAccountFormat) String() string {
	result := ""
	for _, s := range []string{l.AccountType, l.Institution, redactPrefix(l.AccountID), l.Remaining} {
		if s != "" {
			result += s + ":"
		}
	}
	if result == "" {
		return result
	}
	return result[:len(result)-1]
}

// LedgerAccountName returns a suitable account name for a ledger file
func LedgerAccountName(a Account) string {
	return LedgerFormat(a).String()
}

func redactPrefix(s string) string {
	suffix := s
	if len(s) > RedactSuffixLength {
		suffix = s[len(s)-RedactSuffixLength:]
	}
	return strings.Repeat("*", redactPrefixLength) + suffix
}
