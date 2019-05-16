package ledger

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	idTag = "id"
)

type Transaction struct {
	Comment  string
	Date     time.Time
	Payee    string
	Postings []Posting
	Tags     map[string]string
}

func parsePayeeLine(txn *Transaction, line string) error {
	tokens := strings.SplitN(line, ";", 2)
	if len(tokens) == 0 {
		return fmt.Errorf("Not enough tokens for payee line: %s", line)
	}
	line = strings.TrimSpace(tokens[0])
	if len(tokens) == 2 {
		txn.Comment, txn.Tags = parseTags(strings.TrimSpace(tokens[1]))
	}
	tokens = strings.SplitN(line, " ", 2)
	if len(tokens) != 2 {
		return fmt.Errorf("Not enough tokens for payee line: %s", line)
	}
	date, payee := strings.TrimSpace(tokens[0]), strings.TrimSpace(tokens[1])
	txn.Payee = payee
	var err error
	txn.Date, err = time.Parse("2006/01/02", date)
	if err != nil {
		return err
	}
	return nil
}

func parseTags(comment string) (string, map[string]string) {
	tags := make(map[string]string)
	// TODO parse tags
	return comment, tags
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

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func (t Transaction) ID() string {
	return t.Tags[idTag]
}

func (t Transaction) String() string {
	postings := make([]string, 0, len(t.Postings))
	accountLen, amountLen := 0, 0
	for _, posting := range t.Postings {
		accountLen = max(accountLen, len(posting.Account))
		if posting.Amount != nil {
			amountLen = max(amountLen, len(posting.Amount.String()))
		}
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
		strings.Join(postings, "    "),
	)
}
