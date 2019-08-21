package directconnect

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/aclindsa/ofxgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"golang.org/x/time/rate"
)

type recordCloser struct {
	io.Reader
	closed bool
}

func (r *recordCloser) Close() error {
	r.closed = true
	if readCloser, ok := r.Reader.(io.ReadCloser); ok {
		return readCloser.Close()
	}
	return nil
}

func (r *recordCloser) IsClosed() bool {
	return r.closed
}

func (r *recordCloser) Read(b []byte) (int, error) {
	if r.closed {
		return 0, errors.New("reader already closed")
	}
	return r.Reader.Read(b)
}

func TestNew(t *testing.T) {
	url := "URL"
	config := Config{AppID: "app ID"}
	client, err := newSimpleClient(url, config)
	require.NoError(t, err)
	assert.Equal(t, config.AppID, string(client.ID()))
}

func TestNewClient(t *testing.T) {
	expectedURL := "some URL"
	logger := zap.NewNop()
	loggerErr := errors.New("some logger error")
	client := &ofxgo.BasicClient{}
	limiter := rate.NewLimiter(rate.Inf, 0)
	ofxVersion := "bad OFX version"
	for _, tc := range []struct {
		description   string
		ofxVersionErr bool
		loggerErr     bool
	}{
		{description: "happy path"},
		{description: "bad OFX", ofxVersionErr: true},
		{description: "bad logger", loggerErr: true},
	} {
		t.Run(tc.description, func(t *testing.T) {
			config := Config{
				AppID:      "some app ID",
				AppVersion: "some app version",
				ClientID:   "some client ID",
				OFXVersion: ofxgo.OfxVersion200.String(),
				NoIndent:   true,
			}
			if tc.ofxVersionErr {
				config.OFXVersion = ofxVersion
			}
			getLogger := func() (*zap.Logger, error) {
				if tc.loggerErr {
					return nil, loggerErr
				}
				return logger, nil
			}
			getClient := func(url string, c *ofxgo.BasicClient) (ofxgo.Client, error) {
				assert.Equal(t, expectedURL, url)
				assert.Equal(t, config.AppID, c.AppID)
				assert.Equal(t, config.AppVersion, c.AppVer)
				assert.Equal(t, config.OFXVersion, c.SpecVersion.String())
				return client, nil
			}
			getLimiter := func(url string) *rate.Limiter {
				assert.Equal(t, expectedURL, url)
				return limiter
			}

			c, err := newClient(expectedURL, config, getLogger, getClient, getLimiter)
			if tc.loggerErr {
				assert.Equal(t, loggerErr, err)
				return
			} else if tc.ofxVersionErr {
				assert.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}
			require.IsType(t, &sageClient{}, c)
			sage := c.(*sageClient)
			assert.Equal(t, client, sage.Client)
			assert.Equal(t, logger, sage.Logger)
			assert.Equal(t, limiter, sage.Limiter)
		})
	}
}

func TestGetLoggerFromEnv(t *testing.T) {
	defer os.Setenv(loggerDevEnv, os.Getenv(loggerDevEnv)) // reset after test

	os.Setenv(loggerDevEnv, "true")
	logger, err := getLoggerFromEnv()
	assert.NotNil(t, logger)
	assert.NoError(t, err)

	os.Setenv(loggerDevEnv, "false")
	logger, err = getLoggerFromEnv()
	assert.NotNil(t, logger)
	assert.NoError(t, err)

	os.Unsetenv(loggerDevEnv)
	logger, err = getLoggerFromEnv()
	assert.NotNil(t, logger)
	assert.NoError(t, err)
}

func TestSageRequest(t *testing.T) {
	c, err := newSimpleClient("url", Config{})
	require.NoError(t, err)
	req := &ofxgo.Request{}
	_, err = c.Request(req)
	assert.Error(t, err)
}

func TestRequest(t *testing.T) {
	for _, tc := range []struct {
		description string
		requestErr  bool
		parseErr    bool
		expectErr   bool
	}{
		{description: "happy path"},
		{description: "request error", requestErr: true, expectErr: true},
		{description: "parse error", parseErr: true, expectErr: true},
	} {
		t.Run(tc.description, func(t *testing.T) {
			someBody := bytes.NewBufferString("some body")
			bodyCloser := &recordCloser{Reader: someBody}
			someResponse := &http.Response{
				Body: bodyCloser,
			}
			someRequestErr := errors.New("some request error")
			someParseErr := errors.New("some parse error")
			someParsedResponse := &ofxgo.Response{
				Signon: ofxgo.SignonResponse{
					Org: ofxgo.String("some org"),
				},
			}
			someRequest := &ofxgo.Request{URL: "some URL"}

			serveRequest := func(req *ofxgo.Request) (*http.Response, error) {
				assert.Equal(t, someRequest, req)
				if tc.requestErr {
					return nil, someRequestErr
				}
				return someResponse, nil
			}
			parseResponse := func(r io.Reader) (*ofxgo.Response, error) {
				assert.Equal(t, bodyCloser, r)
				if tc.parseErr {
					return nil, someParseErr
				}
				return someParsedResponse, nil
			}

			resp, err := request(someRequest, serveRequest, parseResponse)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, someParsedResponse, resp)
			assert.True(t, bodyCloser.closed)
		})
	}
}

func TestRequestNoParse(t *testing.T) {
	c, err := newSimpleClient("url", Config{})
	require.NoError(t, err)
	req := &ofxgo.Request{}
	_, err = c.RequestNoParse(req)
	assert.Error(t, err)
}

type FakeMarshaler struct {
	marshalRequest func(*ofxgo.Request) (io.Reader, error)
}

func (f FakeMarshaler) MarshalRequest(req *ofxgo.Request) (io.Reader, error) {
	return f.marshalRequest(req)
}

func TestDoInstrumentedRequest(t *testing.T) {
	for _, tc := range []struct {
		description string
		logLevel    zapcore.Level
		marshalErr  bool
		postErr     bool
		readErr     bool
		expectErr   bool
	}{
		{
			description: "happy path",
			logLevel:    zap.InfoLevel,
		},
		{
			description: "debug replaces body",
			logLevel:    zap.DebugLevel,
		},
		{
			description: "marshal error",
			marshalErr:  true,
			expectErr:   true,
		},
		{
			description: "post error",
			postErr:     true,
			expectErr:   true,
		},
		{
			description: "read error",
			readErr:     true,
			expectErr:   true,
			logLevel:    zap.DebugLevel,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			someRequest := &ofxgo.Request{URL: "some request"}
			logger := zaptest.NewLogger(t, zaptest.Level(tc.logLevel))

			someMarshalErr := errors.New("some marshal error")
			bufContent := "some buffer"
			someMarshalBuf := bytes.NewBufferString(bufContent)
			someMarshaller := FakeMarshaler{func(req *ofxgo.Request) (io.Reader, error) {
				assert.Equal(t, someRequest, req)
				if tc.marshalErr {
					return nil, someMarshalErr
				}
				return someMarshalBuf, nil
			}}

			someBody := bytes.NewBufferString("some body")
			bodyCloser := &recordCloser{Reader: someBody}
			someResponse := &http.Response{Body: bodyCloser}
			somePostErr := errors.New("some post error")
			doPostRequest := func(url string, r io.Reader) (*http.Response, error) {
				assert.Equal(t, someRequest.URL, url)
				data, err := ioutil.ReadAll(r)
				assert.NoError(t, err)
				assert.Equal(t, bufContent, string(data))
				if tc.postErr {
					return nil, somePostErr
				}
				if tc.readErr {
					bodyCloser.Close()
				}
				return someResponse, nil
			}

			resp, err := doInstrumentedRequest(someRequest, logger, someMarshaller, doPostRequest)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.logLevel == zap.DebugLevel {
				assert.True(t, bodyCloser != resp.Body, "Response body should be replaced with a different, identical body")
				assert.True(t, bodyCloser.closed, "Body should be closed")
			} else {
				assert.True(t, bodyCloser == resp.Body, "Response body should NOT be replaced")
				assert.False(t, bodyCloser.closed, "Body should NOT be closed")
			}
			assert.Equal(t, someResponse, resp, "Response should appear identical")
		})
	}
}

func TestNewRequestMarshaler(t *testing.T) {
	c := &ofxgo.BasicClient{
		AppID:          "myofx",
		AppVer:         "1000",
		NoIndent:       true,
		CarriageReturn: true,
	}
	marshaler := &sageClient{
		Client: c,
	}

	req := &ofxgo.Request{
		Signon: ofxgo.SignonRequest{
			UserID:   "some user",
			UserPass: "some pass",
		},
	}

	t.Run("OFX 1XX", func(t *testing.T) {
		c.SpecVersion = ofxgo.OfxVersion102
		buf, err := marshaler.MarshalRequest(req)
		require.NoError(t, err)
		dataBytes, err := ioutil.ReadAll(buf)
		require.NoError(t, err)
		data := string(dataBytes)
		assert.NotEmpty(t, data)
		assert.NotContains(t, strings.Replace(data, "\r\n", "", -1), "\n", `All line endings should be \r\n`)
		assert.Contains(t, data, "<DTCLIENT>")
		assert.NotContains(t, data, "</DTCLIENT>", "OFX 1XX does not have element end tags")
	})

	t.Run("OFX 2XX", func(t *testing.T) {
		c.SpecVersion = ofxgo.OfxVersion200
		buf, err := marshaler.MarshalRequest(req)
		require.NoError(t, err)
		dataBytes, err := ioutil.ReadAll(buf)
		require.NoError(t, err)
		data := string(dataBytes)
		assert.NotEmpty(t, data)
		assert.NotContains(t, strings.Replace(data, "\r\n", "", -1), "\n", `All line endings should be \r\n`)
		assert.Contains(t, data, "<DTCLIENT>")
		assert.Contains(t, data, "</DTCLIENT>")
	})
}
