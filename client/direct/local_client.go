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
	return l.rawRequest(url, r, http.DefaultClient.Do)
}

func (l *localClient) rawRequest(url string, r io.Reader, doRequest func(*http.Request) (*http.Response, error)) (*http.Response, error) {
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

	req, err := http.NewRequest(http.MethodPost, url, r) // nolint:gosec // URL variable required to fit OFX client interface
	if err != nil {
		panic("Impossible request error (covered by isLocalhost check): " + err.Error())
	}
	req.Header.Set("Content-Type", "application/x-ofx")
	response, err := doRequest(req)
	if err != nil {
		return nil, err
	}

	if response.StatusCode/100 != 2 {
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
	return l.marshalRequest(r, r.SetClientFields, r.Marshal)
}

func (l *localClient) marshalRequest(r *ofxgo.Request, setFieldsFn func(ofxgo.Client), marshaler func() (*bytes.Buffer, error)) (io.Reader, error) {
	setFieldsFn(l)

	const fakePassword = "something"
	r.Signon.UserPass = fakePassword // bypass validity checks for including a password
	b, err := marshaler()
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
	return l.requestNoParse(r, l.MarshalRequest, l.RawRequest)
}

func (l *localClient) requestNoParse(
	r *ofxgo.Request,
	marshaler func(r *ofxgo.Request) (io.Reader, error),
	rawRequest func(url string, r io.Reader) (*http.Response, error),
) (*http.Response, error) {
	buf, err := marshaler(r)
	if err != nil {
		return nil, err
	}
	return rawRequest(r.URL, buf)
}

// Request runs the given request and parses the result
func (l *localClient) Request(r *ofxgo.Request) (*ofxgo.Response, error) {
	return l.request(r, l.RequestNoParse, ofxgo.ParseResponse)
}

func (l *localClient) request(
	r *ofxgo.Request,
	requestNoParse func(r *ofxgo.Request) (*http.Response, error),
	parse func(reader io.Reader) (*ofxgo.Response, error),
) (*ofxgo.Response, error) {
	response, err := requestNoParse(r)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return parse(response.Body)
}
