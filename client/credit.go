package client

import (
	"time"

	"github.com/aclindsa/ofxgo"
)

type CreditCard struct {
	baseAccount
}

func NewCreditCard(id string, institution Institution) Account {
	return CreditCard{
		baseAccount: baseAccount{
			id:          id,
			institution: institution,
		},
	}
}

func (cc CreditCard) Statement(start, end time.Time) (ofxgo.Request, error) {
	return generateCCStatement(cc, start, end, ofxgo.RandomUID)
}

func generateCCStatement(
	cc CreditCard,
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
