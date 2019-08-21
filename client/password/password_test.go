package password

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ json.Marshaler = &Password{}
var _ json.Unmarshaler = &Password{}

func TestPasswordMarshalsToNothing(t *testing.T) {
	result, err := json.Marshal(New("testing"))
	require.NoError(t, err)
	assert.Equal(t, "null", string(result))
}

func TestPasswordUnmarshals(t *testing.T) {
	var p Password
	err := json.Unmarshal([]byte(`"hey there"`), &p)
	require.NoError(t, err)
	assert.Equal(t, New("hey there"), &p)

	someStruct := struct {
		Username string
		Password *Password
	}{}
	err = json.Unmarshal([]byte(`{"Username":"username", "Password":"password"}`), &someStruct)
	require.NoError(t, err)

	assert.Equal(t, "username", someStruct.Username)
	assert.Equal(t, New("password"), someStruct.Password)
}

func TestPasswordSet(t *testing.T) {
	p := New("some password")
	p.Set(New("some other password"))
	assert.Equal(t, New("some other password"), p)
}
