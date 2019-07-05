package sync

import (
	"io/ioutil"
	"os"
	"sync"

	"github.com/johnstarich/sage/client"
	"github.com/pkg/errors"
)

var (
	accountsMu sync.Mutex
)

// Accounts writes this account store to the given file name, passwords are included
func Accounts(fileName string, accounts *client.AccountStore) error {
	accountsMu.Lock()
	defer accountsMu.Unlock()
	b, err := accounts.MarshalWithPassword()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, b, os.ModePerm)
	return errors.Wrap(err, "Error writing account store to disk")
}
