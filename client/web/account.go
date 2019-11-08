package web

import (
	"encoding/json"

	"github.com/johnstarich/sage/client/model"
)

// Account is a web connect enabled account
type Account interface {
	model.Account
}

type webAccount struct {
	AccountID          string
	AccountDescription string
	AccountType        string
	WebConnect         Connector
}

func (w *webAccount) ID() string {
	return w.AccountID
}

func (w *webAccount) Description() string {
	return w.AccountDescription
}

func (w *webAccount) Institution() model.Institution {
	return w.WebConnect
}

func (w *webAccount) Type() string {
	return w.AccountType
}

func (w *webAccount) UnmarshalJSON(b []byte) error {
	var account struct {
		AccountID          string
		AccountDescription string
		AccountType        string
		WebConnect         *json.RawMessage
	}

	if err := json.Unmarshal(b, &account); err != nil {
		return err
	}
	w.AccountID = account.AccountID
	w.AccountDescription = account.AccountDescription
	w.AccountType = account.AccountType
	if account.WebConnect == nil {
		return nil // defer validation to caller
	}
	var wc passwordConnector
	if err := json.Unmarshal(*account.WebConnect, &wc); err != nil {
		return err
	}
	var err error
	w.WebConnect, err = Connect(&wc)
	return err
}

// UnmarshalAccount attempts to unmarshal the given bytes into a known Web Connect account type
func UnmarshalAccount(b []byte) (Account, error) {
	var account webAccount
	if err := json.Unmarshal(b, &account); err != nil {
		return nil, err
	}
	return &account, nil
}
