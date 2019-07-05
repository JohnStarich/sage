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
	institution baseInstitution
}

func (b baseAccount) Institution() Institution {
	return b.institution
}

func (b baseAccount) ID() string {
	return b.id
}

func (b baseAccount) Description() string {
	return b.description
}

type baseAccountJSON struct {
	ID          string
	Description string
	Institution json.RawMessage
}

func (b *baseAccount) UnmarshalJSON(buf []byte) error {
	var account baseAccountJSON
	if err := json.Unmarshal(buf, &account); err != nil {
		return err
	}
	b.id = account.ID
	b.description = account.Description
	return json.Unmarshal([]byte(account.Institution), &b.institution)
}

func (b baseAccount) MarshalJSON() ([]byte, error) {
	return b.marshalJSON(false)
}

func (b baseAccount) marshalJSON(withPassword bool) ([]byte, error) {
	var instData json.RawMessage
	var err error
	if withPassword {
		instData, err = b.institution.MarshalWithPassword()
	} else {
		instData, err = json.Marshal(b.institution)
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(baseAccountJSON{
		ID:          b.id,
		Description: b.description,
		Institution: instData,
	})
}

func (b baseAccount) MarshalWithPassword() ([]byte, error) {
	return b.marshalJSON(true)
}

// LedgerAccountName returns a suitable account name for a ledger file
func LedgerAccountName(a Account) string {
	var accountType string
	switch a.(type) {
	case *CreditCard:
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
