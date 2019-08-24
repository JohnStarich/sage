package direct

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/client/model"
	sErrors "github.com/johnstarich/sage/errors"
	"github.com/pkg/errors"
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

// Validate checks the direct connect account for invalid data
func Validate(account Account) error {
	var errs sErrors.Errors
	errs.AddErr(model.ValidateAccount(account))
	if connector, ok := account.Institution().(Connector); ok {
		errs.AddErr(validateConnector(connector))
	}

	switch impl := account.(type) {
	case Bank:
		errs.ErrIf(impl.BankID() == "", "Routing number must not be empty")
	case *bankAccount:
		errs.ErrIf(impl.BankID() == "", "Routing number must not be empty")
		kind := ParseAccountType(impl.BankAccountType)
		errs.ErrIf(kind != CheckingType && kind != SavingsType, "Account type must be %s or %s", CheckingType, SavingsType)
	case *CreditCard:
		// no additional validation required
	}

	return errs.ErrOrNil()
}

func validateConnector(connector Connector) error {
	var errs sErrors.Errors
	if errs.ErrIf(connector == nil, "Direct connect must not be empty") {
		return errs.ErrOrNil()
	}
	errs.AddErr(model.ValidateInstitution(connector))
	errs.ErrIf(connector.URL() == "", "Institution URL must not be empty")
	u, err := url.Parse(connector.URL())
	if err != nil {
		errs.AddErr(errors.Wrap(err, "Institution URL is malformed"))
	} else {
		errs.ErrIf(u.Scheme != "https" && u.Hostname() != "localhost", "Institution URL is required to use HTTPS")
	}

	errs.ErrIf(connector.Username() == "", "Institution username must not be empty")
	errs.ErrIf(connector.Password() == "" && !IsLocalhostTestURL(connector.URL()), "Institution password must not be empty")
	config := connector.Config()
	errs.ErrIf(config.AppID == "", "Institution app ID must not be empty")
	errs.ErrIf(config.AppVersion == "", "Institution app ID must not be empty")
	if !errs.ErrIf(config.OFXVersion == "", "Institution OFX version must not be empty") {
		_, err := ofxgo.NewOfxVersion(config.OFXVersion)
		errs.AddErr(err)
	}
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
		return &maybeBank, Validate(&maybeBank)
	}

	var creditCard CreditCard
	if err := json.Unmarshal(b, &creditCard); err != nil {
		return nil, err
	}
	return &creditCard, Validate(&creditCard)
}
