package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ json.Marshaler = baseInstitution{}

func TestInstitution(t *testing.T) {
	c := Config{AppID: "some app ID"}
	i := NewInstitution(
		"Some important place",
		"1234",
		"some org",
		"some URL",
		"some user",
		"some password",
		c,
	)

	assert.Equal(t, "some URL", i.URL())
	assert.Equal(t, "some org", i.Org())
	assert.Equal(t, "1234", i.FID())
	assert.Equal(t, "some user", i.Username())
	assert.Equal(t, NewPassword("some password"), i.Password())
	assert.Equal(t, "Some important place", i.Description())
	assert.Equal(t, c, i.Config())
}
