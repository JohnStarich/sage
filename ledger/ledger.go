package ledger

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
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
		buf.WriteRune('\n')
	}
	return buf.String()
}

func (l *Ledger) Validate() error {
	if len(l.transactions) == 0 {
		return nil
	}

	transactions := l.transactions

	balances := make(map[string]decimal.Decimal)
	foundOpeningBalance := false
	for _, p := range l.transactions[0].Postings {
		if strings.HasPrefix(p.Account, "equity:") {
			if p.ID() != "Opening-Balance" && !strings.HasPrefix(p.Account, "equity:Opening Balance") {
				// this appears to be a custom equity line, ignore it
				continue
			}
			foundOpeningBalance = true
			if err := l.transactions[0].Validate(); err != nil {
				return err
			}
			transactions = transactions[1:]
		} else {
			balances[p.Account] = p.Amount
		}
	}
	if !foundOpeningBalance {
		// auto-create balances from first appearances
		balances = make(map[string]decimal.Decimal)
		for _, txn := range l.transactions {
			for _, p := range txn.Postings {
				if _, exists := balances[p.Account]; !exists {
					var balance decimal.Decimal
					if p.Balance != nil {
						balance = *p.Balance
					}
					balances[p.Account] = balance.Sub(p.Amount)
				}
			}
		}
	}

	previousAccountTxn := make(map[string]Transaction, len(balances))
	for _, txn := range transactions {
		if err := txn.Validate(); err != nil {
			return err
		}
		for _, p := range txn.Postings {
			if p.Balance == nil {
				balances[p.Account] = balances[p.Account].Add(p.Amount)
				previousAccountTxn[p.Account] = txn
				continue
			}
			prevBalance, balExists := balances[p.Account]
			if !balExists {
				return errors.Errorf("Balance assertion found for account '%s', but no opening balance detected:\n%s", p.Account, txn)
			}
			beforeAssertion := p.Balance.Sub(p.Amount)
			if !prevBalance.Equal(beforeAssertion) {
				openingBalHelper := ""
				if !foundOpeningBalance {
					openingBalHelper = " (opening balances were auto-generated)"
				}
				return errors.Errorf(
					"Failed balance assertion for account '%s'%s: difference = %s\nTransaction 1:\n%s\nTransaction 2:\n%s",
					p.Account,
					openingBalHelper,
					prevBalance.Sub(beforeAssertion),
					previousAccountTxn[p.Account],
					txn,
				)
			}
			balances[p.Account] = *p.Balance
			previousAccountTxn[p.Account] = txn
		}
	}
	return nil
}

func duplicateTransactionError(id string) error {
	return fmt.Errorf("Duplicate transaction ID found: '%s'", id)
}
