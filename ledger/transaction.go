package ledger

import (
	"bufio"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/johnstarich/sage/math"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	idTag      = "id"
	DateFormat = "2006/01/02"
)

var (
	missingAmountErr = fmt.Errorf("A transaction's postings may only have one missing amount, and it must be the last posting")
)

// Transaction is a strict(er) representation of a ledger transaction. The extra restrictions are used to verify correctness more easily.
type Transaction struct {
	Comment  string `json:",omitempty"`
	Date     time.Time
	Payee    string
	Postings []Posting
	Tags     map[string]string `json:",omitempty"`
}

type Transactions []*Transaction

func readAllTransactions(scanner *bufio.Scanner) ([]Transaction, error) {
	var transactions []Transaction
	type readerState struct {
		txn             Transaction
		readingPostings bool
		missingAmount   bool
		sum             decimal.Decimal
	}

	var state readerState

	endTxn := func() error {
		if !state.readingPostings {
			return nil
		}
		if len(state.txn.Postings) < 2 {
			return fmt.Errorf("A transaction must have at least two postings:\n%s", state.txn.String())
		}
		var total decimal.Decimal
		for _, p := range state.txn.Postings[:len(state.txn.Postings)-1] {
			total = total.Sub(p.Amount)
		}
		lastPosting := &state.txn.Postings[len(state.txn.Postings)-1]
		if !lastPosting.Amount.Equal(total) {
			return fmt.Errorf("Detected unbalanced transaction:\n%s", state.txn.String())
		}
		// valid txn
		transactions = append(transactions, state.txn)
		state = readerState{}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimLine := strings.TrimSpace(line)
		switch {
		case trimLine == "" || trimLine[0] == ';':
			// is blank line
			if err := endTxn(); err != nil {
				return nil, err
			}
		case !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t"):
			if err := endTxn(); err != nil {
				return nil, err
			}
			// is txn payee line
			err := parsePayeeLine(&state.txn, line)
			if err != nil {
				return nil, err
			}
			state.readingPostings = true
		case state.readingPostings:
			// is posting line
			if state.missingAmount {
				return nil, fmt.Errorf("Missing amount is only allowed on the last posting.")
			}
			posting, err := NewPostingFromString(line)
			switch {
			case err == missingAmountErr:
				state.missingAmount = true
				posting.Amount = state.sum
				posting.Currency = usd
			case err != nil:
				return nil, err
			default:
				state.sum = state.sum.Sub(posting.Amount)
			}
			state.txn.Postings = append(state.txn.Postings, posting)
		default:
			return nil, fmt.Errorf("Unknown line format detected: %s", line)
		}
	}
	err := endTxn()
	return transactions, err
}

func parsePayeeLine(txn *Transaction, line string) error {
	tokens := strings.SplitN(line, ";", 2)
	line = strings.TrimSpace(tokens[0])
	if len(tokens) == 2 {
		txn.Comment, txn.Tags = parseTags(strings.TrimSpace(tokens[1]))
	}
	tokens = strings.SplitN(line, " ", 2)
	date := strings.TrimSpace(tokens[0])
	if len(tokens) == 2 {
		txn.Payee = strings.TrimSpace(tokens[1])
	}
	var err error
	txn.Date, err = time.Parse(DateFormat, date)
	if err != nil {
		return err
	}
	return nil
}

func parseTags(comment string) (string, map[string]string) {
	if !strings.ContainsRune(comment, ':') {
		return comment, nil
	}

	tags := make(map[string]string)
	commentEnd := strings.LastIndexByte(comment[:strings.IndexRune(comment, ':')], ' ')
	var newComment string
	if commentEnd != -1 {
		newComment = strings.TrimSpace(comment[:commentEnd])
	}
	tagStrings := strings.Split(comment[commentEnd+1:], ",")
	for _, tagString := range tagStrings {
		keyValue := strings.SplitN(tagString, ":", 2)
		if len(keyValue) != 2 {
			return comment, nil
		}
		key, value := strings.TrimSpace(keyValue[0]), strings.TrimSpace(keyValue[1])
		tags[key] = value
	}
	return newComment, tags
}

func serializeComment(comment string, tags map[string]string) string {
	if len(tags) > 0 {
		tagStrings := make([]string, 0, len(tags))
		for k, v := range tags {
			tagStrings = append(tagStrings, fmt.Sprintf("%s: %s", k, v))
		}
		sort.Strings(tagStrings)
		if comment != "" {
			comment += " "
		}
		comment += strings.Join(tagStrings, ", ")
	}
	if comment != "" {
		comment = " ; " + comment
	}
	return comment
}

func (t Transaction) ID() string {
	return t.Tags[idTag]
}

func (t Transaction) Balanced() bool {
	var sum decimal.Decimal
	for _, p := range t.Postings {
		sum = sum.Add(p.Amount)
	}
	return sum.IsZero()
}

func (t Transaction) Validate() error {
	if len(t.Postings) < 2 {
		return errors.New("Transactions must have a minimum of 2 postings")
	}
	if !t.Balanced() {
		return errors.Errorf("Transaction is not balanced - postings do not sum to zero: %+v", t.Postings)
	}
	return nil
}

func (t Transaction) String() string {
	postings := make([]string, 0, len(t.Postings))
	accountLen, amountLen := 0, 0
	for _, posting := range t.Postings {
		accountLen = math.MaxInt(accountLen, len(posting.Account))
		amountLen = math.MaxInt(amountLen, len(posting.Amount.String()))
	}
	for _, posting := range t.Postings {
		postings = append(postings, posting.FormatTable(-accountLen, amountLen))
	}
	return fmt.Sprintf(
		"%4d/%02d/%02d %s%s\n    %s\n",
		t.Date.Year(),
		t.Date.Month(),
		t.Date.Day(),
		t.Payee,
		serializeComment(t.Comment, t.Tags),
		strings.Join(postings, "\n    "),
	)
}

func (txns Transactions) Sort() {
	sort.SliceStable(txns, func(a, b int) bool {
		return txns[a].Date.Before(txns[b].Date)
	})
}

func (t Transaction) matches(search string) int {
	payee := strings.ToLower(t.Payee)
	comment := strings.ToLower(t.Comment)
	date := strings.ToLower(t.Date.Format("Monday 2 January 2006"))
	postings := make([]string, 0, len(t.Postings))
	for _, p := range t.Postings {
		postings = append(postings, strings.ToLower(p.Account))
	}

	score := 0

	for _, token := range strings.Split(strings.ToLower(search), " ") {
		if strings.Contains(payee, token) {
			score++
		}
		if strings.Contains(comment, token) {
			score++
		}
		if strings.Contains(date, token) {
			score++
		}
		for _, p := range postings {
			if strings.Contains(p, token) {
				score++
			}
		}
	}
	return score
}
