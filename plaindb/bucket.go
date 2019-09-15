package plaindb

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

// Bucket reads and writes records on a DB
type Bucket interface {
	// Iter iterates over all values, assigning each value to 'v', then calling fn with it's ID
	Iter(v interface{}, fn func(id string) (keepGoing bool)) error
	// Get reads the record with key 'id' into 'v'
	Get(id string, v interface{}) (found bool, err error)
	// Put writes the record 'v' with key 'id'
	Put(id string, v interface{}) error
}

type bucket struct {
	name   string
	path   string
	mu     sync.RWMutex
	saveFn func(*bucket) error

	version string
	data    map[string]interface{}
}

type unmarshalBucket struct {
	Version string
	Data    map[string]*json.RawMessage
}

type marshalBucket struct {
	Version string
	Data    map[string]interface{}
}

func (b *bucket) Iter(v interface{}, fn func(id string) (keepGoing bool)) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for id, value := range b.data {
		if err := assign(v, value); err != nil {
			return b.wrapErr(err)
		}
		if !fn(id) {
			return nil
		}
	}
	return nil
}

func (b *bucket) Get(id string, v interface{}) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	value, found := b.data[id]
	if !found {
		return false, nil
	}
	return found, b.wrapErr(assign(v, value))
}

func (b *bucket) Put(id string, v interface{}) error {
	b.mu.Lock()
	b.data[id] = v
	b.mu.Unlock()
	return b.saveFn(b)
}

func (b *bucket) wrapErr(err error) error {
	return errors.Wrap(err, "Bucket "+b.name)
}

func saveBucket(b *bucket) (returnErr error) {
	dir := filepath.Dir(b.path)
	file, err := ioutil.TempFile(dir, filepath.Base(b.path)+".*.tmp")
	if err != nil {
		return b.wrapErr(err)
	}
	defer func() {
		closeErr := file.Close()
		rmErr := os.Remove(file.Name()) // clean up tmp file, if it wasn't renamed
		if returnErr == nil {
			if rmErr != nil && !os.IsNotExist(rmErr) {
				returnErr = b.wrapErr(rmErr)
			}
			if closeErr != nil {
				returnErr = b.wrapErr(closeErr)
			}
		}
	}()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "    ")
	enc.SetEscapeHTML(false)

	b.mu.RLock()
	err = enc.Encode(marshalBucket{
		Version: b.version,
		Data:    b.data,
	})
	b.mu.RUnlock()
	if err != nil {
		return b.wrapErr(err)
	}

	return b.wrapErr(os.Rename(file.Name(), b.path))
}

// assign sets dest's pointer value to source
func assign(dest interface{}, source interface{}) (err error) {
	if dest == nil {
		return errors.New("dest must not be nil")
	}
	defer func() {
		// reflection can panic if not used perfectly. recover and wrap the error until stable
		if v := recover(); v != nil && err == nil {
			err = errors.Errorf("Reflect error during assignment: %+v", v)
		}
	}()

	destValue := reflect.ValueOf(dest)
	destType := destValue.Type()
	if destType.Kind() != reflect.Ptr {
		return errors.Errorf("dest is not a pointer: %T", dest)
	}
	// dereference pointer value and type for assignment
	destValue = destValue.Elem()
	if !destValue.CanSet() {
		return errors.Errorf("Cannot set value for %T: %+v", dest, dest)
	}
	destType = destValue.Type()

	sourceValue := reflect.ValueOf(source)
	if !sourceValue.Type().AssignableTo(destType) {
		return errors.Errorf("Type %T is not assignable to %T", source, dest)
	}
	destValue.Set(sourceValue)
	return nil
}
