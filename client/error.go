package client

import "strings"

type credErrors []error

func (c credErrors) Error() string {
	var buf strings.Builder
	for _, e := range c {
		buf.WriteRune('\n')
		buf.WriteString(e.Error())
	}
	return buf.String()
}
