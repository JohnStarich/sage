package direct

import (
	"fmt"
	"io"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/ledger"
	"github.com/pkg/errors"
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
func ReadOFX(r io.Reader) ([]model.Account, []ledger.Transaction, error) {
	resp, err := ofxgo.ParseResponse(r)
	if err != nil {
		return nil, nil, err
	}
	return importTransactions(resp, parseTransaction)
}
