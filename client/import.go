package client

import (
	"fmt"
	"io"
	"strings"

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
		return nil, nil, errors.Errorf("No messages received")
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
			parsedTxn := parseTransaction(ofxTxn, currency, account.String(), makeUniqueTxnID(fid, account.AccountID))
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
func ReadOFX(r io.ReadCloser) ([]model.Account, []ledger.Transaction, error) {
	resp, err := ofxgo.ParseResponse(r)
	if err != nil {
		return nil, nil, err
	}
	if err := r.Close(); err != nil {
		return nil, nil, err
	}
	return importTransactions(resp, parseTransaction)
}

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

func makeUniqueTxnID(fid, accountID string) func(txnID string) string {
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
