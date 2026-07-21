// Package middleware contains cross-cutting Gin middleware: request id
// injection, access logging, panic recovery, and (in later commits) field
// level encryption/decryption.
package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HeaderRequestID is the HTTP header used to propagate a request id across
// service boundaries.
const HeaderRequestID = "X-Request-ID"

type requestIDCtxKey struct{}

// RequestID returns middleware that assigns each request a unique id,
// reusing an inbound X-Request-ID header when the caller already provided
// one (e.g. from an upstream gateway). The id is stored on the Gin context,
// the standard context.Context, and echoed back in the response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = uuid.NewString()
		}

		c.Set(string(HeaderRequestID), id)
		ctx := context.WithValue(c.Request.Context(), requestIDCtxKey{}, id)
		c.Request = c.Request.WithContext(ctx)

		c.Writer.Header().Set(HeaderRequestID, id)
		c.Next()
	}
}

// RequestIDFromContext extracts the request id set by RequestID, returning
// "" if none is present (e.g. outside of an HTTP request).
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDCtxKey{}).(string)
	return id
}
