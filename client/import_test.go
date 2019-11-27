package client

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/client/testhelpers"
	"github.com/johnstarich/sage/ledger"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeOFXAmount(f float64) ofxgo.Amount {
	bigF := big.NewFloat(f)
	rat, _ := bigF.Rat(nil)
	return ofxgo.Amount{Rat: *rat}
}

func parseDate(date string) time.Time {
	d, err := time.Parse("2006/01/02", date)
	if err != nil {
		panic(err)
	}
	return d
}

func makeTxn(date string, amount float64) ledger.Transaction {
	txn := makeTxnWithBalance(date, amount, 0)
	txn.Postings[0].Balance = nil
	return txn
}

func makeTxnWithBalance(date string, amount, balance float64) ledger.Transaction {
	amountDec := decimal.NewFromFloat(amount)
	balanceDec := decimal.NewFromFloat(balance)
	return ledger.Transaction{
		Date: parseDate(date),
		Postings: []ledger.Posting{
			{Account: "assets:Bank 1", Amount: amountDec, Balance: &balanceDec},
			{Account: "expenses", Amount: amountDec.Neg()},
		},
	}
}

func TestImportTransactions(t *testing.T) {
	someTxn := ledger.Transaction{Comment: "some txn"}
	someCurrency, err := ofxgo.NewCurrSymbol("USD")
	require.NoError(t, err)
	for _, tc := range []struct {
		description    string
		resp           ofxgo.Response
		expectAccounts []model.Account
		expectTxns     []ledger.Transaction
		expectErr      string
	}{
		{
			description: "no response messages",
			expectErr:   "No messages received",
		},
		{
			description: "no bank txns",
			resp: ofxgo.Response{
				Signon: ofxgo.SignonResponse{
					Fid: ofxgo.String("some FID"),
					Org: ofxgo.String("some org"),
				},
				Bank: []ofxgo.Message{
					&ofxgo.StatementResponse{
						CurDef: *someCurrency,
						BankAcctFrom: ofxgo.BankAcct{
							AcctID:   ofxgo.String("1234"),
							AcctType: ofxgo.AcctTypeChecking,
						},
					},
				},
			},
			expectAccounts: []model.Account{
				&model.BasicAccount{
					AccountID:          "1234",
					AccountType:        model.AssetAccount,
					AccountDescription: "some org - 1234",
					BasicInstitution: model.BasicInstitution{
						InstDescription: "some org",
						InstFID:         "some FID",
						InstOrg:         "some org",
					},
				},
			},
		},
		{
			description: "bank txns",
			resp: ofxgo.Response{
				Signon: ofxgo.SignonResponse{
					Fid: ofxgo.String("some FID"),
					Org: ofxgo.String("some org"),
				},
				Bank: []ofxgo.Message{
					&ofxgo.StatementResponse{
						CurDef: *someCurrency,
						BankAcctFrom: ofxgo.BankAcct{
							AcctID:   ofxgo.String("1234"),
							AcctType: ofxgo.AcctTypeChecking,
						},
						BankTranList: &ofxgo.TransactionList{
							DtStart: ofxgo.Date{Time: time.Now()},
							DtEnd:   ofxgo.Date{Time: time.Now()},
							Transactions: []ofxgo.Transaction{
								{}, // value doesn't matter, goes through parser
							},
						},
					},
				},
			},
			expectAccounts: []model.Account{
				&model.BasicAccount{
					AccountID:          "1234",
					AccountType:        model.AssetAccount,
					AccountDescription: "some org - 1234",
					BasicInstitution: model.BasicInstitution{
						InstDescription: "some org",
						InstFID:         "some FID",
						InstOrg:         "some org",
					},
				},
			},
			expectTxns: []ledger.Transaction{someTxn},
		},
		{
			description: "no credit card txns",
			resp: ofxgo.Response{
				Signon: ofxgo.SignonResponse{
					Fid: ofxgo.String("some FID"),
					Org: ofxgo.String("some org"),
				},
				CreditCard: []ofxgo.Message{
					&ofxgo.CCStatementResponse{
						CurDef: *someCurrency,
						CCAcctFrom: ofxgo.CCAcct{
							AcctID: ofxgo.String("1234"),
						},
					},
				},
			},
			expectAccounts: []model.Account{
				&model.BasicAccount{
					AccountID:          "1234",
					AccountType:        model.LiabilityAccount,
					AccountDescription: "some org - 1234",
					BasicInstitution: model.BasicInstitution{
						InstDescription: "some org",
						InstFID:         "some FID",
						InstOrg:         "some org",
					},
				},
			},
		},
		{
			description: "credit card txns",
			resp: ofxgo.Response{
				Signon: ofxgo.SignonResponse{
					Fid: ofxgo.String("some FID"),
					Org: ofxgo.String("some org"),
				},
				CreditCard: []ofxgo.Message{
					&ofxgo.CCStatementResponse{
						CurDef: *someCurrency,
						CCAcctFrom: ofxgo.CCAcct{
							AcctID: ofxgo.String("1234"),
						},
						BankTranList: &ofxgo.TransactionList{
							DtStart: ofxgo.Date{Time: time.Now()},
							DtEnd:   ofxgo.Date{Time: time.Now()},
							Transactions: []ofxgo.Transaction{
								{}, // value doesn't matter, goes through parser
							},
						},
					},
				},
			},
			expectAccounts: []model.Account{
				&model.BasicAccount{
					AccountID:          "1234",
					AccountType:        model.LiabilityAccount,
					AccountDescription: "some org - 1234",
					BasicInstitution: model.BasicInstitution{
						InstDescription: "some org",
						InstFID:         "some FID",
						InstOrg:         "some org",
					},
				},
			},
			expectTxns: []ledger.Transaction{someTxn},
		},
		{
			description: "bad institution type",
			resp: ofxgo.Response{
				Signon: ofxgo.SignonResponse{
					Fid: ofxgo.String("some FID"),
					Org: ofxgo.String("some org"),
				},
				Bank: []ofxgo.Message{
					&ofxgo.ProfileResponse{},
				},
			},
			expectErr: "Invalid statement type: *ofxgo.ProfileResponse",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			parser := func(txn ofxgo.Transaction, currency, accountName string, makeTxnID func(string) string) ledger.Transaction {
				return someTxn
			}
			accounts, txns, err := importTransactions(&tc.resp, parser)
			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Equal(t, tc.expectErr, err.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectAccounts, accounts)
			assert.Equal(t, tc.expectTxns, txns)
		})
	}
}

func TestReadOFX(t *testing.T) {
	t.Run("no signon", func(t *testing.T) {
		_, _, err := ReadOFX(strings.NewReader(`
OFXHEADER:100
DATA:OFXSGML
VERSION:102

<OFX></OFX>`))
		require.Error(t, err)
		assert.Equal(t, "Missing opening SIGNONMSGSRSV1 xml element", err.Error())
	})

	t.Run("no transactions", func(t *testing.T) {
		_, _, err := ReadOFX(strings.NewReader(`
OFXHEADER:100
DATA:OFXSGML
VERSION:102

<OFX>
<SIGNONMSGSRSV1>
	<SONRS>
		<STATUS>
			<CODE>0
			<SEVERITY>INFO
		</STATUS>
		<LANGUAGE>ENG
		<FI>
			<ORG>SOMEORG
			<FID>SOMEFID
		</FI>
	</SONRS>
</SIGNONMSGSRSV1>
<BANKMSGSRSV1>
	<STMTTRNRS>
		<TRNUID>0
		<STATUS>
			<CODE>0
			<SEVERITY>INFO
		</STATUS>
	</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>
`))
		require.NoError(t, err)
	})
}

func TestParseOFX(t *testing.T) {
	resp := &ofxgo.Response{
		Signon: ofxgo.SignonResponse{
			Fid: ofxgo.String("some FID"),
			Org: ofxgo.String("some org"),
		},
		CreditCard: []ofxgo.Message{
			&ofxgo.CCStatementResponse{
				CCAcctFrom: ofxgo.CCAcct{
					AcctID: ofxgo.String("1234"),
				},
				BankTranList: &ofxgo.TransactionList{
					DtStart: ofxgo.Date{Time: time.Now()},
					DtEnd:   ofxgo.Date{Time: time.Now()},
					Transactions: []ofxgo.Transaction{
						{}, // value doesn't matter, goes through parser
					},
				},
			},
		},
	}
	accounts, txns, err := ParseOFX(resp)
	assert.NoError(t, err)
	// values tested in importTransactions test
	assert.NotEmpty(t, accounts)
	assert.NotEmpty(t, txns)
}

func TestNormalizeCurrency(t *testing.T) {
	assert.Equal(t, "$", normalizeCurrency("USD"))
	assert.Equal(t, "something else", normalizeCurrency("something else"))
}

func TestParseTransaction(t *testing.T) {
	defaultCurrency := "some currency"
	const defaultToUSDRate = 2
	const usd = "$"
	var usdCurrency *ofxgo.Currency
	if currSym, err := ofxgo.NewCurrSymbol("USD"); err != nil {
		panic(err)
	} else {
		usdCurrency = &ofxgo.Currency{
			CurSym:  *currSym,
			CurRate: makeOFXAmount(defaultToUSDRate),
		}
		_, err := usdCurrency.Valid()
		require.NoError(t, err)
	}

	for _, tc := range []struct {
		description string
		accountName string
		txn         ofxgo.Transaction
		expectedTxn ledger.Transaction
	}{
		{
			description: "happy path",
			accountName: "assets:Bank 1",
			txn: ofxgo.Transaction{
				Currency: usdCurrency,
				Name:     ofxgo.String(""),
				Payee:    &ofxgo.Payee{Name: "Some transaction"},
				TrnAmt:   makeOFXAmount(1.25),
			},
			expectedTxn: ledger.Transaction{
				Payee: "Some transaction",
				Postings: []ledger.Posting{
					{Account: "assets:Bank 1", Currency: usd, Amount: decimal.NewFromFloat(1.25)},
					{Account: model.Uncategorized, Currency: usd, Amount: decimal.NewFromFloat(-1.25)},
				},
			},
		},
		{
			description: "name instead of payee",
			accountName: "assets:Bank 1",
			txn: ofxgo.Transaction{
				Currency: usdCurrency,
				Name:     ofxgo.String("Hey there"),
				Payee:    &ofxgo.Payee{Name: "Some transaction"},
				TrnAmt:   makeOFXAmount(1.25),
			},
			expectedTxn: ledger.Transaction{
				Payee: "Hey there",
				Postings: []ledger.Posting{
					{Account: "assets:Bank 1", Currency: usd, Amount: decimal.NewFromFloat(1.25)},
					{Account: model.Uncategorized, Currency: usd, Amount: decimal.NewFromFloat(-1.25)},
				},
			},
		},
	} {
		someFID := "some FID"
		makeTxnID := func(id string) string {
			assert.Equal(t, string(tc.txn.FiTID), id)
			return someFID
		}
		txn := parseTransaction(tc.txn, defaultCurrency, tc.accountName, makeTxnID)
		require.Len(t, txn.Postings, len(tc.expectedTxn.Postings))
		if len(tc.expectedTxn.Postings) > 0 && tc.expectedTxn.Postings[0].Tags == nil {
			// add default ID tag
			tc.expectedTxn.Postings[0].Tags = map[string]string{"id": someFID}
		}
		testhelpers.AssertEqualTransactions(t, tc.expectedTxn, txn)
	}
}
func TestMakeUniqueTxnID(t *testing.T) {
	for _, tc := range []struct {
		fid, accountID, txnID string
		expectedID            string
	}{
		{"some FID", "some account", "some txn", "some FID-some account-some txn"},
		{"some, punctuation", "some account", "txn", "some punctuation-some account-txn"},
		{"some punctuation", "some: account", "txn", "some punctuation-some account-txn"},
	} {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tc.expectedID, MakeUniqueTxnID(tc.fid, tc.accountID)(tc.txnID))
		})
	}
}

func TestBalanceTransactions(t *testing.T) {
	for _, tc := range []struct {
		description  string
		txns         []ledger.Transaction
		balance      float64
		balanceDate  string
		endDate      string
		expectedTxns []ledger.Transaction
	}{
		{
			description: "no transactions",
			balance:     0,
			balanceDate: "2019/01/01",
			endDate:     "2019/01/01",
		},
		{
			description: "one transaction",
			balance:     5.00,
			balanceDate: "2019/01/01",
			endDate:     "2019/01/02",
			txns: []ledger.Transaction{
				makeTxn("2019/01/02", -2.00),
			},
			expectedTxns: []ledger.Transaction{
				makeTxnWithBalance("2019/01/02", -2.00, 3.00),
			},
		},
		{
			description: "sorts transactions",
			balance:     5.00,
			balanceDate: "2019/01/01",
			endDate:     "2019/01/03",
			txns: []ledger.Transaction{
				makeTxn("2019/01/03", -3.00),
				makeTxn("2019/01/02", -1.00),
			},
			expectedTxns: []ledger.Transaction{
				makeTxnWithBalance("2019/01/02", -1.00, 4.00),
				makeTxnWithBalance("2019/01/03", -3.00, 1.00),
			},
		},
		{
			description: "populates prior to balance date",
			balance:     5.00,
			balanceDate: "2019/01/05",
			endDate:     "2019/01/05",
			txns: []ledger.Transaction{
				makeTxn("2019/01/02", -2.00),
			},
			expectedTxns: []ledger.Transaction{
				makeTxnWithBalance("2019/01/02", -2.00, 5.00),
			},
		},
		{
			description: "balance before 3 txns",
			balance:     6.00,
			balanceDate: "2019/01/01",
			endDate:     "2019/01/04",
			txns: []ledger.Transaction{
				makeTxn("2019/01/02", -1.00),
				makeTxn("2019/01/03", -2.00),
				makeTxn("2019/01/04", -3.00),
			},
			expectedTxns: []ledger.Transaction{
				makeTxnWithBalance("2019/01/02", -1.00, 5.00),
				makeTxnWithBalance("2019/01/03", -2.00, 3.00),
				makeTxnWithBalance("2019/01/04", -3.00, 0.00),
			},
		},
		{
			description: "balance after 3 txns",
			balance:     0.00,
			balanceDate: "2019/01/05",
			endDate:     "2019/01/05",
			txns: []ledger.Transaction{
				makeTxn("2019/01/02", -1.00),
				makeTxn("2019/01/03", -2.00),
				makeTxn("2019/01/04", -3.00),
			},
			expectedTxns: []ledger.Transaction{
				makeTxnWithBalance("2019/01/02", -1.00, 5.00),
				makeTxnWithBalance("2019/01/03", -2.00, 3.00),
				makeTxnWithBalance("2019/01/04", -3.00, 0.00),
			},
		},
		{
			description: "balance between 3 txns",
			balance:     5.00,
			balanceDate: "2019/01/03",
			endDate:     "2019/01/04",
			txns: []ledger.Transaction{
				makeTxn("2019/01/01", -1.00),
				makeTxn("2019/01/02", -2.00),
				makeTxn("2019/01/04", -3.00),
			},
			expectedTxns: []ledger.Transaction{
				makeTxnWithBalance("2019/01/01", -1.00, 7.00),
				makeTxnWithBalance("2019/01/02", -2.00, 5.00),
				makeTxnWithBalance("2019/01/04", -3.00, 2.00),
			},
		},
		{
			description: "balance on same day as txn",
			balance:     6.00,
			balanceDate: "2019/01/02",
			endDate:     "2019/01/03",
			txns: []ledger.Transaction{
				makeTxn("2019/01/01", -1.00),
				makeTxn("2019/01/02", -2.00),
				makeTxn("2019/01/02", -3.00),
				makeTxn("2019/01/03", -4.00),
			},
			expectedTxns: []ledger.Transaction{
				makeTxnWithBalance("2019/01/01", -1.00, 11.00),
				makeTxnWithBalance("2019/01/02", -2.00, 9.00),
				makeTxnWithBalance("2019/01/02", -3.00, 6.00),
				makeTxnWithBalance("2019/01/03", -4.00, 2.00),
			},
		},
		{
			description: "refuse to add balances if balance date is after end date",
			balance:     10000,
			balanceDate: "2020/01/01",
			endDate:     "2019/01/02",
			txns: []ledger.Transaction{
				makeTxn("2019/01/01", -1.00),
				makeTxn("2019/01/02", -2.00),
			},
			expectedTxns: []ledger.Transaction{
				makeTxn("2019/01/01", -1.00),
				makeTxn("2019/01/02", -2.00),
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			balance := decimal.NewFromFloat(tc.balance)
			balanceDate := parseDate(tc.balanceDate)
			statementEndDate := parseDate(tc.endDate)
			balanceTransactions(tc.txns, balance, balanceDate, statementEndDate)
			require.Len(t, tc.txns, len(tc.expectedTxns))
			for i := range tc.expectedTxns {
				testhelpers.AssertEqualTransactions(t, tc.expectedTxns[i], tc.txns[i])
			}
		})
	}
}
