package server

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/balac/backend-security-playground/internal/config"
)

func testCfg(port int) config.ServerConfig {
	return config.ServerConfig{
		Host:            "127.0.0.1",
		Port:            port,
		ReadTimeout:     2 * time.Second,
		WriteTimeout:    2 * time.Second,
		IdleTimeout:     2 * time.Second,
		ShutdownTimeout: 2 * time.Second,
	}
}

func TestServer_RunAndGracefulShutdown(t *testing.T) {
	cfg := testCfg(18081)
	srv := New(cfg, zap.NewNop(), "test")

	srv.Engine().GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	ctx, cancel := context.WithCancel(context.Background())
	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- srv.Run(ctx)
	}()

	waitForServer(t, fmt.Sprintf("http://%s/ping", cfg.Addr()))

	resp, err := http.Get(fmt.Sprintf("http://%s/ping", cfg.Addr()))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cancel()

	select {
	case err := <-runErrCh:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestServer_EngineExposesRouter(t *testing.T) {
	srv := New(testCfg(0), zap.NewNop(), "test")
	assert.NotNil(t, srv.Engine())
}

func waitForServer(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready in time", url)
}
