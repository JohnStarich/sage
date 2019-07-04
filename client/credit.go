package client

import (
	"time"

	"github.com/aclindsa/ofxgo"
)

// CreditCard represents a credit card account
type CreditCard struct {
	baseAccount
}

// NewCreditCard creates an account from credit card details
func NewCreditCard(id, description string, institution Institution) Account {
	return &CreditCard{
		baseAccount: baseAccount{
			id:          id,
			description: description,
			institution: institution,
		},
	}
}

func (cc *CreditCard) Statement(start, end time.Time) (ofxgo.Request, error) {
	return generateCCStatement(cc, start, end, ofxgo.RandomUID)
}

func generateCCStatement(
	cc *CreditCard,
	start, end time.Time,
	getUID func() (*ofxgo.UID, error),
) (ofxgo.Request, error) {
	uid, err := getUID()
	if err != nil {
		return ofxgo.Request{}, err
	}

	return ofxgo.Request{
		CreditCard: []ofxgo.Message{
			&ofxgo.CCStatementRequest{
				TrnUID: *uid,
				CCAcctFrom: ofxgo.CCAcct{
					AcctID: ofxgo.String(cc.ID()),
				},
				DtStart: &ofxgo.Date{Time: start},
				DtEnd:   &ofxgo.Date{Time: end},
				Include: true, // Include transactions (instead of only balance information)
			},
		},
	}, nil
}
