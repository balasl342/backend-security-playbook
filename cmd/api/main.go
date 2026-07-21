// Command api is the entrypoint for the backend-security-playground HTTP service.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/balac/backend-security-playground/internal/config"
	applog "github.com/balac/backend-security-playground/internal/logger"
	"github.com/balac/backend-security-playground/internal/server"
)

func main() {
	configPath := os.Getenv("APP_CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	zapLogger, err := applog.New(cfg.Log)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync() //nolint:errcheck

	zapLogger.Info("configuration loaded",
		zap.String("env", cfg.Env),
		zap.String("addr", cfg.Server.Addr()),
		zap.String("crypto_mode", cfg.Crypto.Mode),
	)

	srv := server.New(cfg.Server, zapLogger, cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := srv.Run(ctx); err != nil {
		zapLogger.Fatal("server exited with error", zap.Error(err))
	}
}
