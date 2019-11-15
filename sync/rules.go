package sync

import (
	"github.com/johnstarich/sage/rules"
	"github.com/johnstarich/sage/vcs"
	"github.com/pkg/errors"
)

// Rules writes this rules store to the given file name
func Rules(rulesFile vcs.File, store *rules.Store) error {
	s := store.String()
	err := rulesFile.Write([]byte(s))
	return errors.Wrap(err, "Error writing rules store to disk")
}
