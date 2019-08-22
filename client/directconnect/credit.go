package directconnect

import (
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/client/model"
)

// CreditCard represents a credit card account
type CreditCard struct {
	directAccount
}

// NewCreditCard creates an account from credit card details
func NewCreditCard(id, description string, connector Connector) Account {
	return &CreditCard{
		directAccount: directAccount{
			AccountID:          id,
			AccountDescription: description,
			DirectConnect:      connector,
		},
	}
}

// Statement implements Requestor
func (cc *CreditCard) Statement(req *ofxgo.Request, start, end time.Time) error {
	return generateCCStatement(cc, req, start, end, ofxgo.RandomUID)
}

func generateCCStatement(
	cc *CreditCard,
	req *ofxgo.Request,
	start, end time.Time,
	getUID func() (*ofxgo.UID, error),
) error {
	uid, err := getUID()
	if err != nil {
		return err
	}

	req.CreditCard = append(req.CreditCard, &ofxgo.CCStatementRequest{
		TrnUID: *uid,
		CCAcctFrom: ofxgo.CCAcct{
			AcctID: ofxgo.String(cc.ID()),
		},
		DtStart: &ofxgo.Date{Time: start},
		DtEnd:   &ofxgo.Date{Time: end},
		Include: true, // Include transactions (instead of only balance information)
	})
	return nil
}

func (cc *CreditCard) Type() string {
	return model.LiabilityAccount
}
