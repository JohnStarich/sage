package client

import (
	"fmt"
	"time"

	"github.com/aclindsa/ofxgo"
)

type Account interface {
	ID() string
	Description() string
	Institution() Institution

	Statement(start, end time.Time) (ofxgo.Request, error)
}

type baseAccount struct {
	id          string
	institution Institution
}

func (b baseAccount) Institution() Institution {
	return b.institution
}

func (b baseAccount) ID() string {
	return b.id
}

func (b baseAccount) Description() string {
	// TODO not implemented
	return ""
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

	description := a.Institution().Description()
	accountName := a.Description()
	if accountName == "" {
		accountName = redactPrefix(a.ID())
	}
	return fmt.Sprintf("%s:%s:%s", accountType, description, accountName)
}

func redactPrefix(s string) string {
	const suffixLen = 4
	suffix := s
	if len(s) > suffixLen {
		suffix = s[len(s)-suffixLen:]
	}
	return "****" + suffix
}
