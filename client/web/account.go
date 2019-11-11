package web

import (
	"encoding/json"

	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/redactor"
	"github.com/pkg/errors"
)

// Account is a web connect enabled account
type Account interface {
	model.Account
}

type webAccount struct {
	AccountID          string
	AccountDescription string
	AccountType        string
	WebConnect         driverContainer
}

func (w *webAccount) ID() string {
	return w.AccountID
}

func (w *webAccount) Description() string {
	return w.AccountDescription
}

func (w *webAccount) Institution() model.Institution {
	return w.WebConnect.Data
}

func (w *webAccount) Type() string {
	return w.AccountType
}

type driverContainer struct {
	Driver string
	Data   Connector
}

func (d *driverContainer) UnmarshalJSON(b []byte) error {
	var driver struct {
		Driver string
		Data   *json.RawMessage
	}
	if err := json.Unmarshal(b, &driver); err != nil {
		return err
	}
	d.Driver = driver.Driver
	if driver.Data == nil {
		return nil
	}
	var creds credConnector
	if err := json.Unmarshal(*driver.Data, &creds); err != nil {
		return err
	}
	creds.driver = driver.Driver
	var err error
	d.Data, err = Connect(&creds)
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

type credConnector struct {
	driver string
	// embed all known credential types
	PasswordConnector *passwordConnector
}

func (m *credConnector) Driver() string {
	return m.driver
}

func (m *credConnector) MarshalJSON() ([]byte, error) {
	switch {
	case m.PasswordConnector != nil:
		return json.Marshal(m.PasswordConnector)
	default:
		return nil, errors.Errorf("No contained credentials: %+v", m)
	}
}

func (m *credConnector) Username() string {
	return m.PasswordConnector.Username()
}

func (m *credConnector) Password() redactor.String {
	return m.PasswordConnector.Password()
}
func (m *credConnector) SetPassword(p redactor.String) {
	m.PasswordConnector.SetPassword(p)
}
