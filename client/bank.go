package client

import (
	"time"

	"github.com/aclindsa/ofxgo"
)

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

type Checking struct {
	bankAccount
}

type Savings struct {
	bankAccount
}

const (
	checkingType = "CHECKING"
	savingsType  = "SAVINGS"
)

func NewCheckingAccount(id, bankID, description string, institution Institution) Account {
	return Checking{bankAccount: newBankAccount(id, bankID, description, institution)}
}

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
