package budget

import (
	"strconv"
	"sync"
	"time"

	"github.com/johnstarich/sage/pipe"
	"github.com/johnstarich/sage/plaindb"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

// Store manages budgets
type Store struct {
	mu     sync.Mutex
	bucket plaindb.Bucket
}

// NewStore returns the budgets bucket
func NewStore(db plaindb.DB) (*Store, error) {
	bucket, err := db.Bucket("budgets", "2", &storeUpgrader{})
	return &Store{
		bucket: bucket,
	}, err
}

func formatYear(year int) string {
	return strconv.FormatInt(int64(year), 10)
}

func (s *Store) Month(year int, month time.Month) (Accounts, error) {
	var budget Budget
	var accounts Accounts
	return accounts, pipe.OpFuncs{
		func() error {
			var err error
			budget, err = s.getYear(year)
			return err
		},
		func() error {
			accounts = budget.Month(month)
			return nil
		},
	}.Do()
}

func (s *Store) getYear(year int) (Budget, error) {
	return s.getYearWithTime(time.Now, year)
}

func (s *Store) getYearWithTime(getTime func() time.Time, year int) (Budget, error) {
	// NOTE: If getYear is called for a non-existent year, it will generate a new one *without* inserting it. Be sure to lock to prevent races on write calls
	var budget Budget
	found, err := s.bucket.Get(formatYear(year), &budget)
	if err != nil {
		return budget, err
	}
	if found {
		return budget, nil
	}
	now := getTime().UTC()
	if year > now.Year() {
		return budget, errors.Errorf("No budget found for year: %d", year)
	}

	// generate new year with budgets carried over from the most recent year
	var closestBudget Budget
	err = s.bucket.Iter(&budget, func(string) bool {
		if budget.Year() < year && (closestBudget == nil || budget.Year() > closestBudget.Year()) {
			closestBudget = budget
			return closestBudget.Year() != year-1
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if closestBudget == nil {
		return New(year), nil
	}
	for closestBudget.Year() != year {
		closestBudget = closestBudget.NextYear()
	}
	return closestBudget, nil
}

func (s *Store) SetMonth(year int, month time.Month, account string, budget decimal.Decimal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var yearBudget Budget
	return pipe.OpFuncs{
		func() error {
			var err error
			yearBudget, err = s.getYear(year)
			return err
		},
		func() error {
			return yearBudget.SetMonth(month, account, budget)
		},
		func() error {
			return s.bucket.Put(formatYear(year), yearBudget)
		},
	}.Do()
}

func (s *Store) RemoveMonth(year int, month time.Month, account string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var budget Budget
	return pipe.OpFuncs{
		func() error {
			var err error
			budget, err = s.getYear(year)
			return err
		},
		func() error {
			return budget.RemoveMonth(month, account)
		},
		func() error {
			return s.bucket.Put(formatYear(year), budget)
		},
	}.Do()
}
