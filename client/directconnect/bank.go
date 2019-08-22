package directconnect

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/client/model"
	sErrors "github.com/johnstarich/sage/errors"
)

type bankAccount struct {
	directAccount
	AccountType   string
	RoutingNumber string
}

// Bank is an account with a bank's routing number or 'bank ID'
type Bank interface {
	model.Account

	BankID() string
}

const (
	checkingType = "CHECKING"
	savingsType  = "SAVINGS"
)

// NewCheckingAccount creates an account from checking details
func NewCheckingAccount(id, bankID, description string, institution Connector) Account {
	return newBankAccount(checkingType, id, bankID, description, institution)
}

// NewSavingsAccount creates an account from savings details
func NewSavingsAccount(id, bankID, description string, institution Connector) Account {
	return newBankAccount(savingsType, id, bankID, description, institution)
}

func newBankAccount(accountType, id, bankID, description string, connector Connector) Account {
	return &bankAccount{
		AccountType:   strings.ToUpper(accountType),
		RoutingNumber: bankID,
		directAccount: directAccount{
			AccountID:          id,
			AccountDescription: description,
			DirectConnect:      connector,
		},
	}
}

func (b *bankAccount) BankID() string {
	return b.RoutingNumber
}

func (b *bankAccount) isBank() bool {
	return b.RoutingNumber != ""
}

func (b *bankAccount) Validate() error {
	var errs sErrors.Errors
	errs.AddErr(b.directAccount.Validate())
	errs.ErrIf(b.RoutingNumber == "", "Routing number must not be empty")
	errs.ErrIf(b.AccountType != checkingType && b.AccountType != savingsType, "Account type must be %s or %s", checkingType, savingsType)
	return errs.ErrOrNil()
}

// Statement implements Requestor
func (b *bankAccount) Statement(req *ofxgo.Request, start, end time.Time) error {
	return generateBankStatement(b, req, start, end, b.AccountType, ofxgo.RandomUID)
}

func generateBankStatement(
	b *bankAccount,
	req *ofxgo.Request,
	start, end time.Time,
	accountType string,
	getUID func() (*ofxgo.UID, error),
) error {
	uid, err := getUID()
	if err != nil {
		return err
	}

	accountTypeEnum, err := ofxgo.NewAcctType(accountType)
	if err != nil {
		return err
	}

	req.Bank = append(req.Bank, &ofxgo.StatementRequest{
		TrnUID: *uid,
		BankAcctFrom: ofxgo.BankAcct{
			BankID:   ofxgo.String(b.RoutingNumber),
			AcctID:   ofxgo.String(b.ID()),
			AcctType: accountTypeEnum,
		},
		DtStart: &ofxgo.Date{Time: start},
		DtEnd:   &ofxgo.Date{Time: end},
		Include: true, // Include transactions (instead of only balance information)
	})
	return nil
}

func (b *bankAccount) Type() string {
	return model.AssetAccount
}

func (b *bankAccount) UnmarshalJSON(data []byte) error {
	var bank struct {
		AccountType   string
		RoutingNumber string
	}

	if err := json.Unmarshal(data, &bank); err != nil {
		return err
	}

	b.AccountType = bank.AccountType
	b.RoutingNumber = bank.RoutingNumber
	return json.Unmarshal(data, &b.directAccount)
}

/*
// Bank is an Account plus the Bank's routing number or 'Bank ID'. Common interface for Savings and Checking
type Bank struct {
	Account

	BankID() string
}

func (b bankAccount) BankID() string {
	return b.bankID
}
*/

/*
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
*/
/*
// Checking represents a checking bank account
type Checking struct {
	bankAccount
}

// Savings represents a savings bank account
type Savings struct {
	bankAccount
}

// IsChecking returns true if the given account type is a checking account
func IsChecking(s string) bool {
	return s == checkingType
}

// IsSavings returns true if the given account type is a savings account
func IsSavings(s string) bool {
	return s == savingsType
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
*/
