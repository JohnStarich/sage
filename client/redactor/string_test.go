package redactor

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordMarshalsToNothing(t *testing.T) {
	result, err := json.Marshal(String("testing"))
	require.NoError(t, err)
	assert.Equal(t, "null", string(result))
}

func TestPasswordMarshalsToSomething(t *testing.T) {
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	require.NoError(t, encoder.Encode(String("testing")))
	assert.Contains(t, buf.String(), "testing")
}

func TestPasswordUnmarshals(t *testing.T) {
	var p String
	err := json.Unmarshal([]byte(`"hey there"`), &p)
	require.NoError(t, err)
	assert.Equal(t, String("hey there"), p)

	someStruct := struct {
		Username string
		Password String
	}{}
	err = json.Unmarshal([]byte(`{"Username":"username", "Password":"password"}`), &someStruct)
	require.NoError(t, err)

	assert.Equal(t, "username", someStruct.Username)
	assert.Equal(t, String("password"), someStruct.Password)
}
