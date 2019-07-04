package client

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/ledger"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeCurrency(t *testing.T) {
	assert.Equal(t, "$", normalizeCurrency("USD"))
	assert.Equal(t, "something else", normalizeCurrency("something else"))
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
				assertEqualTransactions(t, tc.expectedTxns[i], tc.txns[i])
			}
		})
	}
}

// assertEqualTransactions carefully compares postings, with special handling for amounts and balances
func assertEqualTransactions(t *testing.T, expected, actual ledger.Transaction) bool {
	failed := false
	for i := range expected.Postings {
		if expected.Postings[i].Balance != actual.Postings[i].Balance {
			if expected.Postings[i].Balance == nil {
				failed = failed || !assert.Nil(t, actual.Postings[i].Balance)
			} else if actual.Postings[i].Balance == nil {
				failed = failed || !assert.NotNil(t, actual.Postings[i].Balance)
			} else {
				failed = failed || !assert.Equal(t,
					expected.Postings[i].Balance.String(),
					actual.Postings[i].Balance.String(),
					"Balances not equal for posting index #%d", i,
				)
			}
		}
		failed = failed || !assert.Equal(t,
			expected.Postings[i].Amount.String(),
			actual.Postings[i].Amount.String(),
			"Amounts not equal for posting index #%d", i,
		)
		expected.Postings[i].Balance = nil
		actual.Postings[i].Balance = nil
		expected.Postings[i].Amount = decimal.Zero
		actual.Postings[i].Amount = decimal.Zero
	}
	failed = failed || !assert.Equal(t, expected, actual)
	return !failed
}

type mockAccount struct {
	baseAccount
	bankID    string
	statement func(start, end time.Time) (ofxgo.Request, error)
}

var _ Bank = mockAccount{}

func (m mockAccount) Statement(start, end time.Time) (ofxgo.Request, error) {
	return m.statement(start, end)
}

func (m mockAccount) BankID() string {
	return m.bankID
}

func TestFetchTransactions(t *testing.T) {
	for _, tc := range []struct {
		description string
		startTime   time.Time
		endTime     time.Time
		queryErr    bool
		requestErr  bool
		responseErr bool
		expectErr   bool
	}{
		{
			description: "happy path",
			startTime:   someStartTime,
			endTime:     someEndTime,
		},
		{
			description: "query error",
			queryErr:    true,
			expectErr:   true,
		},
		{
			description: "request error",
			requestErr:  true,
			expectErr:   true,
		},
		{
			description: "response error",
			responseErr: true,
			expectErr:   true,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			ofxRequest := ofxgo.Request{
				Bank: []ofxgo.Message{ // non-zero statement requests
					&ofxgo.StatementRequest{},
				},
			}
			requestErr := errors.New("query error")
			queryErr := errors.New("query error")
			responseTxns := []ofxgo.Transaction{{TrnAmt: makeOFXAmount(0.4)}}
			responseBalance := makeOFXAmount(2.00)
			responseBalanceDate := parseDate("2019/01/02")
			statementResponse := ofxgo.Response{
				Bank: []ofxgo.Message{
					&ofxgo.StatementResponse{
						BalAmt: responseBalance,
						DtAsOf: ofxgo.Date{Time: responseBalanceDate},
						BankTranList: &ofxgo.TransactionList{
							Transactions: responseTxns,
						},
					},
				},
			}

			account := mockAccount{
				baseAccount: baseAccount{
					institution: institution{
						url:      "some URL",
						fid:      "some FID",
						org:      "some org",
						username: "some username",
						password: NewPassword("some password"),

						config: Config{
							ClientID: "some client ID",
						},
					},
				},
				statement: func(start, end time.Time) (ofxgo.Request, error) {
					assert.Equal(t, tc.startTime, start)
					assert.Equal(t, tc.endTime, end)
					if tc.queryErr {
						return ofxRequest, requestErr
					}
					return ofxRequest, nil
				},
			}
			doRequest := func(req *ofxgo.Request) (*ofxgo.Response, error) {
				requestWithSignon := ofxRequest
				requestWithSignon.URL = "some URL"
				requestWithSignon.Signon = ofxgo.SignonRequest{
					ClientUID: "some client ID",
					Fid:       "some FID",
					Org:       "some org",
					UserID:    "some username",
					UserPass:  "some password",
				}
				assert.Equal(t, &requestWithSignon, req)
				if tc.requestErr {
					return nil, requestErr
				}
				resp := statementResponse
				if tc.responseErr {
					resp.Signon.Status.Code = 1000
				}
				return &resp, nil
			}

			parsedTxns := make([]ledger.Transaction, 0)
			parseTxn := func(ofxgo.Transaction, string, string, func(string) string) ledger.Transaction {
				txn := ledger.Transaction{Comment: "some parsed txn"}
				parsedTxns = append(parsedTxns, txn)
				return txn
			}

			txns, err := fetchTransactions(
				account,
				tc.startTime,
				tc.endTime,
				doRequest,
				parseTxn,
			)
			if tc.expectErr {
				require.Error(t, err)
				if tc.queryErr {
					assert.Equal(t, queryErr, err)
				}
				if tc.requestErr {
					assert.Equal(t, requestErr, err)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, parsedTxns, txns, "returned txns must be equal to result of parse")
		})
	}
}

func makeOFXAmount(f float64) ofxgo.Amount {
	bigF := big.NewFloat(f)
	rat, _ := bigF.Rat(nil)
	return ofxgo.Amount{Rat: *rat}
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
					{Account: "uncategorized", Currency: usd, Amount: decimal.NewFromFloat(-1.25)},
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
					{Account: "uncategorized", Currency: usd, Amount: decimal.NewFromFloat(-1.25)},
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
		assertEqualTransactions(t, tc.expectedTxn, txn)
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
			account := mockAccount{
				baseAccount: baseAccount{
					id:          tc.accountID,
					institution: institution{fid: tc.fid},
				},
			}
			assert.Equal(t, tc.expectedID, makeUniqueTxnID(account)(tc.txnID))
		})
	}
}
