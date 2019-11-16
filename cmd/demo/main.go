package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/aclindsa/xml"
)

// procedurally generate periodic transactions from account details
func main() {
	port := flag.Uint("port", 8080, "Server port to listen on")
	flag.Parse()

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	fmt.Printf("Starting server on %s...\n", addr)
	err := http.ListenAndServe(addr, http.HandlerFunc(handleOFXRequest))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func handleRequestError(resp http.ResponseWriter, err error) {
	resp.WriteHeader(http.StatusBadRequest)
	handleError(resp, err)
}

func handleServerError(resp http.ResponseWriter, err error) {
	fmt.Println(err.Error())
	fmt.Println(string(debug.Stack()))
	resp.WriteHeader(http.StatusInternalServerError)
	handleError(resp, err)
}

func handleError(resp http.ResponseWriter, err error) {
	fmt.Printf("Error: [%s] %s\n%s\n", resp.Header().Get("Status"), err.Error(), debug.Stack())
	_, _ = resp.Write([]byte(err.Error()))
}

var (
	// Poor man's SGML/XML parser for simplistic, error-prone request handling
	versionRe = regexp.MustCompile(`(?m)^(?:<\?OFX OFXHEADER="200" VERSION="|VERSION:)([0-9]{3})\b`)
	orgRe     = regexp.MustCompile(`<ORG>([^<\n]+)`)
	fidRe     = regexp.MustCompile(`<FID>([^<\n]+)`)
	acctRe    = regexp.MustCompile(`<ACCTID>([^<\n]+)`)
	routingRe = regexp.MustCompile(`<BANKID>([^<\n]+)`)
	txnUIDRe  = regexp.MustCompile(`<TRNUID>([^<\n]+)`)
	cookieRe  = regexp.MustCompile(`<CLTCOOKIE>([^<\n]+)`)
	startRe   = regexp.MustCompile(`<DTSTART>([^<\n]+)`)
	endRe     = regexp.MustCompile(`<DTEND>([^<\n]+)`)
)

func handleOFXRequest(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		resp.WriteHeader(http.StatusMethodNotAllowed)
		handleError(resp, errors.New("Method not allowed. Allowed methods: POST"))
		return
	}

	ofxRequestBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		handleError(resp, err)
		return
	}
	ofxRequest := strings.Replace(string(ofxRequestBytes), "\r\n", "\n", -1)

	var demoAccount AccountGenerator
	var version string
	{
		versionMatch := versionRe.FindStringSubmatch(ofxRequest)
		if versionMatch == nil {
			handleRequestError(resp, errors.New("Missing OFX version"))
			return
		}
		version = versionMatch[1]
	}
	{
		org := orgRe.FindStringSubmatch(ofxRequest)
		if org == nil {
			handleRequestError(resp, errors.New("Missing org name"))
			return
		}
		demoAccount.Org = org[1]
	}
	{
		fid := fidRe.FindStringSubmatch(ofxRequest)
		if fid == nil {
			handleRequestError(resp, errors.New("Missing FID"))
			return
		}
		demoAccount.FID = fid[1]
	}
	{
		acct := acctRe.FindStringSubmatch(ofxRequest)
		if acct == nil {
			handleRequestError(resp, errors.New("Missing account ID"))
			return
		}
		demoAccount.AccountID = acct[1]
	}
	{
		routing := routingRe.FindStringSubmatch(ofxRequest)
		if routing == nil && strings.Contains(ofxRequest, "<BANKACCTFROM>") {
			handleRequestError(resp, errors.New("Missing routing number (bank ID)"))
			return
		}
		if routing != nil {
			demoAccount.RoutingNumber = routing[1]
		}
	}
	var txnUID string
	{
		txnUIDMatch := txnUIDRe.FindStringSubmatch(ofxRequest)
		if txnUIDMatch == nil {
			handleRequestError(resp, errors.New("Missing statement transaction UID"))
			return
		}
		txnUID = txnUIDMatch[1]
	}
	var cookie string
	{
		cookieMatch := cookieRe.FindStringSubmatch(ofxRequest)
		if cookieMatch != nil {
			cookie = cookieMatch[1]
		}
	}
	var start, end time.Time
	{
		startMatch := startRe.FindStringSubmatch(ofxRequest)
		if startMatch == nil {
			handleRequestError(resp, errors.New("Missing statement start date"))
			return
		}
		startTime, err := parseDate(startMatch[1])
		if err != nil {
			handleRequestError(resp, err)
			return
		}
		start = startTime
	}
	{
		endMatch := endRe.FindStringSubmatch(ofxRequest)
		if endMatch == nil {
			handleRequestError(resp, errors.New("Missing statement end date"))
			return
		}
		endTime, err := parseDate(endMatch[1])
		if err != nil {
			handleRequestError(resp, err)
			return
		}
		end = endTime
	}

	txns, err := demoAccount.Transactions(version, ofxgo.UID(txnUID), cookie, start, end)
	if err != nil {
		handleServerError(resp, err)
		return
	}
	b, err := txns.Marshal()
	if err != nil {
		handleServerError(resp, err)
		return
	}
	_, err = b.WriteTo(resp)
	if err != nil {
		handleServerError(resp, err)
		return
	}
}

func parseDate(dateStr string) (time.Time, error) {
	var date ofxgo.Date
	d := xml.NewDecoder(strings.NewReader("<x>" + dateStr + "</x>"))
	if err := d.Decode(&date); err != nil {
		return date.Time, err
	}
	return date.Time, nil
}
