package vcs

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func TestOpen(t *testing.T) {
	err := os.RemoveAll("./testdb")
	require.NoError(t, err)
	err = os.MkdirAll("./testdb", 0755)
	require.NoError(t, err)
	err = ioutil.WriteFile("./testdb/bucket.json", []byte(`{}`), 0755)
	require.NoError(t, err)

	repo, err := Open("./testdb")
	require.NoError(t, err)
	assert.NotNil(t, repo)
}

func TestCommitFiles(t *testing.T) {
	cleanup := func() {
		err := os.RemoveAll("./testdb")
		require.NoError(t, err)
	}
	cleanup()
	defer cleanup()
	err := os.MkdirAll("./testdb", 0755)
	require.NoError(t, err)

	repoInt, err := Open("./testdb")
	require.NoError(t, err)
	require.IsType(t, &syncRepo{}, repoInt)
	repo := repoInt.(*syncRepo)

	err = repo.CommitFiles(func() error {
		return ioutil.WriteFile("./testdb/some file.txt", []byte("hello world"), 0755)
	}, "add some file", "./testdb/some file.txt")
	require.NoError(t, err)

	getCount := func() int {
		count := 0
		log, err := repo.repo.Log(&git.LogOptions{})
		require.NoError(t, err)

		err = log.ForEach(func(*object.Commit) error {
			count++
			return nil
		})
		require.NoError(t, err)
		return count
	}
	assert.Equal(t, 1, getCount())

	err = repo.CommitFiles(func() error {
		return ioutil.WriteFile("./testdb/some other file.txt", []byte("hello world"), 0755)
	}, "add some other file", "./testdb/some other file.txt")
	require.NoError(t, err)
	assert.Equal(t, 2, getCount())
}
