package model

import (
	"errors"
	"strings"
)

const (
	// Uncategorized is used as the default account2 on an imported transaction
	Uncategorized = "uncategorized"

	// Ledger account types
	AssetAccount     = "assets"
	LiabilityAccount = "liabilities"
	ExpenseAccount   = "expenses"
	RevenueAccount   = "revenues"

	// RedactSuffixLength the number of characters that remain unredacted at the end of a redacted string
	RedactSuffixLength = 4
	// RedactPrefixLength the number of stars at the end beginning of a redacted string
	RedactPrefixLength = 4
)

// Account identifies an account at a financial institution
type Account interface {
	Description() string
	ID() string
	Institution() Institution
	Type() string
}

type basicAccount struct {
	AccountDescription string
	AccountID          string
	AccountType        string
	BasicInstitution   BasicInstitution
}

func (b basicAccount) Institution() Institution {
	return b.BasicInstitution
}

func (b basicAccount) ID() string {
	return b.AccountID
}

func (b basicAccount) Description() string {
	return b.AccountDescription
}

func (b basicAccount) Type() string {
	return b.AccountType
}

// LedgerAccountFormat represents an account's structured name for a ledger account format
type LedgerAccountFormat struct {
	AccountType string
	Institution string
	AccountID   string
	Remaining   string
}

// LedgerFormat parses the account and returns a ledger account format
func LedgerFormat(a Account) *LedgerAccountFormat {
	return &LedgerAccountFormat{
		AccountType: a.Type(),
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
	return strings.Repeat("*", RedactPrefixLength) + suffix
}
