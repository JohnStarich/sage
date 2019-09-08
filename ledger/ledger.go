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
	idSet        map[string]*Transaction
	mu           sync.RWMutex
}

// New creates a ledger with the given transactions. Must not contain any duplicate IDs
func New(transactions []Transaction) (*Ledger, error) {
	transactionPtrs := makeTransactionPtrs(transactions)
	idSet, _, duplicates := makeIDSet(transactionPtrs)
	if len(duplicates) > 0 {
		return nil, duplicateTransactionError(strings.Join(duplicates, ", "))
	}
	return &Ledger{
		transactions: transactionPtrs,
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

// makeTransactionPtrs converts to a slice of txn pointers. NOTE: does not copy the underlying txn
func makeTransactionPtrs(transactions []Transaction) []*Transaction {
	transactionPtrs := make([]*Transaction, len(transactions))
	for i := range transactions {
		transactionPtrs[i] = &transactions[i]
	}
	return transactionPtrs
}

func dereferenceTransactions(transactionPtrs Transactions) []Transaction {
	transactions := make([]Transaction, len(transactionPtrs))
	for i := range transactionPtrs {
		transactions[i] = *transactionPtrs[i]
	}
	return transactions
}

func makeIDSet(transactions []*Transaction) (idSet map[string]*Transaction, uniqueTxns []*Transaction, duplicates []string) {
	idSet = make(map[string]*Transaction, len(transactions)*2)
	for _, transaction := range transactions {
		txnIsDupe := false
		if id := transaction.ID(); id != "" {
			if idSet[id] != nil {
				txnIsDupe = true
				duplicates = append(duplicates, id)
			}
			idSet[id] = transaction
		}
		for _, posting := range transaction.Postings {
			if id := posting.ID(); id != "" {
				if idSet[id] != nil {
					txnIsDupe = true
					duplicates = append(duplicates, id)
				}
				idSet[id] = transaction
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
	sortedTxns := make(Transactions, len(l.transactions))
	copy(sortedTxns, l.transactions)
	sortedTxns.Sort()
	var buf bytes.Buffer
	for _, txn := range sortedTxns {
		buf.WriteString(txn.String())
		buf.WriteRune('\n')
	}
	return buf.String()
}

// Validate returns a descriptive error should anything be wrong with the current ledger's transactions
func (l *Ledger) Validate() error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for ix, txn := range l.transactions {
		if err := txn.Validate(); err != nil {
			return NewValidateError(ix, err)
		}
	}
	return nil
}

func duplicateTransactionError(id string) error {
	return errors.Errorf("Duplicate transaction IDs found: %s", id)
}

// FirstTransactionTime returns the first transaction's Date field. Returns 0 if there are no transactions
func (l *Ledger) FirstTransactionTime() time.Time {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.transactions) == 0 {
		var t time.Time
		return t
	}
	return l.transactions[0].Date
}

// LastTransactionTime returns the last transaction's Date field. Returns 0 if there are no transactions
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
	transactionPtrs := makeTransactionPtrs(txns)
	for i := range transactionPtrs {
		transactionPtrs[i].Date = transactionPtrs[i].Date.UTC()
	}
	idSet, newTransactions, _ := makeIDSet(append(l.transactions, transactionPtrs...))
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
// Current interval is monthly.
func (l *Ledger) Balances() (start, end *time.Time, balances map[string][]decimal.Decimal) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.transactions) == 0 {
		return
	}
	balances = make(map[string][]decimal.Decimal)
	start, end = timePtr(l.transactions[0].Date), timePtr(l.transactions[0].Date)
	for _, txn := range l.transactions {
		if txn.Date.Before(*start) {
			start = timePtr(txn.Date)
		}
		if txn.Date.After(*end) {
			end = timePtr(txn.Date)
		}
	}

	getMonthNum := func(t time.Time) int {
		return int(t.Month()) - 1 + 12*t.Year()
	}
	startMonthNum := getMonthNum(*start)
	intervals := getMonthNum(*end) - startMonthNum + 1

	for _, txn := range l.transactions {
		index := getMonthNum(txn.Date) - startMonthNum
		for _, p := range txn.Postings {
			if _, ok := balances[p.Account]; !ok {
				balances[p.Account] = make([]decimal.Decimal, intervals)
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

func timePtr(t time.Time) *time.Time {
	return &t
}

// UpdateTransaction replaces a transaction where ID is 'id' with 'transaction'
// The new transaction must be valid
func (l *Ledger) UpdateTransaction(id string, transaction Transaction) error {
	if id == OpeningBalanceID {
		return NewValidateError(0, errors.New("Update opening balances with /api/v1/updateOpeningBalance"))
	}
	return l.updateTransaction(id, transaction)
}

func (l *Ledger) updateTransaction(id string, transaction Transaction) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	existingTxn := l.idSet[id]
	if existingTxn == nil {
		return errors.New("Transaction not found by ID: " + id)
	}

	txnCopy := *existingTxn
	if !transaction.Date.IsZero() {
		txnCopy.Date = transaction.Date.UTC()
	}
	if transaction.Comment != "" {
		txnCopy.Comment = transaction.Comment
	}
	if len(transaction.Postings) > 0 {
		txnCopy.Postings = transaction.Postings
	}
	if err := txnCopy.Validate(); err != nil {
		return err
	}

	*existingTxn = txnCopy
	l.transactions.Sort()
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

// OpeningBalances attempts to find the opening balances transaction and return it
func (l *Ledger) OpeningBalances() (opening Transaction, found bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	existingTxn := l.idSet[OpeningBalanceID]
	if existingTxn == nil {
		return
	}
	return *existingTxn, true
}

// UpdateOpeningBalance inserts or updates an account's opening balance for this ledger.
func (l *Ledger) UpdateOpeningBalance(opening Transaction) error {
	if err := opening.Validate(); err != nil {
		return err
	}

	if opening.Date.IsZero() {
		return errors.New("Opening transaction date must be set")
	}

	if !isOpeningTransaction(opening) {
		return errors.New("One of the transaction's postings must have an ID of " + OpeningBalanceID)
	}

	// only allow some fields in the update for now
	newOpening := Transaction{
		Date:     opening.Date,
		Payee:    "* Opening Balance",
		Postings: opening.Postings,
	}

	if l.idSet[OpeningBalanceID] == nil {
		return l.AddTransactions([]Transaction{newOpening})
	}
	return l.updateTransaction(OpeningBalanceID, newOpening)
}

func isOpeningTransaction(txn Transaction) bool {
	for _, p := range txn.Postings {
		if p.IsOpeningBalance() {
			return true
		}
	}
	return false
}
