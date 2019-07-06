package ledger

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

// Ledger tracks transactions from multiple institutions. Include error checking and validation for all ledger changes.
// Serializes into a "plain-text accounting" ledger file.
type Ledger struct {
	transactions Transactions
	idSet        map[string]bool
	mu           sync.RWMutex
}

// New creates a ledger with the given transactions. Must not contain any duplicate IDs
func New(transactions []Transaction) (*Ledger, error) {
	idSet, _, duplicates := makeIDSet(transactions)
	if len(duplicates) > 0 {
		return nil, duplicateTransactionError(strings.Join(duplicates, ", "))
	}
	return &Ledger{
		transactions: transactions,
		idSet:        idSet,
	}, nil
}

// NewFromReader creates a ledger from the given "plain-text accounting" ledger-encoded reader
func NewFromReader(reader io.Reader) (*Ledger, error) {
	var transactions []Transaction
	scanner := bufio.NewScanner(reader)
	transactions, err := readAllTransactions(scanner)
	if err != nil {
		return nil, err
	}
	return New(transactions)
}

func makeIDSet(transactions []Transaction) (idSet map[string]bool, uniqueTxns []Transaction, duplicates []string) {
	idSet = make(map[string]bool, len(transactions)*2)
	for _, transaction := range transactions {
		txnIsDupe := false
		if id := transaction.ID(); id != "" {
			if idSet[id] {
				txnIsDupe = true
				duplicates = append(duplicates, id)
			}
			idSet[id] = true
		}
		for _, posting := range transaction.Postings {
			if id := posting.ID(); id != "" {
				if idSet[id] {
					txnIsDupe = true
					duplicates = append(duplicates, id)
				}
				idSet[id] = true
			}
		}
		if !txnIsDupe {
			uniqueTxns = append(uniqueTxns, transaction)
		}
	}
	return
}

func (l *Ledger) String() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var buf bytes.Buffer
	for _, txn := range l.transactions {
		buf.WriteString(txn.String())
		buf.WriteRune('\n')
	}
	return buf.String()
}

// Validate returns a descriptive error should anything be wrong with the current ledger's transactions
func (l *Ledger) Validate() error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.transactions) == 0 {
		return nil
	}

	transactions := l.transactions

	balances := make(map[string]decimal.Decimal)
	foundOpeningBalance := false
	for _, p := range l.transactions[0].Postings {
		if strings.HasPrefix(p.Account, "equity:") {
			if !p.isOpeningBalance() {
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
	for ix, txn := range transactions {
		if err := txn.Validate(); err != nil {
			return NewValidateError(ix, err)
		}
		for _, p := range txn.Postings {
			if p.Balance == nil {
				balances[p.Account] = balances[p.Account].Add(p.Amount)
				previousAccountTxn[p.Account] = txn
				continue
			}
			prevBalance, balExists := balances[p.Account]
			if !balExists {
				return NewValidateError(ix, errors.Errorf("Balance assertion found for account '%s', but no opening balance detected:\n%s", p.Account, txn))
			}
			beforeAssertion := p.Balance.Sub(p.Amount)
			if !prevBalance.Equal(beforeAssertion) {
				openingBalHelper := ""
				if !foundOpeningBalance {
					openingBalHelper = " (opening balances were auto-generated)"
				}
				err := errors.Errorf(
					"Failed balance assertion for account '%s'%s: difference = %s\nTransaction 1:\n%s\nTransaction 2:\n%s",
					p.Account,
					openingBalHelper,
					prevBalance.Sub(beforeAssertion),
					previousAccountTxn[p.Account],
					txn,
				)
				return NewValidateError(ix, err)
			}
			balances[p.Account] = *p.Balance
			previousAccountTxn[p.Account] = txn
		}
	}
	return nil
}

func duplicateTransactionError(id string) error {
	return errors.Errorf("Duplicate transaction IDs found: %s", id)
}

// LastTransactionTime returns the last transactions Date field. Returns 0 if there are no transactions
func (l *Ledger) LastTransactionTime() time.Time {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.transactions) == 0 {
		var t time.Time
		return t
	}
	return l.transactions[len(l.transactions)-1].Date
}

// AddTransactions attempts to add the provided transactions.
// Returns an error if the ledger fails validation (i.e. fail balance assertions).
// In the event of an error, attempts to add all valid transactions up to the error.
func (l *Ledger) AddTransactions(txns []Transaction) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	idSet, newTransactions, _ := makeIDSet(append(l.transactions, txns...))
	Transactions(newTransactions).Sort()
	testLedger := &Ledger{
		idSet:        idSet,
		transactions: newTransactions,
	}
	err := testLedger.Validate()
	if err != nil {
		validateErr, ok := err.(Error)
		if ok && validateErr.firstFailedTxnIndex >= len(l.transactions) {
			// if this is our own error type and the first failed transaction is in the new txn set, partially apply the new txns
			newTransactions = newTransactions[:validateErr.firstFailedTxnIndex]
			idSet, _, _ = makeIDSet(newTransactions)
		} else {
			return err
		}
	}
	l.idSet = idSet
	l.transactions = newTransactions
	return err
}

// Balances returns a cumulative balance sheet for all accounts over the given time period.
func (l *Ledger) Balances() (start, end time.Time, balances map[string][]decimal.Decimal) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.transactions) == 0 {
		return
	}
	balances = make(map[string][]decimal.Decimal)
	start, end = l.transactions[0].Date, l.transactions[0].Date
	for _, txn := range l.transactions {
		if txn.Date.Before(start) {
			start = txn.Date
		}
		if txn.Date.After(end) {
			end = txn.Date
		}
	}

	const granularity = 10
	interval := end.Sub(start) / (granularity - 1) // subtract 1 to handle txns on end date

	for _, txn := range l.transactions {
		index := txn.Date.Sub(start) / interval
		for _, p := range txn.Postings {
			if _, ok := balances[p.Account]; !ok {
				balances[p.Account] = make([]decimal.Decimal, granularity)
			}
			balances[p.Account][index] = balances[p.Account][index].Add(p.Amount)
		}
	}

	// convert to cumulative sum
	for _, amounts := range balances {
		for i := range amounts {
			if i != 0 {
				amounts[i] = amounts[i].Add(amounts[i-1])
			}
		}
	}
	return
}

// UpdateTransaction replaces a transaction where ID is 'id' with 'transaction'
// The new transaction must be valid
func (l *Ledger) UpdateTransaction(id string, transaction Transaction) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.idSet[id] {
		return errors.New("Transaction not found by ID: " + id)
	}
	txnIndex := -1
	for ix, txn := range l.transactions {
		if txn.ID() == id {
			txnIndex = ix
			break
		}
		for _, p := range txn.Postings {
			if p.ID() == id {
				txnIndex = ix
				break
			}
		}
	}
	if txnIndex == -1 {
		panic("ID set out of sync with ledger transactions")
	}

	txnCopy := l.transactions[txnIndex]
	if transaction.Comment != "" {
		txnCopy.Comment = transaction.Comment
	}
	if len(transaction.Postings) > 0 {
		txnCopy.Postings = transaction.Postings
	}
	if err := txnCopy.Validate(); err != nil {
		return err
	}

	l.transactions[txnIndex] = txnCopy
	return nil
}

// UpdateAccount changes all transactions' accounts matching oldAccount to newAccount
func (l *Ledger) UpdateAccount(oldAccount, newAccount string) error {
	if newAccount == "" {
		return errors.New("New account name must not be empty")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for t := range l.transactions {
		for p := range l.transactions[t].Postings {
			if l.transactions[t].Postings[p].Account == oldAccount {
				l.transactions[t].Postings[p].Account = newAccount
			}
		}
	}
	return nil
}
