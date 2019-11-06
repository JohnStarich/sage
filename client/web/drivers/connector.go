package drivers

import "github.com/johnstarich/sage/redactor"

// Connector contains user credentials to log into an institution
type Connector interface {
	Username() string
}

type PasswordConnector interface {
	Password() redactor.String
}

/*
// ideas for future connector types:

type AccountConnector interface {
	AccountID() string
}

type OneTimePasswordConnector interface {
	GenPassword() string
}

*/
