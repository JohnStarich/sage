package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/ledger"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

// Transactions downloads and returns transactions from a bank or credit card account for the given time period, ending today
func Transactions(account Account, start, end time.Time) ([]ledger.Transaction, error) {
	client, err := clientForInstitution(account.Institution())
	if err != nil {
		return nil, err
	}

	return fetchTransactions(
		account,
		start, end,
		balanceTransactions,
		client.Request,
		parseTransaction,
	)
}

func fetchTransactions(
	account Account,
	start, end time.Time,
	balanceTransactions func([]ledger.Transaction, decimal.Decimal, time.Time, time.Time),
	doRequest func(*ofxgo.Request) (*ofxgo.Response, error),
	parseTransaction func(ofxgo.Transaction, string, string, func(string) string) ledger.Transaction,
) ([]ledger.Transaction, error) {
	query, err := account.Statement(start, end)
	if err != nil {
		return nil, err
	}
	if len(query.Bank) == 0 && len(query.CreditCard) == 0 {
		return nil, errors.Errorf("Invalid statement query: does not contain any statement requests: %+v", query)
	}

	accountName := LedgerAccountName(account)
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

	makeTxnID := makeUniqueTxnID(account)
	var txns []ledger.Transaction

	for _, message := range statements {
		var balance decimal.Decimal
		var balanceCurrency string
		var balanceDate, statementEndDate time.Time
		var statementTxns []ofxgo.Transaction
		switch statement := message.(type) {
		case *ofxgo.StatementResponse:
			balance, err = decimal.NewFromString(statement.BalAmt.String())
			if err != nil {
				return nil, err
			}
			balanceCurrency = normalizeCurrency(statement.CurDef.String())
			balanceDate = statement.DtAsOf.Time
			statementEndDate = statement.BankTranList.DtEnd.Time
			statementTxns = statement.BankTranList.Transactions
		case *ofxgo.CCStatementResponse:
			balance, err = decimal.NewFromString(statement.BalAmt.String())
			if err != nil {
				return nil, err
			}
			balanceCurrency = normalizeCurrency(statement.CurDef.String())
			balanceDate = statement.DtAsOf.Time
			statementEndDate = statement.BankTranList.DtEnd.Time
			statementTxns = statement.BankTranList.Transactions
		default:
			return nil, fmt.Errorf("Invalid statement type: %T", message)
		}

		for _, txn := range statementTxns {
			parsedTxn := parseTransaction(txn, balanceCurrency, accountName, makeTxnID)
			txns = append(txns, parsedTxn)
		}

		balanceTransactions(txns, balance, balanceDate, statementEndDate)
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

func parseTransaction(txn ofxgo.Transaction, currency, accountName string, makeTxnID func(string) string) ledger.Transaction {
	if txn.Currency != nil {
		if ok, _ := txn.Currency.Valid(); ok {
			currency = normalizeCurrency(txn.Currency.CurSym.String())
		}
	}

	name := string(txn.Name)
	if name == "" && txn.Payee != nil {
		name = string(txn.Payee.Name)
	}

	// TODO can ofxgo lib support a decimal type instead of big.Rat?
	// NOTE: TrnAmt uses big.Rat internally, which can't form an invalid number with .String()
	amount := decimal.RequireFromString(txn.TrnAmt.String())

	id := makeTxnID(string(txn.FiTID))

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
	}
}

// balanceTransactions sorts and adds balances to each transaction
func balanceTransactions(txns []ledger.Transaction, balance decimal.Decimal, balanceDate time.Time, statementEndDate time.Time) {
	ledger.Transactions(txns).Sort()

	if balanceDate.After(statementEndDate) {
		// don't trust this balance, it was recorded after the statement end date
		return
	}

	balanceDateIndex := len(txns)
	for i, txn := range txns {
		if txn.Date.After(balanceDate) {
			// the end of balance date
			balanceDateIndex = i
			break
		}
	}

	runningBalance := balance
	for i := balanceDateIndex - 1; i >= 0; i-- {
		txns[i].Postings[0].Balance = decToPtr(runningBalance)
		runningBalance = runningBalance.Sub(txns[i].Postings[0].Amount)
	}
	runningBalance = balance
	for i := balanceDateIndex; i < len(txns); i++ {
		runningBalance = runningBalance.Add(txns[i].Postings[0].Amount)
		txns[i].Postings[0].Balance = decToPtr(runningBalance)
	}
}

func makeUniqueTxnID(account Account) func(string) string {
	institution := account.Institution()
	// Follows FITID recommendation from OFX 102 Section 3.2.1
	idPrefix := institution.FID() + "-" + account.ID() + "-"
	return func(txnID string) string {
		id := idPrefix + txnID
		// clean ID for use as an hledger tag
		// TODO move tag (de)serialization into ledger package
		id = strings.Replace(id, ",", "", -1)
		id = strings.Replace(id, ":", "", -1)
		return id
	}
}
