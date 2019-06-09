package ledger

import "fmt"

type Error struct {
	firstFailedTxnIndex int
	cause               error
}

func NewValidateError(firstFailure int, cause error) error {
	if cause == nil {
		return nil
	}
	return Error{firstFailure, cause}
}

func (e Error) Error() string {
	return fmt.Sprintf("Failed to validate ledger at transaction index #%d: %s", e.firstFailedTxnIndex, e.cause)
}
