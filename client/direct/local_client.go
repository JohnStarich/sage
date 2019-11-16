package direct

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/aclindsa/ofxgo"
	"github.com/pkg/errors"
)

const (
	localhostPrefix = "http://localhost"
)

var (
	errBadLocalhost = errors.New("Refusing to send OFX request to localhost. URL must start with '" + localhostPrefix + "' and not contain a password")
)

// localClient enables insecure requests on localhost, provided that no passwords are involved
type localClient struct {
	ofxgo.Client
}

// newLocalClient returns a new localClient for the given URL and basic client
func newLocalClient(url string, client *ofxgo.BasicClient) (ofxgo.Client, error) {
	if !IsLocalhostTestURL(url) {
		return nil, errBadLocalhost
	}
	return &localClient{
		Client: client,
	}, nil
}

// RawRequest runs a raw request for the given URL and reader against localhost. Errors if the host isn't for localhost OR a password field is included.
func (l *localClient) RawRequest(url string, r io.Reader) (*http.Response, error) {
	if !IsLocalhostTestURL(url) {
		return nil, errBadLocalhost
	}
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if strings.Contains(string(body), "<USERPASS>") {
		return nil, errBadLocalhost
	}
	r = bytes.NewBuffer(body)

	response, err := http.Post(url, "application/x-ofx", r) // nolint:gosec // URL variable required to fit OFX client interface
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		return nil, errors.New("OFXQuery request status: " + response.Status)
	}

	return response, nil
}

// IsLocalhostTestURL returns true if this is a valid URL and starts with http://localhost
func IsLocalhostTestURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	return err == nil && u.Scheme == "http" && u.Hostname() == "localhost"
}

// MarshalRequest implement the requestMarshaler interface to handle the special empty password case
func (l *localClient) MarshalRequest(r *ofxgo.Request) (io.Reader, error) {
	r.SetClientFields(l)

	const fakePassword = "something"
	r.Signon.UserPass = fakePassword // bypass validity checks for including a password
	b, err := r.Marshal()
	if err != nil {
		return nil, err
	}
	requestString := b.String()
	foundPass := false
	for _, passwordElem := range []string{"<USERPASS>" + fakePassword + "</USERPASS>", "<USERPASS>" + fakePassword} {
		if strings.Contains(requestString, passwordElem) {
			requestString = strings.Replace(requestString, passwordElem, "", 1)
			foundPass = true
			break
		}
	}
	if !foundPass {
		return nil, errors.New("Error formatting with an empty password")
	}
	return strings.NewReader(requestString), nil
}

// RequestNoParse runs a raw request by marshalling the given request, returns the raw response
func (l *localClient) RequestNoParse(r *ofxgo.Request) (*http.Response, error) {
	buf, err := l.MarshalRequest(r)
	if err != nil {
		return nil, err
	}
	return l.RawRequest(r.URL, buf)
}

// Request runs the given request and parses the result
func (l *localClient) Request(r *ofxgo.Request) (*ofxgo.Response, error) {
	response, err := l.RequestNoParse(r)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	ofxresp, err := ofxgo.ParseResponse(response.Body)
	if err != nil {
		return nil, err
	}
	return ofxresp, nil
}
