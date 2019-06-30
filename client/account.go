package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aclindsa/ofxgo"
)

const (
	// RedactSuffixLength the number of characters that remain unredacted at the end of a string
	RedactSuffixLength = 4
)

// Account identifies an account at a financial institution
type Account interface {
	ID() string
	Description() string
	Institution() Institution

	Statement(start, end time.Time) (ofxgo.Request, error)
}

type baseAccount struct {
	id          string
	description string
	institution Institution
}

func (b baseAccount) Institution() Institution {
	return b.institution
}

func (b baseAccount) ID() string {
	return b.id
}

func (b baseAccount) Description() string {
	if b.description != "" {
		return b.description
	}
	return b.id
}

func (b baseAccount) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID          string
		Description string
		Institution Institution
	}{
		b.id,
		b.description,
		b.institution,
	})
}

// LedgerAccountName returns a suitable account name for a ledger file
func LedgerAccountName(a Account) string {
	var accountType string
	switch a.(type) {
	case CreditCard:
		accountType = "liabilities"
	case Bank:
		accountType = "assets"
	default:
		panic(fmt.Sprintf("Unknown account type: %T", a))
	}

	description := a.Institution().Description() // TODO use FID? not very plain-text accounting friendly
	accountName := redactPrefix(a.ID())          // don't use account description - can lead to duplicate accounts
	return fmt.Sprintf("%s:%s:%s", accountType, description, accountName)
}

func redactPrefix(s string) string {
	suffix := s
	if len(s) > RedactSuffixLength {
		suffix = s[len(s)-RedactSuffixLength:]
	}
	return "****" + suffix
}
