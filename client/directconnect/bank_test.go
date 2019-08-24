package directconnect

import (
	"errors"
	"testing"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/client/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Bank = &bankAccount{}
var _ Requestor = &bankAccount{}

var (
	someEndTime   = time.Now()
	someStartTime = someEndTime.Add(-5 * time.Minute)
)

func TestBankStatementFromAccountType(t *testing.T) {
	b := bankAccount{
		AccountType:   CheckingType.String(),
		directAccount: directAccount{},
	}
	var req ofxgo.Request
	err := b.Statement(&req, someStartTime, someEndTime)
	require.NoError(t, err)
}

func TestBankGenerateStatement(t *testing.T) {
	someID := "some ID"
	someRoutingNumber := "some routing number"
	someDescription := "some description"
	someInstitution := &directConnect{
		BasicInstitution: model.BasicInstitution{InstDescription: "some institution"},
	}
	savings := NewSavingsAccount(someID, someRoutingNumber, someDescription, someInstitution).(*bankAccount)
	checking := NewCheckingAccount(someID, someRoutingNumber, someDescription, someInstitution).(*bankAccount)

	for _, tc := range []struct {
		description         string
		account             *bankAccount
		inputAccountType    string
		uidErr              bool
		expectErr           bool
		expectedAccountType string
	}{
		{
			description:         "happy path savings",
			account:             savings,
			inputAccountType:    SavingsType.String(),
			expectedAccountType: SavingsType.String(),
		},
		{
			description:         "happy path checking",
			account:             checking,
			inputAccountType:    CheckingType.String(),
			expectedAccountType: CheckingType.String(),
		},
		{
			description:      "UID error",
			account:          checking,
			inputAccountType: CheckingType.String(),
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

			var req ofxgo.Request
			err := generateBankStatement(tc.account, &req, someStartTime, someEndTime, tc.inputAccountType, getUID)
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
							BankID:   ofxgo.String(tc.account.RoutingNumber),
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

func TestBankStatement(t *testing.T) {
	var req ofxgo.Request
	b := bankAccount{AccountType: CheckingType.String()}
	err := b.Statement(&req, someStartTime, someEndTime)
	require.NoError(t, err)
	require.Len(t, req.Bank, 1)
	require.IsType(t, &ofxgo.StatementRequest{}, req.Bank[0])
	acctType := req.Bank[0].(*ofxgo.StatementRequest).BankAcctFrom.AcctType.String()
	assert.Equal(t, CheckingType.String(), acctType)
}
