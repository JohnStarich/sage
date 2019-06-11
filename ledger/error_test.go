package ledger

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidateError(t *testing.T) {
	assert.Nil(t, NewValidateError(1, nil))

	e := errors.New("some error")
	validateErr := NewValidateError(1, e)
	require.Error(t, validateErr)
	assert.Equal(t, "Failed to validate ledger at transaction index #1: "+e.Error(), validateErr.Error())
}
