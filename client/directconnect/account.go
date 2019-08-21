package directconnect

import (
	"encoding/json"

	"github.com/johnstarich/sage/client/model"
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

/*
func (b directAccount) MarshalJSON() ([]byte, error) {
	return b.marshalJSON(false)
}

func (b directAccount) marshalJSON(withPassword bool) ([]byte, error) {
	var connector json.RawMessage
	var err error
	if withPassword {
		connector, err = b.DirectConnect.MarshalWithPassword()
	} else {
		connector, err = json.Marshal(b.DirectConnect)
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(directAccount{
		AccountID:          b.AccountID,
		AccountDescription: b.AccountDescription,
		DirectConnect:      &connector,
	})
}

func (b directAccount) MarshalWithPassword() ([]byte, error) {
	return b.marshalJSON(true)
}
*/

// UnmarshalAccount attempts to unmarshal the given bytes into a known Direct Connect account type
func UnmarshalAccount(b []byte) (Account, error) {
	var maybeBank bankAccount
	if err := json.Unmarshal(b, &maybeBank); err != nil {
		return nil, err
	}
	if maybeBank.isBank() {
		return &maybeBank, nil
	}

	var creditCard CreditCard
	if err := json.Unmarshal(b, &creditCard); err != nil {
		return nil, err
	}
	return &creditCard, nil
}
