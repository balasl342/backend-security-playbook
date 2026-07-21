package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/balac/backend-security-playground/internal/config"
)

func TestNew_JSONFormat(t *testing.T) {
	l, err := New(config.LogConfig{Level: "info", Format: "json"})
	require.NoError(t, err)
	require.NotNil(t, l)
}

func TestNew_ConsoleFormat(t *testing.T) {
	l, err := New(config.LogConfig{Level: "debug", Format: "console"})
	require.NoError(t, err)
	require.NotNil(t, l)
}

func TestNew_InvalidLevel(t *testing.T) {
	_, err := New(config.LogConfig{Level: "not-a-level", Format: "json"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log level")
}

func TestMust_PanicsOnInvalidLevel(t *testing.T) {
	assert.Panics(t, func() {
		Must(config.LogConfig{Level: "bogus", Format: "json"})
	})
}

func TestMust_ReturnsLoggerOnValidConfig(t *testing.T) {
	assert.NotPanics(t, func() {
		l := Must(config.LogConfig{Level: "info", Format: "json"})
		assert.NotNil(t, l)
	})
}

func TestWithContext_FromContext_RoundTrip(t *testing.T) {
	base := zap.NewNop()
	custom := zap.NewExample()

	ctx := WithContext(context.Background(), custom)
	got := FromContext(ctx, base)

	assert.Same(t, custom, got)
}

func TestFromContext_FallsBackWhenAbsent(t *testing.T) {
	base := zap.NewNop()
	got := FromContext(context.Background(), base)
	assert.Same(t, base, got)
}
