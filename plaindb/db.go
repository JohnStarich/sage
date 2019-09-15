package plaindb

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	// MaxUpgradeAttempts is the maximum number of times a data record will attempt to be upgraded successively.
	// Used to prevent version loops. i.e. upgrading to v3 but goes v1 -> v2 -> v1 infinitely
	MaxUpgradeAttempts = 1000
)

// Upgrader upgrades data to the given version
type Upgrader interface {
	// Parse parses the original JSON data for the given version
	Parse(dataVersion, id string, data *json.RawMessage) (interface{}, error)
	// Upgrade upgrades 'data' to 'dataVersion'. May be run multiple times to incrementally upgrade the data.
	Upgrade(dataVersion, id string, data interface{}) (newVersion string, newData interface{}, err error)
}

// DB creates buckets that can read or write JSON data
type DB interface {
	// Bucket returns a bucket with 'name.json' on disk, and auto-upgraded to 'version'
	Bucket(name, version string, upgrader Upgrader) (Bucket, error)
}

type database struct {
	path string
}

// Open ...
func Open(path string) (DB, error) {
	path = filepath.Clean(path)
	err := os.MkdirAll(path, 0755)
	return &database{
		path: path,
	}, err
}

func (db *database) Bucket(name, version string, upgrader Upgrader) (Bucket, error) {
	return db.bucket(name, version, upgrader, ioutil.ReadFile, saveBucket)
}

func (db *database) bucket(
	name, version string,
	upgrader Upgrader,
	readFile func(string) ([]byte, error),
	saveFn func(*bucket) error,
) (Bucket, error) {
	if upgrader == nil {
		return nil, errors.New("Upgrader must not be nil")
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
		return nil, err
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
		for id := range data {
			currentVersion := bucketBytes.Version
			upgradeAttempts := 0
			for currentVersion != version {
				if upgradeAttempts > MaxUpgradeAttempts {
					return nil, errors.Errorf("Too many upgrade attempts to version: %q. Possibly a version upgrade loop? Current version: %q", version, currentVersion)
				}
				upgradeAttempts++

				newVersion, newValue, err := upgrader.Upgrade(currentVersion, id, data[id])
				if err != nil {
					return nil, err
				}
				if newVersion == currentVersion {
					return nil, errors.Errorf("Could not upgrade %q data from %q to %q: %+v", name, currentVersion, version, data[id])
				}
				currentVersion = newVersion
				data[id] = newValue
			}
		}
	}

	return &bucket{
		name:    name,
		path:    path,
		saveFn:  saveFn,
		version: version,
		data:    data,
	}, nil
}
