package client

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/ledger"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

func importTransactions(
	resp *ofxgo.Response,
	parseTransaction transactionParser,
) (skeletonAccounts []model.Account, allTxns []ledger.Transaction, importErr error) {
	messages := append(resp.Bank, resp.CreditCard...)
	if len(messages) == 0 {
		return nil, nil, errors.New("No messages received")
	}
	fid := resp.Signon.Fid.String()
	org := resp.Signon.Org.String()

	var txns []ledger.Transaction
	for _, message := range messages {
		var ofxTxns []ofxgo.Transaction
		var currency string
		account := model.LedgerAccountFormat{Institution: org}
		switch statement := message.(type) {
		case *ofxgo.CCStatementResponse:
			account.AccountType = model.LiabilityAccount
			account.AccountID = statement.CCAcctFrom.AcctID.String()
			if statement.BankTranList != nil {
				ofxTxns = statement.BankTranList.Transactions
			}
			currency = normalizeCurrency(statement.CurDef.String())
		case *ofxgo.StatementResponse:
			account.AccountType = model.AssetAccount
			account.AccountID = statement.BankAcctFrom.AcctID.String()
			if statement.BankTranList != nil {
				ofxTxns = statement.BankTranList.Transactions
			}
			currency = normalizeCurrency(statement.CurDef.String())
		default:
			return nil, nil, errors.Errorf("Invalid statement type: %T", message)
		}

		for _, ofxTxn := range ofxTxns {
			parsedTxn := parseTransaction(ofxTxn, currency, account.String(), MakeUniqueTxnID(fid, account.AccountID))
			txns = append(txns, parsedTxn)
		}

		skeletonAccounts = append(skeletonAccounts, &model.BasicAccount{
			AccountDescription: fmt.Sprintf("%s - %s", org, account.AccountID),
			AccountID:          account.AccountID,
			AccountType:        account.AccountType,
			BasicInstitution: model.BasicInstitution{
				InstDescription: org,
				InstFID:         fid,
				InstOrg:         org,
			},
		})
	}
	return skeletonAccounts, txns, nil
}

// ReadOFX reads r and parses it for an OFX file's transactions
func ReadOFX(r io.Reader) ([]model.Account, []ledger.Transaction, error) {
	resp, err := ofxgo.ParseResponse(r)
	if err != nil {
		return nil, nil, err
	}
	return ParseOFX(resp)
}

// ParseOFX parses the OFX response for its transactions
func ParseOFX(resp *ofxgo.Response) ([]model.Account, []ledger.Transaction, error) {
	return importTransactions(resp, parseTransaction)
}

type transactionParser func(txn ofxgo.Transaction, currency, accountName string, makeTxnID func(string) string) ledger.Transaction

func normalizeCurrency(currency string) string {
	switch currency {
	case "USD":
		return "$"
	default:
		return currency
	}
}

func MakeUniqueTxnID(fid, accountID string) func(txnID string) string {
	// Follows FITID recommendation from OFX 102 Section 3.2.1
	idPrefix := fid + "-" + accountID + "-"
	return func(txnID string) string {
		id := idPrefix + txnID
		// clean ID for use as an hledger tag
		// TODO move tag (de)serialization into ledger package
		id = strings.Replace(id, ",", "", -1)
		id = strings.Replace(id, ":", "", -1)
		return id
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
				Account:  model.Uncategorized,
				Amount:   amount.Neg(),
				Currency: currency,
			},
		},
	}
}

// balanceTransactions sorts and adds balances to each transaction
func balanceTransactions(txns []ledger.Transaction, balance decimal.Decimal, balanceDate time.Time, statementEndDate time.Time) {
	{
		// convert to ptrs, sort, then copy back results
		// TODO make more efficient should we add back auto-balances
		txnPtrs := make(ledger.Transactions, len(txns))
		for i := range txns {
			txn := txns[i] // copy txn
			txnPtrs[i] = &txn
		}
		txnPtrs.Sort()
		for i := range txnPtrs {
			txns[i] = *txnPtrs[i]
		}
	}

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

// decToPtr makes a copy of d and returns a reference to it
func decToPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}
