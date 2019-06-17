package client

import (
	"errors"
	"testing"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	someEndTime   = time.Now()
	someStartTime = someEndTime.Add(-5 * time.Minute)
)

func TestBankAccount(t *testing.T) {
	assert.Equal(t, "some ID", bankAccount{bankID: "some ID"}.BankID())
}

func TestNewCheckingAccount(t *testing.T) {
	someID := "some ID"
	someRoutingNumber := "some routing number"
	someDescription := "some description"
	someInstitution := institution{description: "some institution"}
	a := NewCheckingAccount(someID, someRoutingNumber, someDescription, someInstitution)
	assert.IsType(t, Checking{}, a)
	require.Implements(t, (*Bank)(nil), a)

	assert.Equal(t, someRoutingNumber, a.(Bank).BankID())
	assert.Equal(t, someInstitution, a.Institution())
	assert.Equal(t, someID, a.ID())
}

func TestNewSavingsAccount(t *testing.T) {
	someID := "some ID"
	someRoutingNumber := "some routing number"
	someDescription := "some description"
	someInstitution := institution{description: "some institution"}
	a := NewSavingsAccount(someID, someRoutingNumber, someDescription, someInstitution)
	assert.IsType(t, Savings{}, a)
	require.Implements(t, (*Bank)(nil), a)

	assert.Equal(t, someRoutingNumber, a.(Bank).BankID())
	assert.Equal(t, someInstitution, a.Institution())
	assert.Equal(t, someID, a.ID())
}

func TestBankStatementFromAccountType(t *testing.T) {
	b := bankAccount{}
	_, err := b.statementFromAccountType(someStartTime, someEndTime, checkingType)
	require.NoError(t, err)
}

func TestGenerateBankStatement(t *testing.T) {
	someID := "some ID"
	someRoutingNumber := "some routing number"
	someDescription := "some description"
	someInstitution := institution{description: "some institution"}
	savings := NewSavingsAccount(someID, someRoutingNumber, someDescription, someInstitution).(Savings).bankAccount
	checking := NewCheckingAccount(someID, someRoutingNumber, someDescription, someInstitution).(Checking).bankAccount

	for _, tc := range []struct {
		description         string
		account             bankAccount
		inputAccountType    string
		uidErr              bool
		expectErr           bool
		expectedAccountType string
	}{
		{
			description:         "happy path savings",
			account:             savings,
			inputAccountType:    savingsType,
			expectedAccountType: savingsType,
		},
		{
			description:         "happy path checking",
			account:             checking,
			inputAccountType:    checkingType,
			expectedAccountType: checkingType,
		},
		{
			description:      "UID error",
			account:          checking,
			inputAccountType: checkingType,
			uidErr:           true,
			expectErr:        true,
		},
		{
			description:      "account type error",
			account:          checking,
			inputAccountType: "nope",
			expectErr:        true,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			uid := ofxgo.UID("some UID")
			uidErr := errors.New("some UID error")
			getUID := func() (*ofxgo.UID, error) {
				if tc.uidErr {
					return nil, uidErr
				}
				return &uid, nil
			}

			req, err := generateBankStatement(tc.account, someStartTime, someEndTime, tc.inputAccountType, getUID)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			acctTypeEnum, err := ofxgo.NewAcctType(tc.expectedAccountType)
			require.NoError(t, err)

			assert.Equal(t, ofxgo.Request{
				Bank: []ofxgo.Message{
					&ofxgo.StatementRequest{
						TrnUID: uid,
						BankAcctFrom: ofxgo.BankAcct{
							BankID:   ofxgo.String(tc.account.BankID()),
							AcctID:   ofxgo.String(tc.account.ID()),
							AcctType: acctTypeEnum,
						},
						DtStart: &ofxgo.Date{Time: someStartTime},
						DtEnd:   &ofxgo.Date{Time: someEndTime},
						Include: true, // Include transactions (instead of only balance information)
					},
				},
			}, req)
		})
	}
}

func TestCheckingStatement(t *testing.T) {
	req, err := Checking{}.Statement(someStartTime, someEndTime)
	require.NoError(t, err)
	require.Len(t, req.Bank, 1)
	require.IsType(t, &ofxgo.StatementRequest{}, req.Bank[0])
	acctType := req.Bank[0].(*ofxgo.StatementRequest).BankAcctFrom.AcctType.String()
	assert.Equal(t, checkingType, acctType)
}

func TestSavingsStatement(t *testing.T) {
	req, err := Savings{}.Statement(someStartTime, someEndTime)
	require.NoError(t, err)
	require.Len(t, req.Bank, 1)
	require.IsType(t, &ofxgo.StatementRequest{}, req.Bank[0])
	acctType := req.Bank[0].(*ofxgo.StatementRequest).BankAcctFrom.AcctType.String()
	assert.Equal(t, savingsType, acctType)
}
