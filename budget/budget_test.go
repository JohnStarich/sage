package budget

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const someYear = 2020

func TestNew(t *testing.T) {
	b := New(someYear)
	assert.Equal(t, &budget{
		BudgetYear: someYear,
		Months:     map[time.Month]Accounts{},
	}, b)
}

func TestYear(t *testing.T) {
	assert.Equal(t, someYear, (&budget{BudgetYear: someYear}).Year())
}

var dec = decimal.NewFromFloat

func TestSetMonths(t *testing.T) {
	getTime := getTimeFn(someYear, time.January)
	type monthAccountsPair struct {
		time.Month
		Accounts
	}

	t.Run("SetMonth with time.Now", func(t *testing.T) {
		b := New(someYear)
		assert.Error(t, b.SetMonth(time.January, "", dec(10)), "'account' should be invalid")
	})

	for _, tc := range []struct {
		description string
		months      []monthAccountsPair
		expected    map[time.Month]Accounts
		err         string
	}{
		{
			description: "happy path",
			months: []monthAccountsPair{
				{time.February, Accounts{"expenses": dec(10)}},
			},
			expected: map[time.Month]Accounts{
				time.February: {"expenses": dec(10)},
			},
		},
		{
			description: "mixed case",
			months: []monthAccountsPair{
				{time.February, Accounts{"eXpEnSeS": dec(10)}},
			},
			expected: map[time.Month]Accounts{
				time.February: {"expenses": dec(10)},
			},
		},
		{
			description: "invalid month",
			months: []monthAccountsPair{
				{0, Accounts{"expenses": dec(10)}},
			},
			err: "Invalid month: 0",
		},
		{
			description: "empty account",
			months: []monthAccountsPair{
				{time.February, Accounts{"": dec(10)}},
			},
			err: "Account must be specified",
		},
		{
			description: "happy path - multiple budgets",
			months: []monthAccountsPair{
				{time.January, Accounts{"expenses": decimal.NewFromFloat(10)}},
				{time.February, Accounts{"expenses": decimal.NewFromFloat(20)}},
			},
			expected: map[time.Month]Accounts{
				time.January:  {"expenses": decimal.NewFromFloat(10)},
				time.February: {"expenses": decimal.NewFromFloat(20)},
			},
		},
		{
			description: "multiple budgets in-order",
			months: []monthAccountsPair{
				{time.January, Accounts{"expenses": decimal.NewFromFloat(10)}},
				{time.March, Accounts{"revenues": decimal.NewFromFloat(20)}},
			},
			expected: map[time.Month]Accounts{
				time.January: {"expenses": decimal.NewFromFloat(10)},
				time.March: {
					"expenses": decimal.NewFromFloat(10),
					"revenues": decimal.NewFromFloat(20),
				},
			},
		},
		{
			description: "multiple budgets reverse-order",
			months: []monthAccountsPair{
				{time.March, Accounts{"revenues": decimal.NewFromFloat(20)}},
				{time.January, Accounts{"expenses": decimal.NewFromFloat(10)}},
			},
			expected: map[time.Month]Accounts{
				time.January: {"expenses": decimal.NewFromFloat(10)},
				time.March:   {"revenues": decimal.NewFromFloat(20)},
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			b := New(someYear).(*budget)
			for _, pair := range tc.months {
				for account, budget := range pair.Accounts {
					err := b.setMonth(getTime, pair.Month, account, budget)
					if tc.err == "" {
						require.NoError(t, err)
					} else {
						require.Error(t, err)
						assert.Equal(t, tc.err, err.Error())
						return
					}
				}
			}
			assert.Equal(t, &budget{
				BudgetYear: someYear,
				Months:     tc.expected,
			}, b)
		})
	}
}

func getTimeFn(year int, month time.Month) func() time.Time {
	return func() time.Time {
		return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	}
}

func TestSetBetweenMonths(t *testing.T) {
	t.Run("set on previous month should not affect later months", func(t *testing.T) {
		getTime := getTimeFn(someYear, time.December)
		b := New(someYear).(*budget)
		require.NoError(t, b.setMonth(getTime, time.January, "expenses", dec(10)))
		assert.Equal(t, dec(10), b.Month(time.February).Get("expenses"))

		require.NoError(t, b.setMonth(getTime, time.January, "expenses", dec(25)))
		assert.Equal(t, dec(10), b.Month(time.February).Get("expenses"))
	})

	t.Run("set on current month should NOT set future month", func(t *testing.T) {
		getTime := getTimeFn(someYear, time.February)
		b := New(someYear).(*budget)
		require.NoError(t, b.setMonth(getTime, time.February, "expenses", dec(10)))
		assert.Equal(t, dec(10), b.Month(time.March).Get("expenses"))

		require.NoError(t, b.setMonth(getTime, time.February, "expenses", dec(25)))
		assert.Equal(t, dec(25), b.Month(time.March).Get("expenses"))
	})
}

func TestMonth(t *testing.T) {
	for _, tc := range []struct {
		description string
		months      map[time.Month]Accounts
		expected    map[time.Month]Accounts
	}{
		{
			description: "no budgets",
			months:      map[time.Month]Accounts{},
			expected:    map[time.Month]Accounts{},
		},
		{
			description: "one budget carries over",
			months: map[time.Month]Accounts{
				time.November: {"expenses": dec(10)},
			},
			expected: map[time.Month]Accounts{
				time.November: {"expenses": dec(10)},
				time.December: {"expenses": dec(10)},
			},
		},
		{
			description: "latest budget carries over",
			months: map[time.Month]Accounts{
				time.September: {"expenses": dec(30), "revenues": dec(20)},
				time.November:  {"expenses": dec(10)},
			},
			expected: map[time.Month]Accounts{
				time.September: {"expenses": dec(30), "revenues": dec(20)},
				time.October:   {"expenses": dec(30), "revenues": dec(20)},
				time.November:  {"expenses": dec(10)},
				time.December:  {"expenses": dec(10)},
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			b := &budget{
				BudgetYear: someYear,
				Months:     tc.months,
			}
			for month := time.January; month < time.December; month++ {
				assert.Equal(t, tc.expected[month], b.Month(month))
			}
		})
	}
}

func TestNextYear(t *testing.T) {
	next := (&budget{
		BudgetYear: someYear,
		Months: map[time.Month]Accounts{
			time.February: {"expenses": dec(50)},
			time.November: {"expenses": dec(10), "revenues": dec(20)},
		},
	}).NextYear()
	assert.Equal(t, &budget{
		BudgetYear: someYear + 1,
		Months: map[time.Month]Accounts{
			time.January: {"expenses": dec(10), "revenues": dec(20)},
		},
	}, next)
}

func TestRemoveMonth(t *testing.T) {
	getTime := getTimeFn(someYear, time.December)
	b := New(someYear).(*budget)
	require.NoError(t, b.setMonth(getTime, time.February, "expenses", dec(10)))

	assert.NoError(t, b.removeMonth(getTime, time.March, "expenses"))
	assert.NotContains(t, b.Month(time.March), "expenses", "Budget should be removed")
	assert.Equal(t, dec(10), b.Month(time.February).Get("expenses"), "Prior month should be unaffected")

	assert.Error(t, b.RemoveMonth(time.January, ""), "'account' should be invalid")
	assert.Error(t, b.RemoveMonth(0, "expenses"), "'month' should be invalid")
}
