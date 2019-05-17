package ledger

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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
	transactions, err := readAllTransactions(scanner)
	if err != nil {
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
