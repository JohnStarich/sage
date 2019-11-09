package web

import (
	"time"

	"github.com/aclindsa/ofxgo"

	"github.com/johnstarich/sage/client/model"
	sErrors "github.com/johnstarich/sage/errors"
	"github.com/johnstarich/sage/redactor"
)

// Connector downloads statements from an institution's website
type Connector interface {
	model.Institution
	CredConnector
	Requestor
}

// Requestor downloads statements from an institution's website
type Requestor interface {
	// Statement downloads transactions with browser between the start and end times
	Statement(browser Browser, start, end time.Time) (*ofxgo.Response, error)
}

// CredConnector is used by a Driver to create a full Connector
type CredConnector interface {
	// Driver is the name of the driver
	Driver() string
}

type PasswordConnector interface {
	CredConnector

	Username() string
	Password() redactor.String
	SetPassword(redactor.String)
}

/*
// ideas for future connector types:

type TOTPConnector interface {
	CredConnector

	Username() string
	Seed()     redactor.String
}

*/

type passwordConnector struct {
	DriverName        string
	ConnectorUsername string
	ConnectorPassword redactor.String
}

func (p *passwordConnector) Driver() string {
	return p.DriverName
}

func (p *passwordConnector) Username() string {
	return p.ConnectorUsername
}

func (p *passwordConnector) Password() redactor.String {
	return p.ConnectorPassword
}

func (p *passwordConnector) SetPassword(password redactor.String) {
	p.ConnectorPassword = password
}

func Validate(account Account) error {
	var errs sErrors.Errors
	errs.AddErr(model.ValidateAccount(account))
	inst := account.Institution()
	connector, ok := inst.(Connector)
	if !ok {
		return errs.ErrOrNil()
	}
	if passConnector, ok := connector.(PasswordConnector); ok {
		errs.ErrIf(passConnector.Username() == "", "Institution username must not be empty")
		errs.ErrIf(passConnector.Password() == "", "Institution password must not be empty")
	}
	return errs.ErrOrNil()
}
