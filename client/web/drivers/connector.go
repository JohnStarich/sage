package drivers

import (
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/redactor"
)

// Connector downloads statements from an institution's website
type Connector interface {
	model.Institution
	CredConnector
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

type passwordConnector struct {
	InstDescription   string
	ConnectorUsername string
	ConnectorPassword redactor.String
}

func (p *passwordConnector) Username() string {
	return p.ConnectorUsername
}

func (p *passwordConnector) Password() redactor.String {
	return p.ConnectorPassword
}

/*
// ideas for future connector types:

type TOTPConnector interface {
	CredConnector

	Username() string
	Seed()     redactor.String
}

*/
