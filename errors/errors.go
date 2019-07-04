package errors

import "strings"

// Errors makes it easy to combine multiple errors into a single string
type Errors []error

func (e Errors) Error() string {
	var buf strings.Builder
	for _, err := range e {
		buf.WriteRune('\n')
		buf.WriteString(err.Error())
	}
	return buf.String()
}
