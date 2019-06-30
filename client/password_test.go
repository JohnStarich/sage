package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ json.Marshaler = Password("")

func TestPasswordMarshalsToNothing(t *testing.T) {
	result, err := json.Marshal(Password("testing"))
	require.NoError(t, err)
	assert.Equal(t, "null", string(result))
}
