// Package server wires the Gin engine to an *http.Server with a graceful
// shutdown lifecycle.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/balac/backend-security-playground/internal/config"
	"github.com/balac/backend-security-playground/internal/middleware"
)

// Server wraps an http.Server bound to a Gin engine.
type Server struct {
	httpServer *http.Server
	engine     *gin.Engine
	logger     *zap.Logger
	cfg        config.ServerConfig
}

// New constructs a Server. env selects Gin's run mode: "production" runs in
// gin.ReleaseMode, anything else (e.g. "development") runs in gin.DebugMode.
// Callers register routes on Engine() before calling Run.
func New(cfg config.ServerConfig, log *zap.Logger, env string) *Server {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(
		middleware.RequestID(),
		middleware.Recovery(log),
		middleware.AccessLog(log),
	)

	httpServer := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      engine,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &Server{
		httpServer: httpServer,
		engine:     engine,
		logger:     log,
		cfg:        cfg,
	}
}

// Engine exposes the underlying Gin engine for route registration.
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Run starts the HTTP server and blocks until ctx is cancelled, at which
// point it attempts a graceful shutdown bounded by cfg.ShutdownTimeout.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("http server listening", zap.String("addr", s.cfg.Addr()))
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("server: listen and serve: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("shutdown signal received, draining connections")
		return s.shutdown()
	}
}

func (s *Server) shutdown() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server: graceful shutdown: %w", err)
	}

	s.logger.Info("http server shut down cleanly")
	return nil
}
