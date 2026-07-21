package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AccessLog returns middleware that logs one structured entry per request,
// tagged with the request id assigned by RequestID.
func AccessLog(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		requestID, _ := c.Get(string(HeaderRequestID))

		fields := []zap.Field{
			zap.String("request_id", toString(requestID)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
			base.Error("request completed with errors", fields...)
			return
		}

		base.Info("request completed", fields...)
	}
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}
