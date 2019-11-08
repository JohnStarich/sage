package web

import (
	"time"

	"github.com/aclindsa/ofxgo"

	"github.com/johnstarich/sage/client/model"
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
