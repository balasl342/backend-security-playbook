// Command api is the entrypoint for the backend-security-playground HTTP service.
package main

import (
	"log"
	"os"

	"go.uber.org/zap"

	"github.com/balac/backend-security-playground/internal/config"
	applog "github.com/balac/backend-security-playground/internal/logger"
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
	zapLogger.Info("server bootstrap lands in a later commit")
}
