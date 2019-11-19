package direct

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/aclindsa/ofxgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocalClient(t *testing.T) {
	clientPtr := new(ofxgo.BasicClient)
	_, err := newLocalClient("some URL", clientPtr)
	assert.Equal(t, errBadLocalhost, err)

	client, err := newLocalClient("http://localhost", clientPtr)
	require.NoError(t, err)
	require.IsType(t, &localClient{}, client)
	lClient := client.(*localClient)
	assert.True(t, clientPtr == lClient.Client)
}

func TestLocalRawRequest(t *testing.T) {
	resp, err := (&localClient{}).RawRequest("garbage URL", nil)
	if !assert.Error(t, err) {
		assert.NoError(t, resp.Body.Close())
	}
}

type badReader struct{}

func (b *badReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("Bad read")
}

func TestLocalRawRequestImpl(t *testing.T) {
	client := &localClient{}
	someURL := "some random URL"
	someLocalURL := "http://localhost/"
	someResponse := &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewReader(nil))}
	someErr := errors.New("some error")
	someBody := strings.NewReader("some body")
	somePasswordBody := strings.NewReader("some <USERPASS>body")
	someBodyErr := &badReader{}

	t.Run("is NOT localhost", func(t *testing.T) {
		doRequest := func(req *http.Request) (*http.Response, error) {
			panic("should not reach here")
		}
		resp, err := client.rawRequest(someURL, someBody, doRequest)
		if !assert.Equal(t, errBadLocalhost, err) {
			assert.NoError(t, resp.Body.Close())
		}
	})

	t.Run("is localhost", func(t *testing.T) {
		doRequest := func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, someLocalURL, req.URL.String())
			assert.Equal(t, "application/x-ofx", req.Header.Get("Content-Type"))
			return someResponse, someErr
		}
		resp, err := client.rawRequest(someLocalURL, someBody, doRequest)
		if !assert.Equal(t, someErr, err) {
			assert.NoError(t, resp.Body.Close())
		}
	})

	t.Run("provided password", func(t *testing.T) {
		doRequest := func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, someLocalURL, req.URL.String())
			assert.Equal(t, "application/x-ofx", req.Header.Get("Content-Type"))
			return someResponse, someErr
		}
		resp, err := client.rawRequest(someLocalURL, somePasswordBody, doRequest)
		if !assert.Equal(t, errBadLocalhost, err) {
			assert.NoError(t, resp.Body.Close())
		}
	})

	t.Run("read error", func(t *testing.T) {
		doRequest := func(req *http.Request) (*http.Response, error) {
			panic("should not reach here")
		}
		resp, err := client.rawRequest(someLocalURL, someBodyErr, doRequest)
		if !assert.Error(t, err) {
			assert.NoError(t, resp.Body.Close())
		}
		assert.Equal(t, "Bad read", err.Error())
	})

	t.Run("non-200 response code", func(t *testing.T) {
		doRequest := func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "bummer",
				StatusCode: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewReader(nil)),
			}, nil
		}
		resp, err := client.rawRequest(someLocalURL, someBody, doRequest)
		if !assert.Error(t, err) {
			assert.NoError(t, resp.Body.Close())
		} else {
			assert.Equal(t, "OFXQuery request status: bummer", err.Error())
		}
	})

	t.Run("happy path", func(t *testing.T) {
		doRequest := func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "woot",
				StatusCode: 299,
				Body:       ioutil.NopCloser(bytes.NewReader(nil)),
			}, nil
		}
		resp, err := client.rawRequest(someLocalURL, someBody, doRequest)
		require.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
	})
}

func TestMarshalRequest(t *testing.T) {
	client := &localClient{Client: &ofxgo.BasicClient{}}
	_, err := client.MarshalRequest(&ofxgo.Request{})
	assert.Error(t, err)
}

func TestMarshalRequestImpl(t *testing.T) {
	t.Run("password included", func(t *testing.T) {
		lClient := &localClient{}
		setClientFields := func(client ofxgo.Client) {
			assert.Equal(t, lClient, client)
		}

		req := &ofxgo.Request{}
		marshaler := func() (*bytes.Buffer, error) {
			assert.Equal(t, "something", string(req.Signon.UserPass))
			return bytes.NewBuffer([]byte(`<USERPASS>something</USERPASS>`)), nil
		}
		buf, err := lClient.marshalRequest(req, setClientFields, marshaler)
		require.NoError(t, err)
		require.NotNil(t, buf)
		data, err := ioutil.ReadAll(buf)
		require.NoError(t, err)
		assert.Equal(t, "", string(data))
	})

	t.Run("empty password", func(t *testing.T) {
		lClient := &localClient{}
		setClientFields := func(client ofxgo.Client) {
			assert.Equal(t, lClient, client)
		}

		req := &ofxgo.Request{}
		marshaler := func() (*bytes.Buffer, error) {
			assert.Equal(t, "something", string(req.Signon.UserPass))
			return bytes.NewBuffer([]byte(`nothing recognizable`)), nil
		}
		_, err := lClient.marshalRequest(req, setClientFields, marshaler)
		require.Error(t, err)
		assert.Equal(t, "Error formatting with an empty password", err.Error())
	})
}

func TestLocalRequestNoParse(t *testing.T) {
	client := &localClient{Client: &ofxgo.BasicClient{}}
	resp, err := client.RequestNoParse(&ofxgo.Request{})
	if !assert.Error(t, err) {
		assert.NoError(t, resp.Body.Close())
	}
}

func TestRequestNoParseImpl(t *testing.T) {
	client := &localClient{}
	someReader := strings.NewReader("some reader")
	someResponse := &http.Response{Body: ioutil.NopCloser(bytes.NewReader(nil))}
	marshaler := func(r *ofxgo.Request) (io.Reader, error) {
		return someReader, nil
	}
	rawRequest := func(url string, r io.Reader) (*http.Response, error) {
		assert.Equal(t, someReader, r)
		return someResponse, nil
	}

	resp, err := client.requestNoParse(&ofxgo.Request{}, marshaler, rawRequest)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, someResponse, resp)
}

func TestLocalRequest(t *testing.T) {
	client := &localClient{Client: &ofxgo.BasicClient{}}
	_, err := client.Request(&ofxgo.Request{})
	assert.Error(t, err)
}

func TestLocalRequestImpl(t *testing.T) {
	client := &localClient{}
	someOFXRequest := &ofxgo.Request{}
	someHTTPResponse := &http.Response{Body: ioutil.NopCloser(bytes.NewReader(nil))}
	someOFXResponse := &ofxgo.Response{}

	requestNoParse := func(r *ofxgo.Request) (*http.Response, error) {
		assert.True(t, someOFXRequest == r)
		return someHTTPResponse, nil
	}
	parse := func(reader io.Reader) (*ofxgo.Response, error) {
		assert.True(t, someHTTPResponse.Body == reader)
		return someOFXResponse, nil
	}
	resp, err := client.request(someOFXRequest, requestNoParse, parse)
	require.NoError(t, err)
	assert.True(t, resp == someOFXResponse)
}
