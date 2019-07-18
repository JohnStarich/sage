package client

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/pkg/errors"
)

const (
	// RedactSuffixLength the number of characters that remain unredacted at the end of a string
	RedactSuffixLength = 4
	redactPrefixLength = 4

	// Ledger account types
	AssetAccount     = "assets"
	LiabilityAccount = "liabilities"
	ExpenseAccount   = "expenses"
	RevenueAccount   = "revenues"
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

type bankLike struct {
	AccountType   string
	RoutingNumber string
}

func (b bankLike) isBank() bool {
	return b.RoutingNumber != ""
}

func UnmarshalBuiltinAccount(b []byte) (Account, error) {
	maybeBank := bankLike{}
	if err := json.Unmarshal(b, &maybeBank); err != nil {
		return nil, err
	}
	maybeBank.AccountType = strings.ToUpper(maybeBank.AccountType)
	if maybeBank.isBank() {
		if IsChecking(maybeBank.AccountType) {
			checkingAccount := &Checking{}
			if err := json.Unmarshal(b, checkingAccount); err != nil {
				return nil, err
			}
			return checkingAccount, nil
		}
		if IsSavings(maybeBank.AccountType) {
			savingsAccount := &Savings{}
			if err := json.Unmarshal(b, savingsAccount); err != nil {
				return nil, err
			}
			return savingsAccount, nil
		}
		return nil, errors.New("Invalid bank AccountType: " + maybeBank.AccountType)
	}

	creditCard := &CreditCard{}
	if err := json.Unmarshal(b, creditCard); err != nil {
		return nil, err
	}
	return creditCard, nil
}
