package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery returns middleware that recovers from panics in downstream
// handlers, logs the panic with a stack trace, and responds with a generic
// 500 instead of crashing the process or leaking internal details.
func Recovery(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				requestID, _ := c.Get(string(HeaderRequestID))

				base.Error("panic recovered",
					zap.Any("panic", r),
					zap.String("request_id", toString(requestID)),
					zap.String("path", c.Request.URL.Path),
					zap.Stack("stack"),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()
		c.Next()
	}
}
