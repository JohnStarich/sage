package client

import (
	"fmt"
	"time"

	"github.com/aclindsa/ofxgo"
)

const (
	RedactSuffixLength = 4
)

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
