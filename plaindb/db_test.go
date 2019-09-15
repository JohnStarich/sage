package plaindb

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUpgrader struct {
	parseFn   func(dataVersion, id string, data *json.RawMessage) (interface{}, error)
	upgradeFn func(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error)
}

func (m *mockUpgrader) Parse(dataVersion, id string, data *json.RawMessage) (interface{}, error) {
	return m.parseFn(dataVersion, id, data)
}

func (m *mockUpgrader) Upgrade(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
	return m.upgradeFn(dataVersion, id, data)
}

func TestOpenNewBucket(t *testing.T) {
	var db DB
	var tmpDir string
	{
		// Open
		var err error
		tmpDir, err = ioutil.TempDir("", "")
		require.NoError(t, err)
		defer require.NoError(t, os.RemoveAll(tmpDir))
		_, err = ioutil.ReadDir(tmpDir)
		require.True(t, os.IsNotExist(err))

		db, err = Open(tmpDir)
		require.NoError(t, err)
		assert.DirExists(t, tmpDir)
		assert.Equal(t, &database{path: tmpDir}, db)
	}

	// Bucket
	b, err := db.Bucket("accounts", "1", &mockUpgrader{})
	assert.NoError(t, err)
	b.(*bucket).saveFn = nil // can't compare functions
	assert.Equal(t, &bucket{
		name:    "accounts",
		path:    filepath.Join(tmpDir, "accounts.json"),
		saveFn:  nil,
		version: "1",
		data:    map[string]interface{}{},
	}, b)
}

func TestBucket(t *testing.T) {
	intParser := func(dataVersion, id string, data *json.RawMessage) (interface{}, error) {
		if data == nil {
			return nil, nil
		}
		i, err := strconv.ParseInt(string(*data), 10, 64)
		return int(i), err
	}
	intUpgrader := func(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
		i, _ := strconv.ParseInt(dataVersion, 10, 64)
		return strconv.FormatInt(i+1, 10), data.(int) + 1, nil
	}
	failUpgrader := func(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
		return "", nil, errors.New("some failure")
	}
	loopUpgrader := func(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
		i, _ := strconv.ParseInt(dataVersion, 10, 64)
		return strconv.FormatInt(-i, 10), data.(int) + 1, nil
	}
	staleUpgrader := func(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
		return dataVersion, data, nil
	}
	someReadErr := errors.New("some read error")

	for _, tc := range []struct {
		description   string
		name, version string
		upgrader      Upgrader
		bucketData    string
		readErr       error

		expectedData map[string]interface{}
		expectedErr  string
	}{
		{
			description: "new bucket",
			name:        "accounts",
			version:     "1",
			upgrader:    &mockUpgrader{parseFn: intParser, upgradeFn: intUpgrader},
			readErr:     os.ErrNotExist,
		},
		{
			description: "bad read",
			upgrader:    &mockUpgrader{parseFn: intParser, upgradeFn: intUpgrader},
			readErr:     someReadErr,
			expectedErr: "some read error",
		},
		{
			description: "empty bucket file",
			name:        "accounts",
			version:     "1",
			upgrader:    &mockUpgrader{parseFn: intParser, upgradeFn: intUpgrader},
			bucketData:  "",
			expectedErr: "unexpected end of JSON input",
		},
		{
			description: "nil upgrader error",
			upgrader:    nil,
			expectedErr: "Upgrader must not be nil",
		},
		{
			description: "upgrade once",
			name:        "accounts",
			version:     "2",
			upgrader:    &mockUpgrader{parseFn: intParser, upgradeFn: intUpgrader},
			bucketData: `
			{
				"Version": "1",
				"Data": {
					"a": 1,
					"b": 2
				}
			}`,
			expectedData: map[string]interface{}{
				"a": 2,
				"b": 3,
			},
		},
		{
			description: "parse failure",
			name:        "accounts",
			version:     "2",
			upgrader:    &mockUpgrader{parseFn: intParser},
			bucketData: `
			{
				"Version": "1",
				"Data": {
					"a": "not an int",
					"b": 2
				}
			}`,
			expectedErr: "strconv.ParseInt",
		},
		{
			description: "upgrade once failure",
			name:        "accounts",
			version:     "2",
			upgrader:    &mockUpgrader{parseFn: intParser, upgradeFn: failUpgrader},
			bucketData: `
			{
				"Version": "1",
				"Data": {
					"a": 1,
					"b": 2
				}
			}`,
			expectedErr: "some failure",
		},
		{
			description: "upgrade twice",
			name:        "accounts",
			version:     "3",
			upgrader:    &mockUpgrader{parseFn: intParser, upgradeFn: intUpgrader},
			bucketData: `
			{
				"Version": "1",
				"Data": {
					"a": 1,
					"b": 2
				}
			}`,
			expectedData: map[string]interface{}{
				"a": 3,
				"b": 4,
			},
		},
		{
			description: "upgrade loop",
			name:        "accounts",
			version:     "2",
			upgrader:    &mockUpgrader{parseFn: intParser, upgradeFn: loopUpgrader},
			bucketData: `
			{
				"Version": "1",
				"Data": {
					"a": 1,
					"b": 2
				}
			}`,
			expectedErr: "Too many upgrade attempts",
		},
		{
			description: "upgrade to same version",
			name:        "accounts",
			version:     "2",
			upgrader:    &mockUpgrader{parseFn: intParser, upgradeFn: staleUpgrader},
			bucketData: `
			{
				"Version": "1",
				"Data": {
					"a": 1,
					"b": 2
				}
			}`,
			expectedErr: `Could not upgrade "accounts" data from "1" to "2"`,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			if tc.expectedData == nil {
				tc.expectedData = make(map[string]interface{})
			}

			db := &database{path: "some path"}
			expectedBucketPath := filepath.Join(db.path, tc.name+".json")
			readFile := func(path string) ([]byte, error) {
				assert.Equal(t, expectedBucketPath, path)
				return []byte(tc.bucketData), tc.readErr
			}
			saved := false
			saveFn := func(b *bucket) error {
				saved = true
				return nil
			}

			b, err := db.bucket(tc.name, tc.version, tc.upgrader, readFile, saveFn)
			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
				return
			}

			require.NoError(t, err)
			_ = b.(*bucket).saveFn(nil) // run func, since can't compare func values
			assert.True(t, saved)

			b.(*bucket).saveFn = nil // can't compare functions
			assert.Equal(t, &bucket{
				name:   tc.name,
				path:   expectedBucketPath,
				saveFn: nil,

				version: tc.version,
				data:    tc.expectedData,
			}, b)
		})
	}
}
