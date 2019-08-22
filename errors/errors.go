package errors

import (
	"strings"

	"github.com/pkg/errors"
)

// Errors makes it easy to combine multiple errors into a single string
type Errors []error

// ErrIf appends an error with failureMessage if the condition is true
// Returns the condition to allow for further conditional checks
func (e *Errors) ErrIf(condition bool, failureMessage string, formatArgs ...interface{}) bool {
	if condition {
		*e = append(*e, errors.Errorf(failureMessage, formatArgs...))
	}
	return condition
}

func (e *Errors) AddErr(err error) bool {
	if err != nil {
		*e = append(*e, err)
	}
	return err == nil
}

func (e Errors) ErrOrNil() error {
	if len(e) > 0 {
		return e
	}
	return nil
}

func (e Errors) Error() string {
	var buf strings.Builder
	for _, err := range e {
		buf.WriteRune('\n')
		buf.WriteString(err.Error())
	}
	return buf.String()
}
