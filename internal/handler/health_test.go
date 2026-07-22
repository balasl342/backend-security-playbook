package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubChecker struct {
	name string
	err  error
}

func (s stubChecker) Name() string                    { return s.name }
func (s stubChecker) Check(ctx context.Context) error { return s.err }

func newRouter(h *HealthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.Register(r)
	return r
}

func TestHealthHandler_Liveness_AlwaysOK(t *testing.T) {
	h := NewHealthHandler(time.Second, stubChecker{name: "postgres", err: errors.New("down")})
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}

func TestHealthHandler_Readiness_AllHealthy(t *testing.T) {
	h := NewHealthHandler(time.Second,
		stubChecker{name: "postgres"},
		stubChecker{name: "redis"},
	)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{
		"status": "ok",
		"checks": {
			"postgres": {"status": "up"},
			"redis": {"status": "up"}
		}
	}`, rec.Body.String())
}

func TestHealthHandler_Readiness_OneUnhealthy(t *testing.T) {
	h := NewHealthHandler(time.Second,
		stubChecker{name: "postgres"},
		stubChecker{name: "redis", err: errors.New("connection refused")},
	)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.JSONEq(t, `{
		"status": "unavailable",
		"checks": {
			"postgres": {"status": "up"},
			"redis": {"status": "down", "error": "connection refused"}
		}
	}`, rec.Body.String())
}

func TestHealthHandler_Readiness_NoCheckers(t *testing.T) {
	h := NewHealthHandler(time.Second)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"ok","checks":{}}`, rec.Body.String())
}
