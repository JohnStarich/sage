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
