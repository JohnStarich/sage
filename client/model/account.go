package model

import (
	"strings"

	sErrors "github.com/johnstarich/sage/errors"
	"github.com/pkg/errors"
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

type BasicAccount struct {
	AccountDescription string
	AccountID          string
	AccountType        string
	BasicInstitution   BasicInstitution
}

func (b *BasicAccount) Institution() Institution {
	return b.BasicInstitution
}

func (b *BasicAccount) ID() string {
	return b.AccountID
}

func (b *BasicAccount) Description() string {
	return b.AccountDescription
}

// Type returns the ledger account type, such as 'assets' or 'liabilities'
func (b *BasicAccount) Type() string {
	return b.AccountType
}

func ValidatePartialAccount(account interface {
	ID() string
	Description() string
}) error {
	var errs sErrors.Errors
	errs.ErrIf(account.Description() == "", "Account description must not be empty")
	errs.ErrIf(account.ID() == "", "Account ID must not be empty")
	return errs.ErrOrNil()
}

func ValidateAccount(account Account) error {
	var errs sErrors.Errors
	errs.AddErr(ValidatePartialAccount(account))
	if !errs.ErrIf(account.Type() == "", "Account type must not be empty") {
		errs.ErrIf(account.Type() != AssetAccount && account.Type() != LiabilityAccount, "Account type must be %q or %q: %q", AssetAccount, LiabilityAccount, account.Type())
	}
	errs.AddErr(ValidateInstitution(account.Institution()))
	return errs.ErrOrNil()
}

func ValidateInstitution(inst Institution) error {
	var errs sErrors.Errors
	if errs.ErrIf(inst == nil, "Institution must not be empty") {
		return errs.ErrOrNil()
	}
	errs.ErrIf(inst.Description() == "", "Institution name must not be empty")
	errs.ErrIf(inst.FID() == "", "Institution FID must not be empty")
	errs.ErrIf(inst.Org() == "", "Institution org must not be empty")
	return errs.ErrOrNil()
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
	if len(components) < 2 {
		return nil, errors.Errorf("Account string must have at least 2 colon separated components: %q", account)
	}
	format.AccountType = components[0]
	if format.AccountType == "" {
		return nil, errors.Errorf("First component in account string must not be empty: %q", account)
	}
	switch format.AccountType {
	case AssetAccount, LiabilityAccount:
		if len(components) < 3 {
			// require accountType:institution:accountNumber format
			format.Remaining = strings.Join(components[1:], ":")
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
	if s == "" {
		return s
	}
	suffix := s
	if len(s) > RedactSuffixLength {
		suffix = s[len(s)-RedactSuffixLength:]
	}
	return strings.Repeat("*", RedactPrefixLength) + suffix
}
