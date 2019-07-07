package client

import (
	"github.com/aclindsa/ofxgo"
	"github.com/pkg/errors"
)

const (
	ofxAuthFailed = 15500
)

var (
	// ErrAuthFailed is returned whenever a signon request fails with an authentication problem
	ErrAuthFailed = errors.New("Username or password is incorrect")
)

// Verify attempts to sign in with the given account. Returns any encountered errors
func Verify(account Account) error {
	client, err := clientForInstitution(account.Institution())
	if err != nil {
		return err
	}

	return verifyAccount(
		account,
		client.Request,
	)
}

func verifyAccount(
	account Account,
	doRequest func(*ofxgo.Request) (*ofxgo.Response, error),
) error {
	institution := account.Institution()
	config := institution.Config()
	query := ofxgo.Request{
		URL: institution.URL(),
		Signon: ofxgo.SignonRequest{
			ClientUID: ofxgo.UID(config.ClientID),
			Org:       ofxgo.String(institution.Org()),
			Fid:       ofxgo.String(institution.FID()),
			UserID:    ofxgo.String(institution.Username()),
			UserPass:  ofxgo.String(*institution.Password().password),
		},
	}

	response, err := doRequest(&query)
	if err != nil {
		return err
	}

	if response.Signon.Status.Code != 0 {
		if response.Signon.Status.Code == ofxAuthFailed {
			return ErrAuthFailed
		}
		meaning, err := response.Signon.Status.CodeMeaning()
		if err != nil {
			return errors.Wrap(err, "Failed to parse OFX response code")
		}
		return errors.Errorf("Nonzero signon status (%d: %s) with message: %s", response.Signon.Status.Code, meaning, response.Signon.Status.Message)
	}
	return nil
}
