package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestRequestID_GeneratesWhenAbsent(t *testing.T) {
	r := newTestRouter()
	r.Use(RequestID())

	var capturedCtxID string
	r.GET("/x", func(c *gin.Context) {
		capturedCtxID = RequestIDFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	respID := rec.Header().Get(HeaderRequestID)
	require.NotEmpty(t, respID)
	assert.Equal(t, respID, capturedCtxID)
}

func TestRequestID_ReusesInboundHeader(t *testing.T) {
	r := newTestRouter()
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderRequestID, "fixed-id-123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, "fixed-id-123", rec.Header().Get(HeaderRequestID))
}

func TestRequestIDFromContext_EmptyWhenAbsent(t *testing.T) {
	assert.Equal(t, "", RequestIDFromContext(context.Background()))
}

func TestAccessLog_LogsRequestCompletion(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	base := zap.New(core)

	r := newTestRouter()
	r.Use(RequestID(), AccessLog(base))
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, 1, logs.Len())
	entry := logs.All()[0]
	assert.Equal(t, "request completed", entry.Message)
	assert.Equal(t, "/ok", entry.ContextMap()["path"])
}

func TestRecovery_RecoversPanicAndReturns500(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	base := zap.New(core)

	r := newTestRouter()
	r.Use(RequestID(), Recovery(base))
	r.GET("/boom", func(c *gin.Context) { panic("kaboom") })

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, rec.Body.String())
	require.Equal(t, 1, logs.Len())
	assert.Equal(t, "panic recovered", logs.All()[0].Message)
}
