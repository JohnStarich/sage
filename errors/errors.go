package errors

import (
	"encoding/json"
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
	if len(e) == 1 {
		// simplify result if there's only one error
		return e[0]
	}
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

func (e Errors) MarshalJSON() ([]byte, error) {
	var errs []interface{}
	for _, err := range e {
		switch err := err.(type) {
		case json.Marshaler:
			errs = append(errs, err)
		default:
			errs = append(errs, map[string]interface{}{"Description": err.Error()})
		}
	}
	return json.Marshal(errs)
}
