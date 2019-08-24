package directconnect

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/client/model"
)

type accountType int

const (
	// CheckingType refers to a bank checking account
	CheckingType accountType = iota + 1
	// SavingsType refers to a bank savings account
	SavingsType
)

// ParseAccountType parses s as a bank account type, like checking or savings
func ParseAccountType(s string) accountType { // nolint - intentionally do not allow custom account types
	switch strings.ToUpper(s) {
	case CheckingType.String():
		return CheckingType
	case SavingsType.String():
		return SavingsType
	default:
		return 0
	}
}

func (a accountType) String() string {
	switch a {
	case CheckingType:
		return "CHECKING"
	case SavingsType:
		return "SAVINGS"
	default:
		return ""
	}
}

type bankAccount struct {
	directAccount
	BankAccountType string
	RoutingNumber   string
}

// Bank is an account with a bank's routing number or 'bank ID'
type Bank interface {
	model.Account

	BankID() string
}

// NewCheckingAccount creates an account from checking details
func NewCheckingAccount(id, bankID, description string, institution Connector) Account {
	return newBankAccount(CheckingType, id, bankID, description, institution)
}

// NewSavingsAccount creates an account from savings details
func NewSavingsAccount(id, bankID, description string, institution Connector) Account {
	return newBankAccount(SavingsType, id, bankID, description, institution)
}

func newBankAccount(kind accountType, id, bankID, description string, connector Connector) Account {
	return &bankAccount{
		BankAccountType: kind.String(),
		RoutingNumber:   bankID,
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

// Statement implements Requestor
func (b *bankAccount) Statement(req *ofxgo.Request, start, end time.Time) error {
	return generateBankStatement(b, req, start, end, b.BankAccountType, ofxgo.RandomUID)
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
		BankAccountType string
		RoutingNumber   string
	}

	if err := json.Unmarshal(data, &bank); err != nil {
		return err
	}

	b.BankAccountType = bank.BankAccountType
	b.RoutingNumber = bank.RoutingNumber
	return json.Unmarshal(data, &b.directAccount)
}
