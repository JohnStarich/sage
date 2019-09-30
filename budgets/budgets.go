package budgets

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

	"github.com/johnstarich/sage/plaindb"
	"github.com/shopspring/decimal"
)

// Budget is a monthly budget tracker
type Budget struct {
	Account string
	Budget  decimal.Decimal
}

func Validate(b Budget) error {
	if b.Account == "" {
		return errors.New("Account name is required for a budget")
	}
	return nil
}

// Store manages budgets
type Store struct {
	bucket plaindb.Bucket
}

func New(db plaindb.DB) (*Store, error) {
	bucket, err := db.Bucket("budgets", "1", &storeUpgrader{})
	return &Store{
		bucket: bucket,
	}, err
}

func (s *Store) Get(account string) (Budget, error) {
	var budget Budget
	found, err := s.bucket.Get(account, &budget)
	if err != nil {
		return budget, err
	}
	if !found {
		return budget, errors.Errorf("No budget found for account: %q", account)
	}
	return budget, nil
}

func (s *Store) GetAll() ([]Budget, error) {
	var budgets []Budget
	var budget Budget
	err := s.bucket.Iter(&budget, func(string) bool {
		budgets = append(budgets, budget)
		return true
	})
	return budgets, err
}

func (s *Store) Add(b Budget) error {
	if err := Validate(b); err != nil {
		return err
	}
	b.Account = strings.ToLower(b.Account)
	var existingBudget Budget
	if found, _ := s.bucket.Get(b.Account, &existingBudget); found {
		return errors.Errorf("Budget already exists: %q", b.Account)
	}
	return s.bucket.Put(b.Account, b)
}

func (s *Store) Update(account string, b Budget) error {
	if err := Validate(b); err != nil {
		return err
	}
	var existingBudget Budget
	if found, _ := s.bucket.Get(account, &existingBudget); !found {
		return errors.Errorf("Budget not found: %q", account)
	}
	if account != b.Account {
		// if renaming, don't clobber an existing budget
		if found, _ := s.bucket.Get(b.Account, &existingBudget); found {
			return errors.Errorf("Budget already exists: %q", b.Account)
		}
	}

	err := s.bucket.Put(b.Account, b)
	if err != nil {
		return err
	}
	if account != b.Account {
		// if renamed, remove budget with old name
		return s.bucket.Put(account, nil)
	}
	return nil
}

func (s *Store) Remove(account string) error {
	var budget Budget
	if found, _ := s.bucket.Get(account, &budget); !found {
		return errors.Errorf("Budget not found: %q", account)
	}
	return s.bucket.Put(account, nil)
}

type storeUpgrader struct{}

func (u *storeUpgrader) Parse(dataVersion, id string, data json.RawMessage) (interface{}, error) {
	if dataVersion != "1" {
		return nil, errors.Errorf("Unsupported version: %q", dataVersion)
	}
	var budget Budget
	err := json.Unmarshal(data, &budget)
	return budget, err
}

func (u *storeUpgrader) Upgrade(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
	if dataVersion != "1" {
		return "", nil, errors.Errorf("Unsupported version: %q", dataVersion)
	}
	return dataVersion, data, nil
}
