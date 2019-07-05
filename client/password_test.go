package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ json.Marshaler = &Password{}
var _ json.Unmarshaler = &Password{}

func TestPasswordMarshalsToNothing(t *testing.T) {
	result, err := json.Marshal(NewPassword("testing"))
	require.NoError(t, err)
	assert.Equal(t, "null", string(result))
}

func TestPasswordUnmarshals(t *testing.T) {
	var p Password
	err := json.Unmarshal([]byte(`"hey there"`), &p)
	require.NoError(t, err)
	assert.Equal(t, NewPassword("hey there"), &p)

	someStruct := struct {
		Username string
		Password *Password
	}{}
	err = json.Unmarshal([]byte(`{"Username":"username", "Password":"password"}`), &someStruct)
	require.NoError(t, err)

	assert.Equal(t, "username", someStruct.Username)
	assert.Equal(t, NewPassword("password"), someStruct.Password)
}

func TestPasswordSet(t *testing.T) {
	p := NewPassword("some password")
	p.Set(NewPassword("some other password"))
	assert.Equal(t, NewPassword("some other password"), p)
}
