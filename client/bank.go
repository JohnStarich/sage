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

type bankAccountJSON struct {
	Description   string
	ID            string
	AccountType   string
	RoutingNumber string
	Institution   json.RawMessage
}

func (b *bankAccount) UnmarshalJSON(buf []byte) error {
	var account bankAccountJSON
	if err := json.Unmarshal(buf, &account); err != nil {
		return err
	}
	b.id = account.ID
	b.description = account.Description
	b.bankID = account.RoutingNumber
	return json.Unmarshal([]byte(account.Institution), &b.institution)
}

func (b bankAccount) marshalJSON(accountType string, withPassword bool) ([]byte, error) {
	var instData json.RawMessage
	var err error
	if withPassword {
		instData, err = b.institution.MarshalWithPassword()
	} else {
		instData, err = json.Marshal(b.institution)
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(bankAccountJSON{
		AccountType:   accountType,
		Description:   b.description,
		ID:            b.id,
		Institution:   instData,
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

// IsChecking returns true if the given account type is a checking account
func IsChecking(s string) bool {
	return s == checkingType
}

// IsSavings returns true if the given account type is a savings account
func IsSavings(s string) bool {
	return s == savingsType
}

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
			institution: newBaseFromInterface(institution),
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

// Statement fetches a statement for a checking account
func (c Checking) Statement(start, end time.Time) (ofxgo.Request, error) {
	return c.statementFromAccountType(start, end, checkingType)
}

// Statement fetches a statement for a savings account
func (s Savings) Statement(start, end time.Time) (ofxgo.Request, error) {
	return s.statementFromAccountType(start, end, savingsType)
}

// MarshalJSON marshals a checking account
func (c Checking) MarshalJSON() ([]byte, error) {
	return c.bankAccount.marshalJSON(checkingType, false)
}

// MarshalWithPassword marshals a checking account and includes the password
func (c Checking) MarshalWithPassword() ([]byte, error) {
	return c.bankAccount.marshalJSON(checkingType, true)
}

// MarshalJSON marshals a savings account
func (s Savings) MarshalJSON() ([]byte, error) {
	return s.bankAccount.marshalJSON(savingsType, false)
}

// MarshalWithPassword marshals a savings account and includes the password
func (s Savings) MarshalWithPassword() ([]byte, error) {
	return s.bankAccount.marshalJSON(savingsType, true)
}
