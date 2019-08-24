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

// AddErr appends an error if it is not nil. Smartly combines errors of type Errors
func (e *Errors) AddErr(err error) bool {
	if err != nil {
		if errs, ok := err.(Errors); ok {
			*e = append(*e, errs...)
		} else {
			*e = append(*e, err)
		}
	}
	return err == nil
}

// ErrOrNil returns e if an error is present, otherwise returns nil
func (e Errors) ErrOrNil() error {
	if len(e) > 0 {
		return e
	}
	return nil
}

func (e Errors) Error() string {
	var buf strings.Builder
	for i, err := range e {
		if i != 0 {
			buf.WriteRune('\n')
		}
		buf.WriteString(err.Error())
	}
	return buf.String()
}
