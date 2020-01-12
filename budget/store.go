package budget

import (
	"strconv"
	"sync"
	"time"

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
	budget, err := s.getYear(year)
	if err != nil {
		return nil, err
	}
	return budget.Month(month), nil
}

func (s *Store) getYear(year int) (Budget, error) {
	// NOTE: If getYear is called for a non-existent year, it will generate a new one *without* inserting it. Be sure to lock to prevent races on write calls
	var budget Budget
	found, err := s.bucket.Get(formatYear(year), &budget)
	if err != nil {
		return budget, err
	}
	if found {
		return budget, nil
	}
	now := time.Now().UTC()
	if year > now.Year() {
		return budget, errors.Errorf("No budget found for year: %d", year)
	}

	// generate new year with budgets carried over from the most recent year
	var closestBudget Budget
	err = s.bucket.Iter(&budget, func(string) bool {
		if budget.Year() < year && (closestBudget == nil || budget.Year() > closestBudget.Year()) {
			closestBudget = budget
		}
		return closestBudget.Year() != year-1
	})
	if err != nil {
		return nil, err
	}
	for closestBudget.Year() != year {
		closestBudget = closestBudget.NextYear()
	}
	return closestBudget, nil
}

func (s *Store) SetMonth(year int, month time.Month, account string, budget decimal.Decimal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	yearBudget, err := s.getYear(year)
	if err != nil {
		return err
	}
	err = yearBudget.SetMonth(month, account, budget)
	if err != nil {
		return err
	}
	return s.bucket.Put(formatYear(year), yearBudget)
}

func (s *Store) RemoveMonth(year int, month time.Month, account string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	budget, err := s.getYear(year)
	if err != nil {
		return err
	}
	err = budget.RemoveMonth(month, account)
	if err != nil {
		return err
	}
	return s.bucket.Put(formatYear(year), budget)
}
