package vcs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-git.v4"
)

func TestFile(t *testing.T) {
	cleanup := func() {
		require.NoError(t, os.RemoveAll("./testdb"))
	}
	cleanup()
	defer cleanup()

	repoInt, err := Open("./testdb")
	require.NoError(t, err)
	repo := repoInt.(*syncRepo)

	f := repo.File("./testdb/bucket.json")
	buf, err := f.Read()
	assert.Empty(t, buf)
	assert.NoError(t, err)

	err = f.Write([]byte("hi there"))
	assert.NoError(t, err)

	buf, err = f.Read()
	require.NoError(t, err)
	assert.Equal(t, "hi there", string(buf))

	commits, err := repo.repo.Log(&git.LogOptions{})
	require.NoError(t, err)
	commit, err := commits.Next()
	require.NoError(t, err)
	files, err := commit.Files()
	require.NoError(t, err)
	file, err := files.Next()
	require.NoError(t, err)
	contents, err := file.Contents()
	require.NoError(t, err)

	assert.Equal(t, "bucket.json", file.Name)
	assert.Equal(t, "hi there", contents)
	assert.Equal(t, "Update ./testdb/bucket.json", commit.Message)
}
