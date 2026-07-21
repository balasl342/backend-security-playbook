// Command api is the entrypoint for the backend-security-playground HTTP service.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/balac/backend-security-playground/internal/config"
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

	fmt.Printf(
		"backend-security-playground: config loaded (env=%s, addr=%s, crypto.mode=%s); server bootstrap lands in a later commit\n",
		cfg.Env, cfg.Server.Addr(), cfg.Crypto.Mode,
	)
}
