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

// Transactions downloads and returns transactions from a bank or credit card account for the given time period, ending today
func Transactions(account Account, duration time.Duration) ([]ledger.Transaction, error) {
	institution := account.Institution()
	client, err := New(institution.URL(), institution.Config())
	if err != nil {
		return nil, err
	}

	return transactions(account, duration, client.Request)
}

func transactions(
	account Account, duration time.Duration,
	doRequest func(*ofxgo.Request) (*ofxgo.Response, error),
) ([]ledger.Transaction, error) {
	query, err := account.Statement(duration)
	if err != nil {
		return nil, err
	}
	if len(query.Bank) == 0 && len(query.CreditCard) == 0 {
		return nil, errors.Errorf("Invalid statement query: does not contain any statement requests: %+v", query)
	}

	institution := account.Institution()
	config := institution.Config()

	query.URL = institution.URL()
	query.Signon = ofxgo.SignonRequest{
		ClientUID: ofxgo.UID(config.ClientID),
		Org:       ofxgo.String(institution.Org()),
		Fid:       ofxgo.String(institution.FID()),
		UserID:    ofxgo.String(institution.Username()),
		UserPass:  ofxgo.String(institution.Password()),
	}

	response, err := doRequest(&query)
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

		accountName := LedgerAccountName(account)
		for _, txn := range statementTxns {
			parsedTxn, err := parseTransaction(txn, balanceCurrency, account, accountName)
			if err != nil {
				return nil, err
			}
			txns = append(txns, parsedTxn)
		}

		balanceTransactions(txns, balance, balanceDate)
	}

	return txns, nil
}

// decToPtr makes a copy of d and returns a reference to it
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

func parseTransaction(txn ofxgo.Transaction, balanceCurrency string, account Account, accountName string) (ledger.Transaction, error) {
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
		return ledger.Transaction{}, err
	}

	institution := account.Institution()
	// Follows FITID recommendation from OFX 102 Section 3.2.1
	id := institution.FID() + "-" + account.ID() + "-" + string(txn.FiTID)
	// clean ID for use as an hledger tag
	// TODO move tag (de)serialization into ledger package
	id = strings.Replace(id, ",", "_", -1)
	id = strings.Replace(id, ":", "_", -1)

	return ledger.Transaction{
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
	}, nil
}

// balanceTransactions sorts and adds balances to each transaction
func balanceTransactions(txns []ledger.Transaction, balance decimal.Decimal, balanceDate time.Time) {
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
