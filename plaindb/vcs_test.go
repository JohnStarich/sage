package plaindb

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func TestNewDBDir(t *testing.T) {
	err := os.RemoveAll("./testdb")
	require.NoError(t, err)
	err = os.MkdirAll("./testdb", 0755)
	require.NoError(t, err)
	err = ioutil.WriteFile("./testdb/bucket.json", []byte(`{}`), 0755)
	require.NoError(t, err)

	repo, err := newSyncRepo("./testdb")
	require.NoError(t, err)
	assert.NotNil(t, repo)
}

func TestSaveBucket(t *testing.T) {
	cleanup := func() {
		err := os.RemoveAll("./testdb")
		require.NoError(t, err)
	}
	cleanup()
	defer cleanup()
	err := os.MkdirAll("./testdb", 0755)
	require.NoError(t, err)

	repo, err := newSyncRepo("./testdb")
	require.NoError(t, err)

	b := &bucket{
		name:  "bucket",
		path:  "./testdb/bucket.json",
		data:  make(map[string]interface{}),
		saver: repo.SaveBucket,
	}
	err = b.Put("some ID", "hello world")
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

	err = b.Put("some other ID", "hello there")
	require.NoError(t, err)
	assert.Equal(t, 2, getCount())
}
