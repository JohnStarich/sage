package directconnect

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
		DirectConnect      *json.RawMessage
	}

	if err := json.Unmarshal(b, &account); err != nil {
		return err
	}
	d.AccountID = account.AccountID
	d.AccountDescription = account.AccountDescription
	if account.DirectConnect == nil {
		return nil // defer validation to caller
	}
	var dc directConnect
	if err := json.Unmarshal(*account.DirectConnect, &dc); err != nil {
		return err
	}
	d.DirectConnect = &dc
	return nil
}

func (d *directAccount) Validate() error {
	var errs sErrors.Errors
	errs.AddErr(model.ValidatePartialAccount(d))
	errs.AddErr(Validate(d.DirectConnect))
	errs.ErrIf(d.AccountID == "", "Account ID must not be empty")
	errs.ErrIf(d.AccountDescription == "", "Account description must not be empty")
	return errs.ErrOrNil()
}

// Validate checks connector for invalid data
func Validate(connector Connector) error {
	var errs sErrors.Errors
	if errs.ErrIf(connector == nil, "Direct connect must not be empty") {
		return errs.ErrOrNil()
	}
	errs.AddErr(model.ValidateInstitution(connector))
	errs.ErrIf(connector.URL() == "", "Institution URL must not be empty")
	errs.ErrIf(connector.Username() == "", "Institution username must not be empty")
	errs.ErrIf(connector.Password() == "" && !IsLocalhostTestURL(connector.URL()), "Institution password must not be empty")
	config := connector.Config()
	errs.ErrIf(config.AppID == "", "Institution app ID must not be empty")
	errs.ErrIf(config.AppVersion == "", "Institution app ID must not be empty")
	errs.ErrIf(config.OFXVersion == "", "Institution OFX version must not be empty")
	return errs.ErrOrNil()
}

// UnmarshalAccount attempts to unmarshal the given bytes into a known Direct Connect account type and validate it
func UnmarshalAccount(b []byte) (Account, error) {
	var maybeBank bankAccount
	if err := json.Unmarshal(b, &maybeBank); err != nil {
		return nil, err
	}
	if maybeBank.isBank() {
		maybeBank.BankAccountType = strings.ToUpper(maybeBank.BankAccountType)
		return &maybeBank, maybeBank.Validate()
	}

	var creditCard CreditCard
	if err := json.Unmarshal(b, &creditCard); err != nil {
		return nil, err
	}
	return &creditCard, creditCard.Validate()
}
