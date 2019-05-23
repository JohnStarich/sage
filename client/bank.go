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

func NewCheckingAccount(id, bankID string, institution Institution) Account {
	return Checking{bankAccount: newBankAccount(id, bankID, institution)}
}

func NewSavingsAccount(id, bankID string, institution Institution) Account {
	return Savings{bankAccount: newBankAccount(id, bankID, institution)}
}

func newBankAccount(id, bankID string, institution Institution) bankAccount {
	return bankAccount{
		bankID: bankID,
		baseAccount: baseAccount{
			id:          id,
			institution: institution,
		},
	}
}

func (b bankAccount) statementFromAccountType(duration time.Duration, accountType string) (ofxgo.Request, error) {
	return generateBankStatement(b, duration, accountType, ofxgo.RandomUID, time.Now)
}

func generateBankStatement(
	b bankAccount,
	duration time.Duration, accountType string,
	getUID func() (*ofxgo.UID, error),
	getTime func() time.Time,
) (ofxgo.Request, error) {
	uid, err := getUID()
	if err != nil {
		return ofxgo.Request{}, err
	}

	end := getTime()
	start := end.Add(-duration)
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

func (c Checking) Statement(duration time.Duration) (ofxgo.Request, error) {
	return c.statementFromAccountType(duration, checkingType)
}

func (s Savings) Statement(duration time.Duration) (ofxgo.Request, error) {
	return s.statementFromAccountType(duration, savingsType)
}
