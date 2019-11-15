package plaindb

import (
	"github.com/johnstarich/sage/vcs"
)

// DBOpt configures the DB built by Open
type DBOpt interface {
	do(*database) error
}

type dbOpt func(*database) error

func (opt dbOpt) do(db *database) error {
	return opt(db)
}

func VersionControl(setRepo *vcs.Repository) DBOpt {
	return dbOpt(func(db *database) error {
		repo, err := vcs.Open(db.path)
		db.repo = repo
		*setRepo = repo
		return err
	})
}
