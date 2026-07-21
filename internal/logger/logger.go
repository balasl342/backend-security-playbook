// Package logger provides a Zap-backed structured logger configured from
// internal/config, plus helpers for propagating a request-scoped logger
// through context.Context.
package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/balac/backend-security-playground/internal/config"
)

type ctxKey struct{}

// New builds a *zap.Logger from the given log configuration.
//
// format "json" produces production-style structured JSON output; any other
// value (e.g. "console") produces human-readable colorized output suitable
// for local development.
func New(cfg config.LogConfig) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("logger: invalid log level %q: %w", cfg.Level, err)
	}

	var zapCfg zap.Config
	if cfg.Format == "json" {
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	l, err := zapCfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		return nil, fmt.Errorf("logger: build: %w", err)
	}

	return l, nil
}

// Must is a convenience wrapper around New that panics on error. Intended
// for use during process startup where a broken logger config should fail
// fast.
func Must(cfg config.LogConfig) *zap.Logger {
	l, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return l
}

// WithContext returns a new context carrying the given logger.
func WithContext(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the logger stored in ctx by WithContext, or fallback
// if none is present.
func FromContext(ctx context.Context, fallback *zap.Logger) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return fallback
}
