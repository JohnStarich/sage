package client

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	loggerDevEnv = "DEVELOPMENT"
)

type sageClient struct {
	ofxgo.Client
	*zap.Logger
	*rate.Limiter
}

func New(url string, config Config) (ofxgo.Client, error) {
	return newClient(url, config, getLoggerFromEnv, ofxgo.GetClient)
}

func newClient(
	url string, config Config,
	getLogger func() (*zap.Logger, error),
	getClient func(string, *ofxgo.BasicClient) ofxgo.Client,
) (ofxgo.Client, error) {
	s := &sageClient{}

	basicClient := &ofxgo.BasicClient{NoIndent: config.NoIndent}
	if config.AppID != "" {
		basicClient.AppID = config.AppID
	}
	if config.AppVersion != "" {
		basicClient.AppVer = config.AppVersion
	}
	if config.OFXVersion != "" {
		ofxVersion, err := ofxgo.NewOfxVersion(config.OFXVersion)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse ofx version")
		}
		basicClient.SpecVersion = ofxVersion
	}
	s.Client = getClient(url, basicClient)
	s.Limiter = rate.NewLimiter(rate.Inf, 0)
	if _, ok := s.Client.(*ofxgo.DiscoverCardClient); ok {
		s.Limiter = rate.NewLimiter(rate.Every(5*time.Second), 1)
	}
	var err error
	s.Logger, err = getLogger()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func getLoggerFromEnv() (*zap.Logger, error) {
	if os.Getenv(loggerDevEnv) == "true" {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}

func (s *sageClient) Request(req *ofxgo.Request) (*ofxgo.Response, error) {
	return request(req, s.RequestNoParse, ofxgo.ParseResponse)
}

func request(
	req *ofxgo.Request,
	serveRequest func(*ofxgo.Request) (*http.Response, error),
	parseResponse func(io.Reader) (*ofxgo.Response, error),
) (*ofxgo.Response, error) {
	response, err := serveRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "Error sending request")
	}
	defer response.Body.Close()

	ofxresp, parseErr := parseResponse(response.Body)
	if parseErr != nil {
		return nil, errors.Wrap(parseErr, "Error parsing response body")
	}
	return ofxresp, nil
}

// RequestNoParse is mostly lifted from basic client's implementation
func (s *sageClient) RequestNoParse(req *ofxgo.Request) (*http.Response, error) {
	return doInstrumentedRequest(req, s.Logger, newRequestMarshaler(s), s.RawRequest)
}

func doInstrumentedRequest(
	req *ofxgo.Request, logger *zap.Logger, marshal requestMarshaler,
	doPostRequest func(string, io.Reader) (*http.Response, error),
) (*http.Response, error) {
	requestData, err := marshal(req)
	if err != nil {
		return nil, err
	}
	logger.Debug("Marshaled request:\n" + requestData.String())

	response, responseErr := doPostRequest(req.URL, requestData)
	if ce := logger.Check(zap.DebugLevel, "Received response"); responseErr == nil && ce != nil {
		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read response body")
		}
		response.Body.Close()
		logger.Debug(string(b))
		response.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	}
	return response, responseErr
}

type requestMarshaler func(*ofxgo.Request) (*bytes.Buffer, error)

func newRequestMarshaler(c ofxgo.Client) requestMarshaler {
	requestReplacer := strings.NewReplacer(
		"</DTCLIENT>", "",
		"</DTSTART>", "",
		"</DTEND>", "",
		"</INCLUDE>", "",
		"</USERID>", "",
		"</USERPASS>", "",
		"</LANGUAGE>", "",
		"</ORG>", "",
		"</FID>", "",
		"</APPID>", "",
		"</APPVER>", "",
		"</TRNUID>", "",
		"</BANKID>", "",
		"</ACCTID>", "",
		"</ACCTTYPE>", "",
	)
	return func(req *ofxgo.Request) (*bytes.Buffer, error) {
		req.SetClientFields(c)

		b, err := req.Marshal()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to marshal request")
		}

		data := b.String()
		// fix for institutions that require Windows-like line endings
		data = strings.Replace(data, "\n", "\r\n", -1)
		if c.OfxVersion().String()[0] == '1' {
			// fix closing tag issue for OFX 102 and USAA
			data = requestReplacer.Replace(data)
		}
		return bytes.NewBufferString(data), nil
	}
}

func (s *sageClient) RawRequest(url string, r io.Reader) (*http.Response, error) {
	reservation := s.Limiter.Reserve()
	if !reservation.OK() {
		return nil, errors.New("Cannot satisfy rate limiter burst condition")
	}
	delay := reservation.Delay()
	s.Logger.Debug("Rate limiting", zap.Duration("delay", delay))
	time.Sleep(delay)
	return s.Client.RawRequest(url, r)
}
