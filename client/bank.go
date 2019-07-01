package client

import (
	"encoding/json"
	"time"

	"github.com/aclindsa/ofxgo"
)

// Bank is an Account plus the Bank's routing number or 'Bank ID'. Common interface for Savings and Checking
type Bank interface {
	Account

	BankID() string
}

type bankAccount struct {
	baseAccount
	bankID string
}

func (b bankAccount) BankID() string {
	return b.bankID
}

func (b bankAccount) marshalJSON(accountType string) ([]byte, error) {
	return json.Marshal(struct {
		Description   string
		ID            string
		AccountType   string
		RoutingNumber string
		Institution   Institution
	}{
		AccountType:   accountType,
		Description:   b.description,
		ID:            b.id,
		Institution:   b.institution,
		RoutingNumber: b.bankID,
	})
}

// Checking represents a checking bank account
type Checking struct {
	bankAccount
}

// Savings represents a savings bank account
type Savings struct {
	bankAccount
}

const (
	checkingType = "CHECKING"
	savingsType  = "SAVINGS"
)

// NewCheckingAccount creates an account from checking details
func NewCheckingAccount(id, bankID, description string, institution Institution) Account {
	return Checking{bankAccount: newBankAccount(id, bankID, description, institution)}
}

// NewSavingsAccount creates an account from savings details
func NewSavingsAccount(id, bankID, description string, institution Institution) Account {
	return Savings{bankAccount: newBankAccount(id, bankID, description, institution)}
}

func newBankAccount(id, bankID, description string, institution Institution) bankAccount {
	return bankAccount{
		bankID: bankID,
		baseAccount: baseAccount{
			id:          id,
			description: description,
			institution: institution,
		},
	}
}

func (b bankAccount) statementFromAccountType(start, end time.Time, accountType string) (ofxgo.Request, error) {
	return generateBankStatement(b, start, end, accountType, ofxgo.RandomUID)
}

func generateBankStatement(
	b bankAccount,
	start, end time.Time,
	accountType string,
	getUID func() (*ofxgo.UID, error),
) (ofxgo.Request, error) {
	uid, err := getUID()
	if err != nil {
		return ofxgo.Request{}, err
	}

	accountTypeEnum, err := ofxgo.NewAcctType(accountType)
	if err != nil {
		return ofxgo.Request{}, err
	}
	return ofxgo.Request{
		Bank: []ofxgo.Message{
			&ofxgo.StatementRequest{
				TrnUID: *uid,
				BankAcctFrom: ofxgo.BankAcct{
					BankID:   ofxgo.String(b.BankID()),
					AcctID:   ofxgo.String(b.ID()),
					AcctType: accountTypeEnum,
				},
				DtStart: &ofxgo.Date{Time: start},
				DtEnd:   &ofxgo.Date{Time: end},
				Include: true, // Include transactions (instead of only balance information)
			},
		},
	}, nil
}

func (c Checking) Statement(start, end time.Time) (ofxgo.Request, error) {
	return c.statementFromAccountType(start, end, checkingType)
}

func (s Savings) Statement(start, end time.Time) (ofxgo.Request, error) {
	return s.statementFromAccountType(start, end, savingsType)
}

func (c Checking) MarshalJSON() ([]byte, error) {
	return c.bankAccount.marshalJSON(checkingType)
}

func (s Savings) MarshalJSON() ([]byte, error) {
	return s.bankAccount.marshalJSON(savingsType)
}
