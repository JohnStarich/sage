package web

import (
	"bytes"

	"go.uber.org/zap"
)

type logWriter zap.Logger

func (l *logWriter) Write(p []byte) (n int, err error) {
	(*zap.Logger)(l).Debug(string(bytes.TrimSpace(p)))
	return len(p), nil
}
