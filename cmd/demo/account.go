package main

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/rand"
	"golang.org/x/text/currency"
)

// AccountGenerator deterministically generates transactions for an account
type AccountGenerator struct {
	Org           string
	FID           string
	AccountID     string
	RoutingNumber string
	seed          uint64
}

type randTransaction struct {
	ID       string
	Date     time.Time
	Payee    string
	Currency string
	Amount   decimal.Decimal
}

func (a *AccountGenerator) getSeed() uint64 {
	if a.seed == 0 {
		a.seed = seedStringToInt(a.FID + "-" + a.AccountID)
	}
	return a.seed
}

func milliseconds(millis uint64) time.Duration {
	return time.Duration(int(millis) * int(time.Millisecond))
}

func seedStringToInt(seed string) uint64 {
	buf := bytes.NewBufferString(seed)
	var reducedVal uint64
	for val, err := binary.ReadUvarint(buf); err == nil; val, err = binary.ReadUvarint(buf) {
		reducedVal = (reducedVal ^ val) * val
	}
	return reducedVal
}

func truncateToYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

func (a *AccountGenerator) seedFromYear(t time.Time) uint64 {
	return a.getSeed() * uint64(t.Year())
}

// Transactions ...
func (a *AccountGenerator) Transactions(ofxVersionStr string, txnUID ofxgo.UID, cookie string, start, end time.Time) (ofxgo.Response, error) {
	ofxVersion, err := ofxgo.NewOfxVersion(ofxVersionStr)
	if err != nil {
		return ofxgo.Response{}, err
	}
	successStatus := ofxgo.Status{
		Code:     0,
		Severity: ofxgo.String("INFO"),
		Message:  ofxgo.String("Success"),
	}
	now := time.Now()
	response := ofxgo.Response{
		Version: ofxVersion,
		Signon: ofxgo.SignonResponse{
			Status:   successStatus,
			DtServer: ofxgo.Date{Time: now},
			Language: ofxgo.String("ENG"),
			Org:      ofxgo.String(a.Org),
			Fid:      ofxgo.String(a.FID),
		},
	}

	if a.RoutingNumber != "" {
		response.Bank = append(response.Bank, &ofxgo.StatementResponse{
			TrnUID:    txnUID,
			Status:    successStatus,
			CltCookie: ofxgo.String(cookie),
			CurDef:    ofxgo.CurrSymbol{Unit: currency.USD},
			DtAsOf:    ofxgo.Date{Time: now},
			BankAcctFrom: ofxgo.BankAcct{
				BankID:   ofxgo.String("routing number"),
				AcctID:   ofxgo.String("account number"),
				AcctType: ofxgo.AcctTypeChecking,
			},
			BankTranList: &ofxgo.TransactionList{
				DtStart:      ofxgo.Date{Time: start},
				DtEnd:        ofxgo.Date{Time: end},
				Transactions: a.transactions(start, end),
			},
		})
	} else {
		response.CreditCard = append(response.CreditCard, &ofxgo.CCStatementResponse{
			TrnUID:    txnUID,
			Status:    successStatus,
			CltCookie: ofxgo.String(cookie),
			CurDef:    ofxgo.CurrSymbol{Unit: currency.USD},
			DtAsOf:    ofxgo.Date{Time: now},
			CCAcctFrom: ofxgo.CCAcct{
				AcctID: ofxgo.String("account number"),
			},
			BankTranList: &ofxgo.TransactionList{
				DtStart:      ofxgo.Date{Time: start},
				DtEnd:        ofxgo.Date{Time: end},
				Transactions: a.transactions(start, end),
			},
		})
	}

	return response, nil
}

func (a *AccountGenerator) transactions(start, end time.Time) []ofxgo.Transaction {
	start = start.UTC()
	end = end.UTC()
	var rng rand.PCGSource
	date := truncateToYear(start)
	year := date.Year()
	seed := a.seedFromYear(date)

	var txns []ofxgo.Transaction
	for date.Before(end) {
		date = date.Add(milliseconds(rng.Uint64() >> 32)) // 32 bits for milliseconds caps out at about 50 days
		if date.Year() != year {
			// re-seed for each year. helps jumps between years.
			year = date.Year()
			seed = a.seedFromYear(date)
			date = truncateToYear(date)
			rng.Seed(seed)
			continue
		}

		if !date.Before(start) {
			txnSeed := seed * uint64(date.Month()) * uint64(date.Day())
			randTxn := newRandTransaction(date, txnSeed)
			txn := ofxgo.Transaction{
				Name:     ofxgo.String(randTxn.Payee),
				DtPosted: ofxgo.Date{Time: randTxn.Date},
				TrnAmt:   ofxgo.Amount{Rat: *randTxn.Amount.Rat()},
				FiTID:    ofxgo.String(randTxn.ID),
				Currency: &ofxgo.Currency{CurSym: ofxgo.CurrSymbol{Unit: currency.USD}},
			}
			if randTxn.Amount.IsNegative() {
				txn.TrnType = ofxgo.TrnTypeDebit
			} else {
				txn.TrnType = ofxgo.TrnTypeCredit
			}
			txns = append(txns, txn)
		}
	}
	return txns
}

func newRandTransaction(date time.Time, seed uint64) randTransaction {
	var rng rand.PCGSource
	rng.Seed(seed)
	random := rand.New(&rng)
	id := strconv.FormatUint(random.Uint64(), 10)
	amount := decimal.NewFromFloat(random.Float64() * float64(random.Intn(100))).Round(2)
	return randTransaction{
		ID:       id,
		Date:     date,
		Payee:    payeeChoices[random.Int()%len(payeeChoices)],
		Currency: "$",
		Amount:   amount,
	}
}

var (
	payeeChoices = []string{
		"Frond n Me",
		"Home Despot",
		"Burger Palace",
		"The Flying Yodel",
		"Screech Sound Systems",
		"Lightship Travel",
		"Half Life Energy",
		"Snowball Cleaners",
		"Flux Timepieces",
		"Dynaworks Fireworks",
		"Pipe Dreams Industries",
		"Primary Color Inc",
		"Yesterday's News",
		"Luna Tick's Bar and Grill",
		"Danger Zones",
		"Roaring Spoon",
		"Lightning Up Counseling",
		"Green Grape Grocer",
		"Hamstrung Deli",
	}
)
