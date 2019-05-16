package ledger

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

type Ledger struct {
	transactions []Transaction
	idSet        map[string]bool
}

func New(transactions []Transaction) (*Ledger, error) {
	idSet := make(map[string]bool, len(transactions)*2)
	for _, transaction := range transactions {
		if id := transaction.ID(); id != "" {
			if idSet[id] {
				return nil, duplicateTransactionError(id)
			}
			idSet[id] = true
		}
		for _, posting := range transaction.Postings {
			if id := posting.ID(); id != "" {
				if idSet[id] {
					return nil, duplicateTransactionError(id)
				}
				idSet[id] = true
			}
		}
	}
	return &Ledger{
		transactions: transactions,
		idSet:        idSet,
	}, nil
}

func NewFromReader(reader io.Reader) (*Ledger, error) {
	var transactions []Transaction
	scanner := bufio.NewScanner(reader)
	var txn Transaction
	readingPostings := false

	endTxn := func() error {
		if !readingPostings {
			return nil
		}
		if len(txn.Postings) < 2 {
			return fmt.Errorf("A transaction must have at least two postings: %s", txn.String())
		}
		readingPostings = false
		transactions = append(transactions, txn)
		txn = Transaction{}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if trimLine := strings.TrimSpace(line); trimLine == "" || trimLine[0] == ';' {
			// is blank line
			if err := endTxn(); err != nil {
				return nil, err
			}
		} else if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			if err := endTxn(); err != nil {
				return nil, err
			}
			// is txn payee line
			err := parsePayeeLine(&txn, line)
			if err != nil {
				return nil, err
			}
			readingPostings = true
		} else if readingPostings {
			// is posting line
			posting, err := NewPostingFromString(line)
			if err != nil {
				return nil, err
			}
			txn.Postings = append(txn.Postings, posting)
		} else {
			return nil, fmt.Errorf("Unknown line format detected: %s", line)
		}
	}
	if err := endTxn(); err != nil {
		return nil, err
	}
	return New(transactions)
}

func (l *Ledger) String() string {
	var buf bytes.Buffer
	for _, txn := range l.transactions {
		buf.WriteString(txn.String())
	}
	return buf.String()
}

func duplicateTransactionError(id string) error {
	return fmt.Errorf("Duplicate transaction ID found: '%s'", id)
}
