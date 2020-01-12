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
	parser   func(dataVersion, id string, data json.RawMessage) (interface{}, error)
	upgrader func(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error)
}

func (m *mockUpgrader) Parse(dataVersion, id string, data json.RawMessage) (interface{}, error) {
	return m.parser(dataVersion, id, data)
}

func (m *mockUpgrader) Upgrade(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
	return m.upgrader(dataVersion, id, data)
}

type mockLegacyUpgrader struct {
	mockUpgrader
	legacyParser func(legacyData json.RawMessage) (version string, data map[string]json.RawMessage, err error)
}

func (m *mockLegacyUpgrader) ParseLegacy(legacyData json.RawMessage) (version string, data map[string]json.RawMessage, err error) {
	return m.legacyParser(legacyData)
}

type mockBucketUpgrader struct {
	mockUpgrader
	bucketUpgrader func(dataVersion string, data map[string]interface{}) (newVersion string, newData map[string]interface{}, err error)
}

func (m *mockBucketUpgrader) UpgradeAll(dataVersion string, data map[string]interface{}) (newVersion string, newData map[string]interface{}, err error) {
	return m.bucketUpgrader(dataVersion, data)
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
		require.IsType(t, &database{}, db)
		db.(*database).repo = nil // nil out for comparison
		assert.Equal(t, &database{
			path:    tmpDir,
			buckets: map[string]*bucket{},
		}, db)
	}

	// Bucket
	b, err := db.Bucket("accounts", "1", &mockUpgrader{})
	assert.NoError(t, err)
	b.(*bucket).saver = nil // can't compare functions
	assert.Equal(t, &bucket{
		name:    "accounts",
		path:    filepath.Join(tmpDir, "accounts.json"),
		saver:   nil,
		version: "1",
		data:    map[string]interface{}{},
	}, b)
}

func TestClose(t *testing.T) {
	db := NewMockDB(MockConfig{
		FileReader: func(path string) ([]byte, error) {
			return []byte(`{}`), nil
		},
	})
	_, err := db.Bucket("something", "", &mockUpgrader{})
	require.NoError(t, err)
	_, err = db.Bucket("some other thing", "", &mockUpgrader{})
	require.NoError(t, err)

	closed := 0
	db.(*mockDatabase).close(func(b *bucket) {
		closed++
	})
	assert.Equal(t, 2, closed)

	assert.NoError(t, db.Close())
	assert.NoError(t, (*database)(nil).Close())
}

func intParser(dataVersion, id string, data json.RawMessage) (interface{}, error) {
	i, err := strconv.ParseInt(string(data), 10, 64)
	return int(i), err
}

func stringParser(dataVersion, id string, data json.RawMessage) (interface{}, error) {
	s, err := strconv.Unquote(string(data))
	return s, err
}

func intUpgrader(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
	i, _ := strconv.ParseInt(dataVersion, 10, 64)
	return strconv.FormatInt(i+1, 10), data.(int) + 1, nil
}

func stringUpgrader(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
	i, _ := strconv.ParseInt(dataVersion, 10, 64)
	return strconv.FormatInt(i+1, 10), data.(string) + "*", nil
}

func failUpgrader(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
	return "", nil, errors.New("some failure")
}

func loopUpgrader(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
	i, _ := strconv.ParseInt(dataVersion, 10, 64)
	return strconv.FormatInt(-i, 10), data.(int) + 1, nil
}

func staleUpgrader(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error) {
	return dataVersion, data, nil
}

func TestBucket(t *testing.T) {
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
			upgrader:    &mockUpgrader{parser: intParser, upgrader: intUpgrader},
			readErr:     os.ErrNotExist,
		},
		{
			description: "bad read",
			upgrader:    &mockUpgrader{parser: intParser, upgrader: intUpgrader},
			readErr:     someReadErr,
			expectedErr: "some read error",
		},
		{
			description: "empty bucket file",
			name:        "accounts",
			version:     "1",
			upgrader:    &mockUpgrader{parser: intParser, upgrader: intUpgrader},
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
			upgrader:    &mockUpgrader{parser: intParser, upgrader: intUpgrader},
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
			upgrader:    &mockUpgrader{parser: intParser},
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
			upgrader:    &mockUpgrader{parser: intParser, upgrader: failUpgrader},
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
			upgrader:    &mockUpgrader{parser: intParser, upgrader: intUpgrader},
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
			upgrader:    &mockUpgrader{parser: intParser, upgrader: loopUpgrader},
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
			upgrader:    &mockUpgrader{parser: intParser, upgrader: staleUpgrader},
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

			var expectedBucketPath string
			saved := false
			db := NewMockDB(MockConfig{
				FileReader: func(path string) ([]byte, error) {
					assert.Equal(t, expectedBucketPath, path)
					return []byte(tc.bucketData), tc.readErr
				},
				Saver: func(b Bucket) error {
					saved = true
					return nil
				},
			})
			expectedBucketPath = filepath.Join(db.(*mockDatabase).path, tc.name+".json")

			b, err := db.Bucket(tc.name, tc.version, tc.upgrader)
			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
				return
			}

			require.NoError(t, err)
			_ = b.(*bucket).saver(nil) // run func, since can't compare func values
			assert.True(t, saved)

			b.(*bucket).saver = nil // can't compare functions
			assert.Equal(t, &bucket{
				name:  tc.name,
				path:  expectedBucketPath,
				saver: nil,

				version: tc.version,
				data:    tc.expectedData,
			}, b)
		})
	}
}

func TestLegacyParse(t *testing.T) {
	saved := false
	db := NewMockDB(MockConfig{
		FileReader: func(path string) ([]byte, error) {
			return []byte(`["first", "second", "third"]`), nil
		},
		Saver: func(b Bucket) error {
			saved = true
			return nil
		},
	})

	indexParser := func(legacyData json.RawMessage) (version string, data map[string]json.RawMessage, err error) {
		var arr []json.RawMessage
		if err := json.Unmarshal(legacyData, &arr); err != nil {
			return "", nil, err
		}
		data = make(map[string]json.RawMessage, len(arr))
		for i, item := range arr {
			data[strconv.FormatInt(int64(i), 10)] = item
		}
		version = "0"
		return
	}

	upgrader := &mockLegacyUpgrader{
		mockUpgrader: mockUpgrader{
			parser:   stringParser,
			upgrader: stringUpgrader,
		},
		legacyParser: indexParser,
	}
	b, err := db.Bucket("accounts", "2", upgrader)
	require.NoError(t, err)
	_ = b.(*bucket).saver(nil) // run func, since can't compare func values
	assert.True(t, saved)

	b.(*bucket).saver = nil // can't compare functions
	assert.Equal(t, &bucket{
		name:  "accounts",
		path:  "mock/accounts.json",
		saver: nil,

		version: "2",
		data: map[string]interface{}{
			"0": "first**",
			"1": "second**",
			"2": "third**",
		},
	}, b)
}

func TestNewMockDB(t *testing.T) {
	// builtin FileReader
	db := NewMockDB(MockConfig{})
	_, err := db.Bucket("something", "", &mockUpgrader{})
	assert.Error(t, err)

	// builtin Saver
	db = NewMockDB(MockConfig{
		FileReader: func(path string) ([]byte, error) {
			return []byte(`{}`), nil
		},
	})
	b, err := db.Bucket("something", "1", &mockUpgrader{})
	require.NoError(t, err)
	assert.NoError(t, b.Put("hi", "hi"))
}

func TestMockDBDump(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		db := NewMockDB(MockConfig{
			FileReader: func(path string) ([]byte, error) {
				return []byte(`{}`), nil
			},
		})
		b, err := db.Bucket("something", "1", &mockUpgrader{})
		require.NoError(t, err)
		assert.Equal(t, `{
    "Version": "1",
    "Data": {}
}
`, db.Dump(b))
	})

	t.Run("not a *bucket", func(t *testing.T) {
		type mockBucket struct {
			bucket
		}
		db := NewMockDB(MockConfig{})
		assert.Panics(t, func() {
			db.Dump(&mockBucket{})
		})
	})

	t.Run("not a mock bucket", func(t *testing.T) {
		db := NewMockDB(MockConfig{})
		assert.Panics(t, func() {
			db.Dump(&bucket{})
		})
	})

	t.Run("fail to encode", func(t *testing.T) {
		db := NewMockDB(MockConfig{
			FileReader: func(path string) ([]byte, error) {
				return []byte(`{}`), nil
			},
		})
		b, err := db.Bucket("something", "1", &mockUpgrader{})
		require.NoError(t, err)
		b.(*bucket).data["hi"] = json.RawMessage(`garbage`)
		assert.Panics(t, func() {
			db.Dump(b)
		})
	})
}

func TestBucketUpgrader(t *testing.T) {
	db := NewMockDB(MockConfig{
		FileReader: func(string) ([]byte, error) {
			return []byte(`{
				"Version": "1",
				"Data": {
					"old": 1
				}
			}`), nil
		},
	})
	mock := mockBucketUpgrader{
		mockUpgrader: mockUpgrader{
			parser:   intParser,
			upgrader: intUpgrader,
		},
		bucketUpgrader: func(dataVersion string, data map[string]interface{}) (string, map[string]interface{}, error) {
			return "", nil, errors.New("Unimplemented")
		},
	}
	_, err := db.Bucket("budgets", "2", &mock)
	require.Error(t, err)
	assert.Equal(t, "Unimplemented", err.Error())

	mock.bucketUpgrader = func(dataVersion string, data map[string]interface{}) (newVersion string, newData map[string]interface{}, err error) {
		newData = make(map[string]interface{}, len(data))
		for oldID, value := range data {
			newData["newIDFormat-"+oldID] = value
		}
		return "2", newData, nil
	}
	bucket, err := db.Bucket("budgets", "2", &mock)
	require.NoError(t, err)
	var num int
	found, err := bucket.Get("newIDFormat-old", &num)
	assert.True(t, found)
	assert.NoError(t, err)
	assert.Equal(t, 1, num)
}
