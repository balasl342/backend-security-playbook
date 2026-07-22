// Package handler contains HTTP handlers (Gin) that translate requests into
// service calls and shape responses.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Checker reports whether a dependency (database, cache, ...) is healthy.
// Implementations should respect ctx's deadline and return promptly.
type Checker interface {
	// Name identifies the dependency in the readiness response, e.g. "postgres".
	Name() string
	Check(ctx context.Context) error
}

// HealthHandler serves liveness and readiness endpoints.
type HealthHandler struct {
	checkers []Checker
	timeout  time.Duration
}

// NewHealthHandler builds a HealthHandler that evaluates the given checkers
// on each readiness probe, bounding each check by timeout.
func NewHealthHandler(timeout time.Duration, checkers ...Checker) *HealthHandler {
	return &HealthHandler{checkers: checkers, timeout: timeout}
}

// Register mounts /healthz and /readyz on the given router.
func (h *HealthHandler) Register(r gin.IRouter) {
	r.GET("/healthz", h.Liveness)
	r.GET("/readyz", h.Readiness)
}

// Liveness reports whether the process itself is up. It never checks
// external dependencies, so a slow database does not take the pod out of
// rotation via the liveness probe (which would trigger a restart instead of
// the traffic-shifting behavior readiness probes are meant for).
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readiness reports whether the service is ready to accept traffic by
// running every registered Checker. It returns 200 only if all checks pass,
// otherwise 503 with a per-dependency breakdown.
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	results := make(gin.H, len(h.checkers))
	allHealthy := true

	for _, checker := range h.checkers {
		if err := checker.Check(ctx); err != nil {
			allHealthy = false
			results[checker.Name()] = gin.H{"status": "down", "error": err.Error()}
			continue
		}
		results[checker.Name()] = gin.H{"status": "up"}
	}

	status := http.StatusOK
	overall := "ok"
	if !allHealthy {
		status = http.StatusServiceUnavailable
		overall = "unavailable"
	}

	c.JSON(status, gin.H{"status": overall, "checks": results})
}
