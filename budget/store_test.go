package budget

import (
	"testing"

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
