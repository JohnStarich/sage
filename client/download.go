package client

import (
	"fmt"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/pkg/errors"
)

func Transactions(a Account, duration time.Duration) error {
	query, err := a.Statement(duration)
	if err != nil {
		return err
	}
	if len(query.Bank) == 0 && len(query.CreditCard) == 0 {
		return errors.Errorf("Invalid statement query: does not contain any statement requests: %+v", query)
	}

	institution := a.Institution()
	config := institution.Config()

	ofxClient, err := New(institution.URL(), config)
	if err != nil {
		return err
	}

	query.URL = institution.URL()
	query.Signon = ofxgo.SignonRequest{
		ClientUID: ofxgo.UID(config.ClientID),
		Org:       ofxgo.String(institution.Org()),
		Fid:       ofxgo.String(institution.FID()),
		UserID:    ofxgo.String(institution.Username()),
		UserPass:  ofxgo.String(institution.Password()),
	}

	response, err := ofxClient.Request(&query)
	if err != nil {
		return err
	}

	if response.Signon.Status.Code != 0 {
		meaning, err := response.Signon.Status.CodeMeaning()
		if err != nil {
			return errors.Wrap(err, "Failed to parse OFX response code")
		}
		return errors.Errorf("Nonzero signon status (%d: %s) with message: %s", response.Signon.Status.Code, meaning, response.Signon.Status.Message)
	}

	if len(query.Bank) > 0 {
		if len(response.Bank) == 0 {
			return errors.Errorf("No banking messages received")
		}

		if stmt, ok := response.Bank[0].(*ofxgo.StatementResponse); ok {
			fmt.Printf("Balance: %s %s (as of %s)\n", stmt.BalAmt, stmt.CurDef, stmt.DtAsOf)
			fmt.Println("Transactions:")
			for _, tran := range stmt.BankTranList.Transactions {
				currency := stmt.CurDef
				if tran.Currency != nil {
					if ok, _ := tran.Currency.Valid(); ok {
						currency = tran.Currency.CurSym
					}
				}

				var name string
				if len(tran.Name) > 0 {
					name = string(tran.Name)
				} else {
					name = string(tran.Payee.Name)
				}

				if len(tran.Memo) > 0 {
					name = name + " - " + string(tran.Memo)
				}

				fmt.Printf("%s %-15s %-11s %s\n", tran.DtPosted, tran.TrnAmt.String()+" "+currency.String(), tran.TrnType, name)
			}
		}
		return nil
	}

	if len(query.CreditCard) > 0 {
		if len(response.CreditCard) == 0 {
			return errors.Errorf("No credit card messages received")
		}

		if stmt, ok := response.CreditCard[0].(*ofxgo.CCStatementResponse); ok {
			fmt.Printf("Balance: %s %s (as of %s)\n", stmt.BalAmt, stmt.CurDef, stmt.DtAsOf)
			fmt.Println("Transactions:")
			for _, tran := range stmt.BankTranList.Transactions {
				currency := stmt.CurDef
				if tran.Currency != nil {
					if ok, _ := tran.Currency.Valid(); ok {
						currency = tran.Currency.CurSym
					}
				}

				var name string
				if len(tran.Name) > 0 {
					name = string(tran.Name)
				} else {
					name = string(tran.Payee.Name)
				}

				if len(tran.Memo) > 0 {
					name = name + " - " + string(tran.Memo)
				}

				fmt.Printf("%s %-15s %-11s %s\n", tran.DtPosted, tran.TrnAmt.String()+" "+currency.String(), tran.TrnType, name)
			}
		}
		return nil
	}

	return errors.Errorf("Unknown account type: %+v", a)
}
