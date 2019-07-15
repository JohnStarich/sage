package client

import (
	"fmt"
	"strings"
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

	description := a.Institution().Description() // TODO use FID? not very plain-text accounting friendly
	accountName := redactPrefix(a.ID())          // don't use account description - can lead to duplicate accounts
	return &LedgerAccountFormat{
		AccountType: accountType,
		Institution: description,
		AccountID:   accountName,
	}
}

// ParseLedgerFormat parses the given account string as a ledger account
func ParseLedgerFormat(account string) *LedgerAccountFormat {
	format := &LedgerAccountFormat{}
	components := strings.Split(account, ":")
	if len(components) == 0 {
		return format
	}
	format.AccountType = components[0]
	switch format.AccountType {
	case AssetAccount, LiabilityAccount:
		if len(components) < 3 {
			// require accountType:institution:accountNumber format
			return format
		}
		format.Institution, format.AccountID = components[1], strings.Join(components[2:], ":")
	default:
		format.Remaining = strings.Join(components[1:], ":")
	}
	return format
}

func (l *LedgerAccountFormat) String() string {
	result := ""
	for _, s := range []string{l.AccountType, l.Institution, l.AccountID, l.Remaining} {
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
	return "****" + suffix
}