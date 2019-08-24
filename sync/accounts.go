package sync

import (
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
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	writeErr := accounts.WriteTo(file)
	closeErr := file.Close()
	if writeErr != nil {
		return errors.Wrap(err, "Error writing account store to disk")
	}
	return errors.Wrap(closeErr, "Error closing accounts file")
}
