package budget

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type v1Budget struct {
	Account string
	Budget  decimal.Decimal
}

type storeUpgrader struct{}

func (u *storeUpgrader) Parse(dataVersion, id string, data json.RawMessage) (interface{}, error) {
	switch dataVersion {
	case "1":
		var budget v1Budget
		err := json.Unmarshal(data, &budget)
		return budget, err
	case "2":
		var budget *budget
		err := json.Unmarshal(data, &budget)
		return budget, err
	default:
		return nil, errors.Errorf("Unsupported version: %q", dataVersion)
	}
}

func (u *storeUpgrader) UpgradeAll(dataVersion string, data map[string]interface{}) (newVersion string, newData map[string]interface{}, err error) {
	const (
		// Hard-coding 2019 since budgets in v1 format are probably not applicable before the year budgets were added.
		year    = 2019
		yearStr = "2019"
		month   = time.January
	)
	switch dataVersion {
	case "1":
		budget := New(year).(*budget)
		budget.Months[month] = make(Accounts)
		for _, value := range data {
			monthBudget := value.(v1Budget)
			budget.Months[month].set(monthBudget.Account, monthBudget.Budget)
		}

		newData = make(map[string]interface{}, 1)
		newData[yearStr] = budget
		return "2", newData, nil
	default:
		return dataVersion, data, nil
	}
}

func (u *storeUpgrader) Upgrade(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
	panic("Not implemented")
}
