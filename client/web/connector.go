package web

import (
	"context"
	"time"

	"github.com/aclindsa/ofxgo"

	"github.com/johnstarich/sage/client/model"
	sErrors "github.com/johnstarich/sage/errors"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/prompter"
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
	Statement(start, end time.Time, accountID string, browser Browser, prompt prompter.Prompter) (*ofxgo.Response, error)
}

// CredConnector is used by a Driver to create a full Connector
type CredConnector interface {
	// Driver is the name of the driver
	Driver() string
}

// PasswordConnector contains credentials for user/pass authentication
type PasswordConnector interface {
	CredConnector

	Username() string
	Password() redactor.String
	SetPassword(redactor.String)
}

// ConnectorValidator performs validation on an account / credential pair, returns any errors
type ConnectorValidator interface {
	Validate(accountID string) error
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
	ConnectorUsername string
	ConnectorPassword redactor.String
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

// Validate checks account for bad values
func Validate(account Account) error {
	var errs sErrors.Errors
	errs.AddErr(model.ValidateAccount(account))
	inst := account.Institution()
	connector, ok := inst.(Connector)
	if !ok {
		return errs.ErrOrNil()
	}
	if validator, ok := connector.(ConnectorValidator); ok {
		errs.AddErr(validator.Validate(account.ID()))
	}
	if passConnector, ok := connector.(PasswordConnector); ok {
		errs.ErrIf(passConnector.Username() == "", "Institution username must not be empty")
		errs.ErrIf(passConnector.Password() == "", "Institution password must not be empty")
	}
	return errs.ErrOrNil()
}

// Statement downloads and returns transactions from a connector for the given time period
func Statement(connector Connector, start, end time.Time, accountIDs []string, parser model.TransactionParser, prompt prompter.Prompter) ([]ledger.Transaction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	browser, err := NewBrowser(ctx, BrowserConfig{Record: true, NoHeadless: true})
	if err != nil {
		return nil, err
	}
	return fetchTransactions(
		connector,
		start, end,
		accountIDs,
		browser,
		prompt,
		parser,
	)
}

func fetchTransactions(
	connector Connector,
	start, end time.Time,
	accountIDs []string,
	browser Browser,
	prompt prompter.Prompter,
	parser model.TransactionParser,
) ([]ledger.Transaction, error) {
	var allTxns []ledger.Transaction
	for _, account := range accountIDs {
		resp, err := connector.Statement(start, end, account, browser, prompt)
		if err != nil {
			return allTxns, err
		}
		_, txns, err := parser(resp)
		allTxns = append(allTxns, txns...)
		if err != nil {
			return allTxns, err
		}
	}
	return allTxns, nil
}
