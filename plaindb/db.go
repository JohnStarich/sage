package plaindb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/johnstarich/sage/vcs"
	"github.com/pkg/errors"
)

const (
	// MaxUpgradeAttempts is the maximum number of times a data record will attempt to be upgraded successively.
	// Used to prevent version loops. i.e. upgrading to v3 but goes v1 -> v2 -> v1 infinitely
	MaxUpgradeAttempts = 1000
)

// Upgrader upgrades data to the given version
type Upgrader interface {
	// Parse parses the original JSON record for the given version
	Parse(dataVersion, id string, data json.RawMessage) (interface{}, error)
	// Upgrade upgrades 'data' to 'dataVersion'. May be run multiple times to incrementally upgrade the data.
	Upgrade(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error)
}

// LegacyUpgrader upgrades data from a legacy, unversioned format
type LegacyUpgrader interface {
	Upgrader
	// ParseLegacy parses the original JSON data as a whole
	// Deprecated in favor of Parse using the version format
	ParseLegacy(legacyData json.RawMessage) (version string, data map[string]json.RawMessage, err error)
}

// BucketUpgrader is only used for schema-level upgrades to a bucket, e.g. updating the ID format. Only runs once, then bucket items are processed.
// For a simpler upgrade implementation, stick to just Upgrader.
type BucketUpgrader interface {
	// UpgradeAll attempts to upgrade all of data once, prior to individual item upgrades
	UpgradeAll(dataVersion string, data map[string]interface{}) (newVersion string, newData map[string]interface{}, err error)
}

// DB creates buckets that can read or write JSON data
type DB interface {
	io.Closer
	// Bucket returns a bucket with 'name.json' on disk, and auto-upgraded to 'version'
	Bucket(name, version string, upgrader Upgrader) (Bucket, error)
}

type database struct {
	path    string
	repo    vcs.Repository
	buckets map[string]*bucket
}

type bucketSaver func(*bucket) error

// Open prepares and creates a DB for the given file path
func Open(path string, opts ...DBOpt) (DB, error) {
	path = filepath.Clean(path)
	if err := os.MkdirAll(path, 0750); err != nil {
		return nil, err
	}
	db := &database{
		path:    path,
		buckets: make(map[string]*bucket),
	}
	for _, opt := range opts {
		if err := opt.do(db); err != nil {
			return nil, err
		}
	}
	return db, nil
}

func (db *database) Bucket(name, version string, upgrader Upgrader) (Bucket, error) {
	saver := saveBucketToDisk
	if db.repo != nil {
		saver = repoSaveBucket(db.repo)
	}
	return db.bucket(name, version, upgrader, ioutil.ReadFile, saver)
}

func (db *database) bucket(
	name, version string,
	upgrader Upgrader,
	readFile func(string) ([]byte, error),
	saver bucketSaver,
) (Bucket, error) {
	if upgrader == nil {
		return nil, errors.New("Upgrader must not be nil")
	}
	if b, exists := db.buckets[name]; exists {
		return b, nil
	}

	path := filepath.Join(db.path, name+".json")
	dataBytes, err := readFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		dataBytes = []byte(`{}`)
	}

	var bucketBytes unmarshalBucket
	if err := json.Unmarshal(dataBytes, &bucketBytes); err != nil {
		legacyUp, ok := upgrader.(LegacyUpgrader)
		if !ok {
			return nil, err
		}
		// try a legacy format too
		version, data, err := legacyUp.ParseLegacy(dataBytes)
		if err != nil {
			return nil, errors.Wrap(err, "Parse legacy format")
		}
		bucketBytes.Version = version
		bucketBytes.Data = data
	}

	data := make(map[string]interface{}, len(bucketBytes.Data))
	for id, bytes := range bucketBytes.Data {
		var err error
		data[id], err = upgrader.Parse(bucketBytes.Version, id, bytes)
		if err != nil {
			return nil, err
		}
	}

	if bucketBytes.Version != version {
		if bucketUpgrader, ok := upgrader.(BucketUpgrader); ok {
			// Attempt an upgrade for the whole bucket first, i.e. ID format changes.
			// Runs only once, since hard to guarantee when it would re-run.
			// Definitely shouldn't re-run during individual item upgrades.
			var err error
			bucketBytes.Version, data, err = bucketUpgrader.UpgradeAll(bucketBytes.Version, data)
			if err != nil {
				return nil, err
			}
		}
	}
	if bucketBytes.Version != version {
		for id := range data {
			upgradedItem, err := upgradeItem(bucketBytes.Version, version, name, upgrader, id, data[id])
			if err != nil {
				return nil, err
			}
			data[id] = upgradedItem
		}
	}

	b := &bucket{
		name:    name,
		path:    path,
		saver:   saver,
		version: version,
		data:    data,
	}

	db.buckets[name] = b
	return b, nil
}

func upgradeItem(currentVersion, finalVersion, name string, upgrader Upgrader, id string, item interface{}) (interface{}, error) {
	upgradeAttempts := 0
	for currentVersion != finalVersion {
		if upgradeAttempts > MaxUpgradeAttempts {
			return nil, errors.Errorf("Too many upgrade attempts to version: %q. Possibly a version upgrade loop? Current version: %q", finalVersion, currentVersion)
		}
		upgradeAttempts++

		newVersion, newItem, err := upgrader.Upgrade(currentVersion, id, item)
		if err != nil {
			return nil, err
		}
		if newVersion == currentVersion {
			return nil, errors.Errorf("Could not upgrade %q data from %q to %q: %+v", name, currentVersion, finalVersion, item)
		}
		currentVersion = newVersion
		item = newItem
	}
	return item, nil
}

// Close locks all buckets to prepare for safe shutdown. Use after close has been called is not defined.
func (db *database) Close() error {
	return db.close(func(b *bucket) {
		b.mu.Lock()
	})
}

func (db *database) close(locker func(b *bucket)) error {
	if db == nil {
		return nil
	}
	for _, b := range db.buckets {
		locker(b)
	}
	return nil
}

func repoSaveBucket(repo vcs.Repository) bucketSaver {
	return func(b *bucket) error {
		saveBucket := func() error {
			return saveBucketToDisk(b)
		}
		return repo.CommitFiles(saveBucket, "Update "+b.name, b.path)
	}
}

// MockDB is a DB with additional mocking utilities
type MockDB interface {
	DB
	Dump(Bucket) string
}

type mockDatabase struct {
	database
	MockConfig
}

// MockConfig contains stubs for a full MockDB
type MockConfig struct {
	FileReader func(path string) ([]byte, error)
	Saver      func(Bucket) error
}

// NewMockDB creates a new DB without a backing file store, to be used in tests
func NewMockDB(conf MockConfig) MockDB {
	if conf.FileReader == nil {
		conf.FileReader = func(string) ([]byte, error) { return nil, nil }
	}
	if conf.Saver == nil {
		conf.Saver = func(Bucket) error { return nil }
	}
	return &mockDatabase{
		database: database{
			path:    "mock",
			buckets: map[string]*bucket{},
		},
		MockConfig: conf,
	}
}

func (db *mockDatabase) Bucket(name, version string, upgrader Upgrader) (Bucket, error) {
	return db.bucket(name, version, upgrader, db.FileReader, func(b *bucket) error { return db.Saver(b) })
}

func (db *mockDatabase) Dump(b Bucket) string {
	bucketStruct, ok := b.(*bucket)
	if !ok {
		panic(fmt.Sprintf("Invalid bucket struct for MockDB.Dump: %T", b))
	}
	if filepath.Dir(bucketStruct.path) != db.path {
		panic("Invalid bucket for MockDB.Dump: Bucket was not created by MockDB")
	}
	var buf bytes.Buffer
	if err := encodeBucket(&buf, bucketStruct); err != nil {
		panic(err)
	}
	return buf.String()
}
