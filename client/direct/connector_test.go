package direct

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/client/model"
	"github.com/johnstarich/sage/ledger"
	"github.com/johnstarich/sage/redactor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func parseDate(date string) time.Time {
	d, err := time.Parse("2006/01/02", date)
	if err != nil {
		panic(err)
	}
	return d
}

type mockAccount struct {
	*bankAccount
	statement func(req *ofxgo.Request, start, end time.Time) error
}

func (m mockAccount) Statement(req *ofxgo.Request, start, end time.Time) error {
	return m.statement(req, start, end)
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
				bankAccount: &bankAccount{
					directAccount: directAccount{
						DirectConnect: &directConnect{
							ConnectorPassword: "some password",
							ConnectorURL:      "some URL",
							ConnectorUsername: "some username",
							ConnectorConfig: Config{
								ClientID: "some client ID",
							},
							BasicInstitution: model.BasicInstitution{
								InstFID: "some FID",
								InstOrg: "some org",
							},
						},
					},
				},
				statement: func(req *ofxgo.Request, start, end time.Time) error {
					assert.Equal(t, tc.startTime, start)
					assert.Equal(t, tc.endTime, end)
					*req = ofxRequest
					if tc.queryErr {
						return requestErr
					}
					return nil
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

			someTransactions := []ledger.Transaction{
				{Comment: "some parsed txn"},
			}
			parser := func(resp *ofxgo.Response) ([]model.Account, []ledger.Transaction, error) {
				if tc.responseErr {
					return nil, nil, errors.New("some resp error")
				}
				return nil, someTransactions, nil
			}

			txns, err := fetchTransactions(
				account.DirectConnect,
				tc.startTime,
				tc.endTime,
				[]Requestor{account},
				doRequest,
				parser,
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
			assert.Equal(t, someTransactions, txns, "returned txns must be equal to result of parse")
		})
	}
}

func makeOFXAmount(f float64) ofxgo.Amount {
	bigF := big.NewFloat(f)
	rat, _ := bigF.Rat(nil)
	return ofxgo.Amount{Rat: *rat}
}

func TestInstitution(t *testing.T) {
	c := Config{AppID: "some app ID"}
	i := New(
		"Some important place",
		"1234",
		"some org",
		"some URL",
		"some user",
		"some password",
		c,
	)

	assert.Equal(t, "some URL", i.URL())
	assert.Equal(t, "some org", i.Org())
	assert.Equal(t, "1234", i.FID())
	assert.Equal(t, "some user", i.Username())
	assert.Equal(t, redactor.String("some password"), i.Password())
	assert.Equal(t, "Some important place", i.Description())
	assert.Equal(t, c, i.Config())
}

func TestLedgerAccountName(t *testing.T) {
	for _, tc := range []struct {
		description  string
		account      Account
		expectedName string
	}{
		{
			description: "credit cards are liability accounts",
			account: NewCreditCard(
				"super cash back",
				"some description",
				&directConnect{BasicInstitution: model.BasicInstitution{InstOrg: "Some Credit Card Co"}},
			),
			expectedName: "liabilities:Some Credit Card Co:****back",
		},
		{
			description: "banks are asset accounts",
			account: NewSavingsAccount(
				"blah account",
				"routing no",
				"blah account description",
				&directConnect{BasicInstitution: model.BasicInstitution{InstOrg: "The Boring Bank"}},
			),
			expectedName: "assets:The Boring Bank:****ount",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expectedName, model.LedgerAccountName(tc.account))
		})
	}
}

func TestValidateConnector(t *testing.T) {
	for _, tc := range []struct {
		name      string
		connector Connector
		errors    []string
		notErrors []string
	}{
		{
			name:   "nil connector",
			errors: []string{"Direct connect must not be empty"},
		},
		{
			name:      "non-local connector",
			connector: &directConnect{},
			errors: []string{
				"Institution name must not be empty",
				"Institution URL must not be empty",
				"Institution URL is required to use HTTPS",
				"Institution username must not be empty",
				"Institution password must not be empty",
				"Institution app ID must not be empty",
				"Institution app version must not be empty",
				"Institution OFX version must not be empty",
			},
		},
		{
			name: "local connector",
			connector: &directConnect{
				ConnectorURL: "http://localhost/",
			},
			notErrors: []string{
				"Institution password must not be empty",
			},
		},
		{
			name: "bad URL",
			connector: &directConnect{
				ConnectorURL: "://garbage",
			},
			errors: []string{
				"Institution URL is malformed",
			},
		},
		{
			name: "bad OFX version",
			connector: &directConnect{
				ConnectorConfig: Config{
					OFXVersion: "ABC",
				},
			},
			errors: []string{
				`Invalid OfxVersion: "ABC"`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateConnector(tc.connector)
			if len(tc.errors) == 0 && len(tc.notErrors) == 0 {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			for _, msg := range tc.errors {
				assert.Contains(t, err.Error(), msg)
			}
			for _, msg := range tc.notErrors {
				assert.NotContains(t, err.Error(), msg)
			}
		})
	}
}

func TestStatement(t *testing.T) {
	connector := &directConnect{}
	_, err := Statement(connector, time.Now(), time.Now(), nil, nil)
	assert.Error(t, err)
}

type mockRequestor struct {
	statementFn func(req *ofxgo.Request, start, end time.Time) error
}

func (m *mockRequestor) Statement(req *ofxgo.Request, start, end time.Time) error {
	return m.statementFn(req, start, end)
}

func TestVerify(t *testing.T) {
	connector := &directConnect{}
	someErr := errors.New("some error")
	requestor := &mockRequestor{statementFn: func(req *ofxgo.Request, start, end time.Time) error {
		return someErr
	}}
	err := Verify(connector, requestor, nil)
	assert.Equal(t, someErr, err)
}

func TestAccounts(t *testing.T) {
	connector := &directConnect{}
	_, err := Accounts(connector, zap.NewNop())
	assert.Error(t, err)
}

func TestAccountsImpl(t *testing.T) {
	connector := &directConnect{}
	logger := zap.NewNop()
	someResp := &ofxgo.Response{
		Signup: []ofxgo.Message{
			&ofxgo.AcctInfoResponse{
				AcctInfo: []ofxgo.AcctInfo{
					{
						CCAcctInfo: &ofxgo.CCAcctInfo{
							CCAcctFrom: ofxgo.CCAcct{
								AcctID: "1234",
							},
							SupTxDl: true,
						},
					},
				},
			},
		},
	}
	doRequest := func(req *ofxgo.Request) (*ofxgo.Response, error) {
		return someResp, nil
	}

	accounts, err := accounts(connector, logger, doRequest)
	require.NoError(t, err)
	assert.Equal(t, []model.Account{
		&CreditCard{
			directAccount: directAccount{
				AccountID:          "1234",
				AccountDescription: "1234",
				DirectConnect:      connector,
			},
		},
	}, accounts)
}

func TestParseAcctInfo(t *testing.T) {
	connector := &directConnect{}
	for _, tc := range []struct {
		description   string
		acctInfo      ofxgo.AcctInfo
		expectAccount model.Account
		expectErr     bool
	}{
		{
			description: "unknown account type",
			expectErr:   true,
		},
		{
			description: "checking account",
			acctInfo: ofxgo.AcctInfo{
				BankAcctInfo: &ofxgo.BankAcctInfo{
					BankAcctFrom: ofxgo.BankAcct{
						AcctID:   "some account ID",
						BankID:   "some bank ID",
						AcctType: ofxgo.AcctTypeChecking,
					},
					SupTxDl: true,
				},
			},
			expectAccount: &bankAccount{
				BankAccountType: CheckingType.String(),
				RoutingNumber:   "some bank ID",
				directAccount: directAccount{
					AccountID:          "some account ID",
					AccountDescription: "some account ID",
					DirectConnect:      connector,
				},
			},
		},
		{
			description: "savings account",
			acctInfo: ofxgo.AcctInfo{
				BankAcctInfo: &ofxgo.BankAcctInfo{
					BankAcctFrom: ofxgo.BankAcct{
						AcctID:   "some account ID",
						BankID:   "some bank ID",
						AcctType: ofxgo.AcctTypeSavings,
					},
					SupTxDl: true,
				},
			},
			expectAccount: &bankAccount{
				BankAccountType: SavingsType.String(),
				RoutingNumber:   "some bank ID",
				directAccount: directAccount{
					AccountID:          "some account ID",
					AccountDescription: "some account ID",
					DirectConnect:      connector,
				},
			},
		},
		{
			description: "unsupported bank account",
			acctInfo: ofxgo.AcctInfo{
				BankAcctInfo: &ofxgo.BankAcctInfo{
					BankAcctFrom: ofxgo.BankAcct{
						AcctID: "some account ID",
						BankID: "some bank ID",
					},
					SupTxDl: true,
				},
			},
			expectErr: true,
		},
		{
			description: "credit card account",
			acctInfo: ofxgo.AcctInfo{
				CCAcctInfo: &ofxgo.CCAcctInfo{
					CCAcctFrom: ofxgo.CCAcct{
						AcctID: "some account ID",
					},
					SupTxDl: true,
				},
			},
			expectAccount: &CreditCard{
				directAccount: directAccount{
					AccountID:          "some account ID",
					AccountDescription: "some account ID",
					DirectConnect:      connector,
				},
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			account, ok := parseAcctInfo(connector, tc.acctInfo, logger)
			if tc.expectErr {
				assert.False(t, ok, "Parse should fail")
				return
			}

			assert.True(t, ok, "Parse should succeed")
			assert.Equal(t, tc.expectAccount, account)
		})
	}
}
