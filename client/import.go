package client

import (
	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/ledger"
	"github.com/pkg/errors"
)

func importTransactions(
	account Account,
	resp *ofxgo.Response,
	parseTransaction transactionParser,
) ([]ledger.Transaction, error) {
	accountName := LedgerAccountName(account)
	messages := append(resp.Bank, resp.CreditCard...)
	if len(messages) == 0 {
		return nil, errors.Errorf("No messages received")
	}

	var txns []ledger.Transaction
	for _, message := range messages {
		var ofxTxns []ofxgo.Transaction
		var currency string
		switch statement := message.(type) {
		case *ofxgo.CCStatementResponse:
			if statement.BankTranList != nil {
				ofxTxns = statement.BankTranList.Transactions
			}
			currency = normalizeCurrency(statement.CurDef.String())
		case *ofxgo.StatementResponse:
			if statement.BankTranList != nil {
				ofxTxns = statement.BankTranList.Transactions
			}
			currency = normalizeCurrency(statement.CurDef.String())
		default:
			return nil, errors.Errorf("Invalid statement type: %T", message)
		}

		for _, ofxTxn := range ofxTxns {
			parsedTxn := parseTransaction(ofxTxn, currency, accountName, makeUniqueTxnID(account))
			txns = append(txns, parsedTxn)
		}
	}
	return txns, nil
}
