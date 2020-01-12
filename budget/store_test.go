package budget

import (
	"testing"
	"time"

	"github.com/johnstarich/sage/plaindb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockDBStore(t *testing.T) *Store {
	db := plaindb.NewMockDB(plaindb.MockConfig{FileReader: func(fileName string) ([]byte, error) {
		return []byte(`{}`), nil
	}})
	store, err := NewStore(db)
	require.NoError(t, err)
	return store
}

func TestNewStore(t *testing.T) {
	store := mockDBStore(t)
	assert.NotNil(t, store.bucket)
}

func TestStoreMonth(t *testing.T) {
	store := mockDBStore(t)
	b := New(someYear).(*budget)
	require.NoError(t, b.SetMonth(time.January, "expenses", dec(10)))
	require.NoError(t, store.bucket.Put(formatYear(someYear), b))

	accounts, err := store.Month(someYear, time.January)
	require.NoError(t, err)
	assert.NotEmpty(t, b.Months[time.January])
	assert.Equal(t, b.Months[time.January], accounts)
}

func TestGetYear(t *testing.T) {
	t.Run("bucket error", func(t *testing.T) {
		store := mockDBStore(t)
		require.NoError(t, store.bucket.Put(formatYear(someYear), "wrong type"))
		_, err := store.getYear(someYear)
		assert.Error(t, err)
	})

	t.Run("future year error", func(t *testing.T) {
		store := mockDBStore(t)
		_, err := store.getYearWithTime(getTimeFn(someYear, time.January), someYear+1)
		require.Error(t, err)
		assert.Equal(t, "No budget found for year: 2021", err.Error())
	})

	t.Run("generate year from previous year", func(t *testing.T) {
		store := mockDBStore(t)

		previousDecade := New(someYear - 10)
		require.NoError(t, previousDecade.SetMonth(time.February, "expenses", dec(10)))
		require.NoError(t, store.bucket.Put(formatYear(someYear-10), previousDecade))

		previousYear := New(someYear - 3)
		require.NoError(t, previousYear.SetMonth(time.March, "revenues", dec(10)))
		require.NoError(t, store.bucket.Put(formatYear(someYear-3), previousYear))

		b, err := store.getYear(someYear)
		require.NoError(t, err)
		require.IsType(t, (*budget)(nil), b)
		assert.Equal(t, dec(10), b.(*budget).Months[time.January]["revenues"])
	})

	t.Run("generate new year", func(t *testing.T) {
		store := mockDBStore(t)

		b, err := store.getYear(someYear)
		require.NoError(t, err)
		require.IsType(t, (*budget)(nil), b)
		assert.Empty(t, b.(*budget).Months)
	})
}

func TestStoreSetMonth(t *testing.T) {
	store := mockDBStore(t)
	assert.NoError(t, store.SetMonth(someYear, time.February, "expenses", dec(10)))
	accounts, err := store.Month(someYear, time.February)
	require.NoError(t, err)
	assert.Equal(t, dec(10), accounts.Get("expenses"))
}

func TestStoreRemoveMonth(t *testing.T) {
	store := mockDBStore(t)
	require.NoError(t, store.SetMonth(someYear, time.February, "expenses", dec(10)))
	accounts, err := store.Month(someYear, time.February)
	require.NoError(t, err)
	require.NotEmpty(t, accounts)

	assert.NoError(t, store.RemoveMonth(someYear, time.February, "expenses"))
	accounts, err = store.Month(someYear, time.February)
	require.NoError(t, err)
	assert.Empty(t, accounts)
}
