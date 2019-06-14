package server

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func recovery(logger *zap.Logger, stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			panicValue := recover()
			if panicValue == nil {
				return
			}
			ce := logger.Check(zap.ErrorLevel, "[Recovery]")
			if ce == nil {
				return
			}

			fields := []zap.Field{zap.Any("error", panicValue)}
			if stack && ce.Entry.Stack == "" {
				fields = append(fields, zap.Stack("stacktrace"))
			} else if !stack {
				ce.Entry.Stack = ""
			}
			ce.Write(fields...)
		}()
		c.Next()
	}
}
