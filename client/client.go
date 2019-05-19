package client

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type sageClient struct {
	ofxgo.Client
	*zap.Logger
}

func New(url string, config Config) (ofxgo.Client, error) {
	s := &sageClient{}

	basicClient := &ofxgo.BasicClient{
		AppID:       "OFXGO",
		AppVer:      "0001",
		SpecVersion: ofxgo.OfxVersion200,
		NoIndent:    config.NoIndent,
	}
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
	s.Client = ofxgo.GetClient(url, basicClient)
	var err error
	if os.Getenv("DEVELOPMENT") == "true" {
		s.Logger, err = zap.NewDevelopment()
	} else {
		s.Logger, err = zap.NewProduction()
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *sageClient) Request(req *ofxgo.Request) (*ofxgo.Response, error) {
	if ce := s.Logger.Check(zap.DebugLevel, "Sending request"); ce != nil {
		reqCopyStruct := *req
		reqCopy := &reqCopyStruct
		reqCopy.SetClientFields(s.Client)
		b, err := reqCopy.Marshal()
		if err == nil {
			ce.Write()
			s.Logger.Debug(b.String())
		} else {
			ce.Write(zap.Error(err))
		}
	}

	response, reqErr := s.RequestNoParse(req)
	if ce := s.Logger.Check(zap.DebugLevel, "Received response"); response != nil && ce != nil {
		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read response body")
		}
		response.Body.Close()
		s.Logger.Debug(string(b))
		response.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	}
	if reqErr != nil {
		return nil, errors.Wrap(reqErr, "Error sending request")
	}
	defer response.Body.Close()

	ofxresp, parseErr := ofxgo.ParseResponse(response.Body)
	if parseErr != nil {
		return nil, errors.Wrap(parseErr, "Error parsing response body")
	}
	return ofxresp, nil
}

var (
	requestReplacer = strings.NewReplacer(
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
)

// RequestNoParse is mostly lifted from basic client's implementation
func (s *sageClient) RequestNoParse(r *ofxgo.Request) (*http.Response, error) {
	r.SetClientFields(s)

	b, err := r.Marshal()
	if err != nil {
		return nil, err
	}

	data := b.String()
	// fix for institutions that require Windows-like line endings
	data = strings.Replace(data, "\n", "\r\n", -1)
	if s.Client.OfxVersion().String()[0] == '1' {
		// fix closing tag issue for OFX 102 and USAA
		data = requestReplacer.Replace(data)
	}
	b = bytes.NewBufferString(data)

	return s.RawRequest(r.URL, b)
}

// RawRequest is mostly lifted from basic client's implementation
func (s *sageClient) RawRequest(url string, r io.Reader) (*http.Response, error) {
	if !strings.HasPrefix(url, "https://") {
		return nil, errors.New("Refusing to send OFX request with possible plain-text password over non-https protocol")
	}

	req, err := http.NewRequest("POST", url, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "httpclient")
	req.Header.Set("Content-Type", "application/x-ofx")
	req.Header.Set("Accept-Type", "application/x-ofx")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	req = req.WithContext(ctx)

	s.Logger.Debug("Sending request", zap.Reflect("headers", req.Header))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		if ce := s.Logger.Check(zap.DebugLevel, "Bad status code in response"); ce != nil {
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to read response body")
			}
			response.Body.Close()
			response.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			ce.Write(zap.Reflect("headers", response.Header))
			s.Logger.Debug(string(body))
		}
		return nil, errors.New("OFXQuery response status: " + response.Status)
	}

	return response, nil
}
