package client

import (
	"errors"
	"testing"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCCStatement(t *testing.T) {
	someID := "some ID"
	someInstitution := institution{description: "some institution"}
	creditCard := NewCreditCard(someID, someInstitution).(CreditCard)

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
			timestamp := time.Now()

			getTime := func() time.Time {
				return timestamp
			}

			dur := 42 * time.Second
			req, err := generateCCStatement(creditCard, dur, getUID, getTime)
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
						DtStart: &ofxgo.Date{Time: timestamp.Add(-dur)},
						DtEnd:   &ofxgo.Date{Time: timestamp},
						Include: true, // Include transactions (instead of only balance information)
					},
				},
			}, req)
		})
	}
}

func TestCreditCardStatement(t *testing.T) {
	req, err := CreditCard{}.Statement(1 * time.Minute)
	require.NoError(t, err)
	require.Len(t, req.CreditCard, 1)
	assert.IsType(t, &ofxgo.CCStatementRequest{}, req.CreditCard[0])
}
