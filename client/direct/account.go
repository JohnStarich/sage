package direct

import (
	"encoding/json"
	"strings"

	"github.com/johnstarich/sage/client/model"
	sErrors "github.com/johnstarich/sage/errors"
)

// Account is a direct connect enabled account
type Account interface {
	model.Account
	Requestor
}

type directAccount struct {
	AccountID          string
	AccountDescription string
	DirectConnect      Connector
}

// ID implements model.Account
func (d *directAccount) ID() string {
	return d.AccountID
}

// Description implements model.Account
func (d *directAccount) Description() string {
	return d.AccountDescription
}

// Institution implements model.Account
func (d *directAccount) Institution() model.Institution {
	return d.DirectConnect
}

func (d *directAccount) UnmarshalJSON(b []byte) error {
	var account struct {
		AccountID          string
		AccountDescription string
		DirectConnect      *directConnect
	}

	if err := json.Unmarshal(b, &account); err != nil {
		return err
	}
	d.AccountID = account.AccountID
	d.AccountDescription = account.AccountDescription
	d.DirectConnect = account.DirectConnect
	return nil
}

// Validate checks the direct connect account for invalid data
func Validate(account Account) error {
	var errs sErrors.Errors
	errs.AddErr(model.ValidateAccount(account))
	if connector, ok := account.Institution().(Connector); ok {
		errs.AddErr(ValidateConnector(connector))
	}

	switch impl := account.(type) {
	case *bankAccount:
		errs.ErrIf(impl.BankID() == "", "Routing number must not be empty")
		kind := ParseAccountType(impl.BankAccountType)
		errs.ErrIf(kind != CheckingType && kind != SavingsType, "Account type must be %q or %q", CheckingType, SavingsType)
	case Bank:
		errs.ErrIf(impl.BankID() == "", "Routing number must not be empty")
	case *CreditCard:
		// no additional validation required
	}

	return errs.ErrOrNil()
}

// UnmarshalAccount attempts to unmarshal the given bytes into a known Direct Connect account type
func UnmarshalAccount(b []byte) (Account, error) {
	var maybeBank bankAccount
	if err := json.Unmarshal(b, &maybeBank); err != nil {
		return nil, err
	}
	if maybeBank.isBank() {
		maybeBank.BankAccountType = strings.ToUpper(maybeBank.BankAccountType)
		return &maybeBank, nil
	}

	var creditCard CreditCard
	err := json.Unmarshal(b, &creditCard)
	return &creditCard, err
}
