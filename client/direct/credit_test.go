package direct

import (
	"errors"
	"testing"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/client/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCCStatement(t *testing.T) {
	someID := "some ID"
	someInstitution := &directConnect{
		BasicInstitution: model.BasicInstitution{InstDescription: "some institution"},
	}
	creditCard := NewCreditCard(someID, "some description", someInstitution).(*CreditCard)

	for _, tc := range []struct {
		description string
		uidErr      bool
		expectErr   bool
	}{
		{
			description: "happy path",
		},
		{
			description: "UID error",
			uidErr:      true,
			expectErr:   true,
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
			err := generateCCStatement(creditCard, &req, someStartTime, someEndTime, getUID)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, ofxgo.Request{
				CreditCard: []ofxgo.Message{
					&ofxgo.CCStatementRequest{
						TrnUID: uid,
						CCAcctFrom: ofxgo.CCAcct{
							AcctID: ofxgo.String(creditCard.ID()),
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

func TestCreditCardStatement(t *testing.T) {
	var req ofxgo.Request
	err := (&CreditCard{}).Statement(&req, someStartTime, someEndTime)
	require.NoError(t, err)
	require.Len(t, req.CreditCard, 1)
	assert.IsType(t, &ofxgo.CCStatementRequest{}, req.CreditCard[0])
}
