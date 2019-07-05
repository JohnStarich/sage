package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientCache(t *testing.T) {
	inst := baseInstitution{url: "some URL"}
	client, err := clientForInstitution(inst)
	require.NoError(t, err)

	client2, err := clientForInstitution(inst)
	require.NoError(t, err)
	assert.True(t, client == client2, "The client must be the same pointer")

	_, err = clientForInstitution(baseInstitution{config: Config{OFXVersion: "not a real version"}})
	assert.Error(t, err)
}
