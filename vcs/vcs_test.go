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
	cleanup := func() {
		require.NoError(t, os.RemoveAll("./testdb"))
	}
	cleanup()
	defer cleanup()
	err := os.MkdirAll("./testdb", 0750)
	require.NoError(t, err)
	err = ioutil.WriteFile("./testdb/bucket.json", []byte(`{}`), 0750)
	require.NoError(t, err)

	repoInt, err := Open("./testdb")
	require.NoError(t, err)
	require.IsType(t, &syncRepo{}, repoInt)

	repo := repoInt.(*syncRepo)
	commits, err := repo.repo.Log(&git.LogOptions{})
	require.NoError(t, err)
	count := 0
	err = commits.ForEach(func(commit *object.Commit) error {
		assert.Equal(t, "Initial commit", commit.Message)

		files, err := commit.Files()
		require.NoError(t, err)
		f, err := files.Next()
		require.NoError(t, err)
		contents, err := f.Contents()
		require.NoError(t, err)

		assert.Equal(t, `bucket.json`, f.Name)
		assert.Equal(t, `{}`, contents)
		count++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCommitFiles(t *testing.T) {
	cleanup := func() {
		require.NoError(t, os.RemoveAll("./testdb"))
	}
	cleanup()
	defer cleanup()

	repoInt, err := Open("./testdb")
	require.NoError(t, err)
	require.IsType(t, &syncRepo{}, repoInt)
	repo := repoInt.(*syncRepo)

	err = repo.CommitFiles(func() error {
		return ioutil.WriteFile("./testdb/some file.txt", []byte("hello world"), 0750)
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
		return ioutil.WriteFile("./testdb/some other file.txt", []byte("hello world"), 0750)
	}, "add some other file", "./testdb/some other file.txt")
	require.NoError(t, err)
	assert.Equal(t, 2, getCount())
}
