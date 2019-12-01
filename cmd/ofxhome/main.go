package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"strings"

	"github.com/johnstarich/go/regext"
	"github.com/johnstarich/sage/client/direct"
	"github.com/johnstarich/sage/client/direct/drivers"
)

const (
	ofxhomePrefix = "ofxhome:"
)

var (
	newLineLocations = regext.MustCompile(`
		(?:" [^"]* ")?   # don't capture new line locations inside quotes
		([ , \{ ])       # make new lines after commas and open braces
	`)
)

func main() {
	ofxhomePath := flag.String("ofxhome", "", "File path to an ofxhome.com XML dump")
	outputPath := flag.String("out", "", "File path to write the ofxhome Go file")
	flag.Parse()
	if *ofxhomePath == "" {
		fmt.Fprintln(os.Stderr, "Missing required flag: -path")
		flag.Usage()
		os.Exit(2)
	}

	if err := run(*ofxhomePath, *outputPath); err != nil {
		fmt.Fprintln(os.Stderr, "Error generating ofxhome Go file:", err.Error())
		os.Exit(1)
	}
}

func run(ofxhomePath, outputPath string) error {
	f, err := os.Open(ofxhomePath)
	if err != nil {
		return err
	}
	defer f.Close()
	ofxhomeGo, err := generateOFXHome(f)
	if err != nil {
		return err
	}
	var writer io.Writer = os.Stdout
	if outputPath != "" {
		writer, err = os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0750)
		if err != nil {
			return err
		}
	}
	_, err = io.Copy(writer, ofxhomeGo)
	return err
}

type xmlInstitution struct {
	XMLName xml.Name `xml:"institution"`
	ID      string   `xml:"id,attr"`
	Name    string   `xml:"name"`
	FID     string   `xml:"fid"`
	Org     string   `xml:"org"`
	URL     string   `xml:"url"`
	Profile struct {
		Bank       bool `xml:"bankmsgset,attr"`
		CreditCard bool `xml:"creditcardmsgset,attr"`
	} `xml:"profile"`
}

func decodeOFXHomeDump(r io.Reader) ([]xmlInstitution, error) {
	decoder := xml.NewDecoder(r)
	decoder.Strict = false
	var dump []xmlInstitution
	for {
		var inst xmlInstitution
		err := decoder.Decode(&inst)
		if err == io.EOF {
			return dump, nil
		}
		if err != nil {
			return nil, err
		}
		dump = append(dump, inst)
	}
}

func generateOFXHome(r io.Reader) (io.Reader, error) {
	dump, err := decodeOFXHomeDump(r)
	if err != nil {
		return nil, err
	}

	ofxDrivers := make([]direct.Driver, 0, len(dump))
	for _, inst := range dump {
		d := drivers.OFXHomeInstitution{
			InstID:          ofxhomePrefix + inst.ID,
			InstDescription: inst.Name,
			InstFID:         inst.FID,
			InstOrg:         inst.Org,
			InstURL:         inst.URL,
		}
		if inst.Profile.Bank {
			d.InstSupport = append(d.InstSupport, direct.MessageBank)
		}
		if inst.Profile.CreditCard {
			d.InstSupport = append(d.InstSupport, direct.MessageCreditCard)
		}
		if updatedDriver, shouldAdd := checkDriver(d); shouldAdd {
			ofxDrivers = append(ofxDrivers, updatedDriver)
		}
	}
	return formatOFXHomeGoFile(ofxDrivers)
}

func formatOFXHomeGoFile(d []direct.Driver) (io.Reader, error) {
	var s strings.Builder
	_, err := s.WriteString(`package drivers

import (
	"github.com/johnstarich/sage/client/direct"
)

func init() {
	direct.Register(ofxDrivers...)
}

var ofxDrivers =`)
	if err != nil {
		return nil, err
	}
	driverSlice := fmt.Sprintf("%#v\n", d)
	driverSlice = newLineLocations.ReplaceAllString(driverSlice, "$0\n")
	driverSlice = strings.Replace(driverSlice, "drivers.", "", -1)
	_, err = s.WriteString(driverSlice)
	if err != nil {
		return nil, err
	}
	driverSliceStr := s.String()
	result, err := format.Source([]byte(driverSliceStr))
	return bytes.NewReader(result), err
}

func checkDriver(d direct.Driver) (update direct.Driver, shouldAdd bool) {
	switch {
	case strings.HasPrefix(d.URL(), "https://ofx.discovercard.com"):
		// Discover OFX has been disabled
		return d, false
	default:
		return d, true
	}
}
