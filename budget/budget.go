package budget

import (
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/shopspring/decimal"
)

// Budget is a monthly budget tracker for a particular year
type Budget interface {
	Year() int
	NextYear() Budget

	Month(month time.Month) Accounts
	SetMonth(month time.Month, account string, budget decimal.Decimal) error
	RemoveMonth(month time.Month, account string) error
}

type budget struct {
	mu sync.RWMutex

	BudgetYear int
	Months     map[time.Month]Accounts
}

// Accounts is a mapping from account names to budget amounts
type Accounts map[string]decimal.Decimal

func (a Accounts) set(account string, budget decimal.Decimal) {
	a[strings.ToLower(account)] = budget
}

func (a Accounts) remove(account string) {
	delete(a, strings.ToLower(account))
}

func (a Accounts) Get(account string) decimal.Decimal {
	return a[strings.ToLower(account)]
}

// New returns a budget that's bound to a given year.
// Every year can have monthly budgets for various accounts. Each new month copies the previous
// month of budgets automatically.
func New(year int) Budget {
	return &budget{
		BudgetYear: year,
		Months:     make(map[time.Month]Accounts),
	}
}

// NextYear creates a new budget for the following year, inheriting the latest budgets in 'b'
func (b *budget) NextYear() Budget {
	b.mu.RLock()
	defer b.mu.RUnlock()
	next := New(b.Year() + 1).(*budget)
	// don't need to lock 'next' since nobody else has a reference to it yet
	next.Months[time.January] = make(Accounts)
	copyAccounts(next.Months[time.January], b.Month(time.December))
	return next
}

func (b *budget) Month(month time.Month) Accounts {
	if _, exists := b.Months[month]; exists {
		b.mu.RLock()
		defer b.mu.RUnlock()
		return b.Months[month]
	}
	return b.allMonths()[month-1]
}

func (b *budget) allMonths() []Accounts {
	b.mu.RLock()
	defer b.mu.RUnlock()
	months := make([]Accounts, time.December)
	var previousAccounts Accounts
	for i := range months {
		accounts, exists := b.Months[time.Month(i+1)]
		if !exists {
			accounts = previousAccounts
		}
		months[i] = accounts
		previousAccounts = accounts
	}
	return months
}

func (b *budget) SetMonth(month time.Month, account string, budget decimal.Decimal) error {
	return b.setMonth(time.Now, month, account, budget)
}

func (b *budget) setMonth(getTime func() time.Time, month time.Month, account string, budget decimal.Decimal) error {
	if month < time.January || month > time.December {
		return errors.Errorf("Invalid month: %d", month)
	}
	if account == "" {
		return errors.New("Account must be specified")
	}

	b.ensureMonth(getTime, month)
	b.mu.Lock()
	b.Months[month].set(account, budget)
	b.mu.Unlock()
	return nil
}

func (b *budget) ensureMonth(getTime func() time.Time, month time.Month) {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, exists := b.Months[month]
	switch {
	case !exists:
		accounts := make(Accounts)
		for i := month - 1; i >= time.January; i-- {
			if _, monthExists := b.Months[i-1]; monthExists {
				copyAccounts(accounts, b.Months[i-1])
				break
			}
		}
		b.Months[month] = accounts
	case exists && month < time.December:
		// copy current budgets to following month, so SetMonth doesn't actually change more than one month's budgets
		now := getTime()
		if now.Year() > b.BudgetYear || (now.Year() == b.BudgetYear && now.Month() > month) {
			_, nextExists := b.Months[month+1]
			if !nextExists {
				b.Months[month+1] = make(Accounts)
				copyAccounts(b.Months[month+1], b.Months[month])
			}
		}
	}
}

func (b *budget) RemoveMonth(month time.Month, account string) error {
	return b.removeMonth(time.Now, month, account)
}

func (b *budget) removeMonth(getTime func() time.Time, month time.Month, account string) error {
	if month < time.January || month > time.December {
		return errors.Errorf("Invalid month: %d", month)
	}
	if account == "" {
		return errors.New("Account must be specified")
	}

	b.ensureMonth(getTime, month)
	b.mu.Lock()
	b.Months[month].remove(account)
	b.mu.Unlock()
	return nil
}

func (b *budget) Year() int {
	return b.BudgetYear
}

func copyAccounts(dest, src Accounts) {
	for account, budget := range src {
		dest[account] = budget
	}
}
