package plaindb

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func TestAssign(t *testing.T) {
	for _, tc := range []interface{}{
		10,
		"some string",
		struct{ A string }{A: "hi"},
		&struct{ A string }{A: "hi"},
		struct {
			A *string
			B []string
		}{A: strPtr("hi"), B: []string{"hi", "there!"}},
	} {
		srcCopy := tc
		var dest interface{}
		assert.NoError(t, assign(&dest, tc))
		assert.Equal(t, tc, dest)
		assert.Equal(t, srcCopy, tc, "Source value should remain unaffected")
	}
}

func TestAssignErrors(t *testing.T) {
	for _, tc := range []struct {
		description  string
		src, dest    interface{}
		expectedDest interface{}
		expectedErr  string
	}{
		{
			description:  "happy path",
			src:          10,
			dest:         new(int),
			expectedDest: intPtr(10),
		},
		{
			description: "nil",
			src:         10,
			dest:        nil,
			expectedErr: "dest must not be nil",
		},
		{
			description: "typed nil",
			src:         10,
			dest:        (*int)(nil),
			expectedErr: "Cannot set value for *int: <nil>",
		},
		{
			description: "incompatible types",
			src:         10,
			dest:        new(string),
			expectedErr: "Type int is not assignable to *string",
		},
		{
			description: "not a pointer",
			src:         10,
			dest:        "lol not a pointer",
			expectedErr: "dest is not a pointer: string",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			t.Logf("Source: %T %+v", tc.src, tc.src)
			t.Logf("Dest:   %T %+v", tc.dest, tc.dest)
			err := assign(tc.dest, tc.src)
			if tc.expectedErr != "" {
				if assert.Error(t, err) {
					t.Logf("Error: %+v", err)
					assert.Equal(t, tc.expectedErr, err.Error())
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDest, tc.dest)
		})
	}
}

func TestBucketPutSave(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer require.NoError(t, os.RemoveAll(tmpDir))
	require.NoError(t, os.MkdirAll(tmpDir, 0755))
	path := filepath.Join(tmpDir, "accounts.json")

	b := &bucket{
		name:    "accounts",
		path:    path,
		version: "1",
		saveFn:  saveBucket,
		data: map[string]interface{}{
			"a": "some string",
			"b": 1,
		},
	}

	err = b.Put("c", true)
	require.NoError(t, err)

	data, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(`
{
    "Version": "1",
    "Data": {
        "a": "some string",
        "b": 1,
        "c": true
    }
}
`)+"\n", string(data))
}

func TestBucketGet(t *testing.T) {
	b := &bucket{
		data: map[string]interface{}{
			"a": "some string",
		},
	}

	var value string

	found, err := b.Get("b", &value)
	assert.False(t, found)
	assert.NoError(t, err)

	found, err = b.Get("a", &value)
	assert.True(t, found)
	assert.NoError(t, err)
}

func TestBucketPut(t *testing.T) {
	var b *bucket
	b = &bucket{
		data: map[string]interface{}{
			"a": "b",
			"c": 1,
		},
		saveFn: func(saveB *bucket) error {
			assert.Equal(t, b, saveB)
			return nil
		},
	}

	err := b.Put("some ID", "some value")
	require.NoError(t, err)

	var value string
	found, err := b.Get("some ID", &value)
	assert.True(t, found)
	assert.NoError(t, err)
	assert.Equal(t, "some value", value)
}

func TestBucketIter(t *testing.T) {
	m := map[string]interface{}{
		"a": "some string",
		"b": true,
		"c": struct{ C string }{C: "some C"},
	}

	b := &bucket{
		name: "accounts",
		data: m,
	}

	t.Run("all records", func(t *testing.T) {
		var value interface{}
		times := 0
		err := b.Iter(&value, func(id string) bool {
			assert.Equal(t, m[id], value)
			times++
			return true
		})
		require.NoError(t, err)
		assert.Equal(t, len(m), times)
	})

	t.Run("2 records", func(t *testing.T) {
		var value interface{}
		times := 0
		err := b.Iter(&value, func(id string) bool {
			times++
			return times < 2
		})
		require.NoError(t, err)
		assert.Equal(t, 2, times)
	})

	t.Run("invalid destination", func(t *testing.T) {
		var value bool
		err := b.Iter(&value, func(id string) bool {
			assert.True(t, value == true || value == false)
			return true
		})
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Bucket accounts: ")
		}
	})
}
