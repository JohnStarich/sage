package sync

import (
	"io/ioutil"
	"os"
	"sync"

	"github.com/johnstarich/sage/rules"
	"github.com/pkg/errors"
)

var (
	rulesMu sync.Mutex
)

// Rules writes this rules store to the given file name
func Rules(fileName string, store *rules.Store) error {
	rulesMu.Lock()
	defer rulesMu.Unlock()
	s := store.String()
	err := ioutil.WriteFile(fileName, []byte(s), os.ModePerm)
	return errors.Wrap(err, "Error writing rules store to disk")
}
