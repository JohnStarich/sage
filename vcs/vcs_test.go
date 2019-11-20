package vcs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

const testDBPath = "./testdb"

func cleanupTestDB(t *testing.T) {
	t.Helper()
	require.NoError(t, os.RemoveAll(testDBPath))
}

func TestOpen(t *testing.T) {
	cleanupTestDB(t)
	defer cleanupTestDB(t)
	err := os.MkdirAll(testDBPath, 0750)
	require.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(testDBPath, "bucket.json"), []byte(`{}`), 0750)
	require.NoError(t, err)

	repoInt, err := Open(testDBPath)
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

func TestOpenMkdirErr(t *testing.T) {
	cleanupTestDB(t)
	defer cleanupTestDB(t)

	err := ioutil.WriteFile(testDBPath, []byte(`I'm not a database!`), 0750)
	require.NoError(t, err)
	_, err = Open(testDBPath)
	require.Error(t, err)
	assert.Equal(t, "mkdir testdb: not a directory", err.Error())
}

func TestCommitFiles(t *testing.T) {
	cleanupTestDB(t)
	defer cleanupTestDB(t)

	repoInt, err := Open(testDBPath)
	require.NoError(t, err)
	require.IsType(t, &syncRepo{}, repoInt)
	repo := repoInt.(*syncRepo)

	// committing nothing fails
	err = repo.CommitFiles(nil, "")
	require.Error(t, err)
	assert.Equal(t, "No files to commit", err.Error())

	// modify and commit files a few times
	err = repo.CommitFiles(func() error {
		return ioutil.WriteFile(filepath.Join(testDBPath, "some file.txt"), []byte("hello world"), 0750)
	}, "add some file", filepath.Join(testDBPath, "some file.txt"))
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
		return ioutil.WriteFile(filepath.Join(testDBPath, "some other file.txt"), []byte("hello world"), 0750)
	}, "add some other file", filepath.Join(testDBPath, "some other file.txt"))
	require.NoError(t, err)
	assert.Equal(t, 2, getCount())
}

func TestCommitNoChanges(t *testing.T) {
	cleanupTestDB(t)
	defer cleanupTestDB(t)

	repoInt, err := Open(testDBPath)
	require.NoError(t, err)
	require.IsType(t, &syncRepo{}, repoInt)
	repo := repoInt.(*syncRepo)

	err = repo.CommitFiles(func() error {
		return ioutil.WriteFile(filepath.Join(testDBPath, "some file.txt"), []byte("hello world"), 0750)
	}, "add some file", filepath.Join(testDBPath, "some file.txt"))
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

	err = repo.CommitFiles(func() error { return nil }, "add same file", filepath.Join(testDBPath, "some file.txt"))
	require.NoError(t, err)
	assert.Equal(t, 1, getCount(), "no commit should be made for unchanged file")
}
