package client

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/ledger"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

func Transactions(a Account, duration time.Duration) ([]ledger.Transaction, error) {
	query, err := a.Statement(duration)
	if err != nil {
		return nil, err
	}
	if len(query.Bank) == 0 && len(query.CreditCard) == 0 {
		return nil, errors.Errorf("Invalid statement query: does not contain any statement requests: %+v", query)
	}

	institution := a.Institution()
	config := institution.Config()

	ofxClient, err := New(institution.URL(), config)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	if response.Signon.Status.Code != 0 {
		meaning, err := response.Signon.Status.CodeMeaning()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse OFX response code")
		}
		return nil, errors.Errorf("Nonzero signon status (%d: %s) with message: %s", response.Signon.Status.Code, meaning, response.Signon.Status.Message)
	}

	statements := append(response.Bank, response.CreditCard...)
	if len(statements) == 0 {
		return nil, errors.Errorf("No messages received")
	}

	accountName := LedgerAccountName(a)
	var txns []ledger.Transaction

	for _, message := range statements {
		var balance decimal.Decimal
		var balanceCurrency string
		var balanceDate time.Time
		var statementTxns []ofxgo.Transaction
		switch statement := message.(type) {
		case *ofxgo.StatementResponse:
			balance, err = decimal.NewFromString(statement.BalAmt.String())
			if err != nil {
				return nil, err
			}
			balanceCurrency = normalizeCurrency(statement.CurDef.String())
			balanceDate = statement.DtAsOf.Time
			statementTxns = statement.BankTranList.Transactions
		case *ofxgo.CCStatementResponse:
			balance, err = decimal.NewFromString(statement.BalAmt.String())
			if err != nil {
				return nil, err
			}
			balanceCurrency = normalizeCurrency(statement.CurDef.String())
			balanceDate = statement.DtAsOf.Time
			statementTxns = statement.BankTranList.Transactions
		default:
			return nil, fmt.Errorf("Invalid statement type: %T", message)
		}

		for _, txn := range statementTxns {
			currency := balanceCurrency
			if txn.Currency != nil {
				if ok, _ := txn.Currency.Valid(); ok {
					currency = normalizeCurrency(txn.Currency.CurSym.String())
				}
			}

			var name string
			if len(txn.Name) > 0 {
				name = string(txn.Name)
			} else {
				name = string(txn.Payee.Name)
			}

			// TODO poke ofxgo lib maintainers to support a decimal type here. 100 bits of precision is good, but there's no reason it can't be the exact same value the institution returned.
			amount, err := decimal.NewFromString(txn.TrnAmt.String())
			if err != nil {
				return nil, err
			}

			// Follows FITID recommendation from OFX 102 Section 3.2.1
			id := institution.FID() + "-" + a.ID() + "-" + string(txn.FiTID)
			// clean ID for use as an hledger tag
			// TODO move tag (de)serialization into ledger package
			id = strings.Replace(id, ",", "_", -1)
			id = strings.Replace(id, ":", "_", -1)

			txns = append(txns, ledger.Transaction{
				Date:  txn.DtPosted.Time,
				Payee: name,
				Postings: []ledger.Posting{
					{
						Account:  accountName,
						Amount:   amount,
						Balance:  nil, // set balance in next section
						Currency: currency,
						Tags:     map[string]string{"id": id},
					},
					{
						Account:  "uncategorized",
						Amount:   amount.Neg(),
						Currency: currency,
					},
				},
			})
		}

		sort.SliceStable(txns, func(a, b int) bool {
			return txns[a].Date.Before(txns[b].Date)
		})

		balanceDateIndex := 0
		for i, txn := range txns {
			if !balanceDate.Before(txn.Date) {
				balanceDateIndex = i
				break
			}
		}

		runningBalance := balance
		for i := range txns[0:balanceDateIndex] {
			i = len(txns) - 1 - i // reverse index
			txns[i].Postings[0].Balance = decToPtr(runningBalance)
			runningBalance = runningBalance.Sub(txns[i].Postings[0].Amount)
		}
		runningBalance = balance
		for i := range txns[balanceDateIndex:] {
			runningBalance = runningBalance.Add(txns[i].Postings[0].Amount)
			txns[i].Postings[0].Balance = decToPtr(runningBalance)
		}
	}

	return txns, nil
}

func decToPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

func normalizeCurrency(currency string) string {
	switch currency {
	case "USD":
		return "$"
	default:
		return currency
	}
}
