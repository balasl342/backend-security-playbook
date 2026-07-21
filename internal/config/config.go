// Package config loads and validates application configuration from
// environment variables, a YAML file, and sane in-code defaults, in that
// order of precedence (env > file > defaults).
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the root configuration object for the service.
type Config struct {
	Env           string              `mapstructure:"env"`
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Redis         RedisConfig         `mapstructure:"redis"`
	Log           LogConfig           `mapstructure:"log"`
	Crypto        CryptoConfig        `mapstructure:"crypto"`
	Secrets       SecretsConfig       `mapstructure:"secrets"`
	Observability ObservabilityConfig `mapstructure:"observability"`
}

// ServerConfig controls the HTTP server.
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// Addr returns the host:port listen address.
func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// DatabaseConfig controls the PostgreSQL connection.
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// DSN returns a PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// RedisConfig controls the optional Redis connection.
type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// LogConfig controls the Zap logger.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// CryptoConfig controls encryption mode and key sourcing (Modules 1, 2, 4, 5).
type CryptoConfig struct {
	// Mode selects application-level encryption behaviour: "plaintext" (Mode A,
	// relies on DB-level TDE) or "envelope" (Mode B, AES-256-GCM with envelope
	// encryption via KMS).
	Mode string `mapstructure:"mode"`

	// KMSProvider selects the key management backend: "local" or "aws".
	KMSProvider string `mapstructure:"kms_provider"`

	// LocalMasterKeyPath is the file holding the local master key material,
	// used when KMSProvider is "local".
	LocalMasterKeyPath string `mapstructure:"local_master_key_path"`

	// AWSKeyID is the KMS key id/ARN used when KMSProvider is "aws".
	AWSKeyID string `mapstructure:"aws_key_id"`

	// AWSRegion is the AWS region for the KMS client.
	AWSRegion string `mapstructure:"aws_region"`
}

// SecretsConfig selects the secret management provider (Module 3).
type SecretsConfig struct {
	// Provider selects the backend: "env", "file", "aws_secrets_manager",
	// "vault", or "azure_key_vault".
	Provider string `mapstructure:"provider"`

	// FilePath is the path to the secrets file, used when Provider is "file".
	FilePath string `mapstructure:"file_path"`

	// AWSSecretsManager settings, used when Provider is "aws_secrets_manager".
	AWSRegion string `mapstructure:"aws_region"`

	// Vault settings, used when Provider is "vault" (mock implementation).
	VaultAddr string `mapstructure:"vault_addr"`

	// Azure Key Vault settings, used when Provider is "azure_key_vault" (mock implementation).
	AzureVaultName string `mapstructure:"azure_vault_name"`
}

// ObservabilityConfig controls metrics and tracing.
type ObservabilityConfig struct {
	MetricsEnabled bool   `mapstructure:"metrics_enabled"`
	TracingEnabled bool   `mapstructure:"tracing_enabled"`
	ServiceName    string `mapstructure:"service_name"`
	OTLPEndpoint   string `mapstructure:"otlp_endpoint"`
}

// Load reads configuration from (in increasing precedence):
//  1. compiled-in defaults
//  2. a YAML file at configPath (if it exists)
//  3. environment variables (prefixed APP_, nested keys joined with '_')
func Load(configPath string) (*Config, error) {
	v := viper.New()

	setDefaults(v)

	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configPath != "" {
		if _, statErr := os.Stat(configPath); statErr == nil {
			v.SetConfigFile(configPath)
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("config: read config file: %w", err)
			}
		} else if !os.IsNotExist(statErr) {
			return nil, fmt.Errorf("config: stat config file: %w", statErr)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config: invalid configuration: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("env", "development")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 10*time.Second)
	v.SetDefault("server.write_timeout", 10*time.Second)
	v.SetDefault("server.idle_timeout", 60*time.Second)
	v.SetDefault("server.shutdown_timeout", 15*time.Second)

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "playground")
	v.SetDefault("database.password", "playground")
	v.SetDefault("database.name", "playground")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	v.SetDefault("crypto.mode", "envelope")
	v.SetDefault("crypto.kms_provider", "local")
	v.SetDefault("crypto.local_master_key_path", "configs/dev-master.key")
	v.SetDefault("crypto.aws_key_id", "")
	v.SetDefault("crypto.aws_region", "us-east-1")

	v.SetDefault("secrets.provider", "env")
	v.SetDefault("secrets.file_path", "configs/secrets.local.yaml")
	v.SetDefault("secrets.aws_region", "us-east-1")
	v.SetDefault("secrets.vault_addr", "http://localhost:8200")
	v.SetDefault("secrets.azure_vault_name", "")

	v.SetDefault("observability.metrics_enabled", true)
	v.SetDefault("observability.tracing_enabled", false)
	v.SetDefault("observability.service_name", "backend-security-playground")
	v.SetDefault("observability.otlp_endpoint", "localhost:4317")
}

// Validate checks that required combinations of settings are coherent.
func (c *Config) Validate() error {
	switch c.Crypto.Mode {
	case "plaintext", "envelope":
	default:
		return fmt.Errorf("crypto.mode must be 'plaintext' or 'envelope', got %q", c.Crypto.Mode)
	}

	switch c.Crypto.KMSProvider {
	case "local", "aws":
	default:
		return fmt.Errorf("crypto.kms_provider must be 'local' or 'aws', got %q", c.Crypto.KMSProvider)
	}

	if c.Crypto.KMSProvider == "aws" && c.Crypto.AWSKeyID == "" {
		return fmt.Errorf("crypto.aws_key_id is required when crypto.kms_provider is 'aws'")
	}

	switch c.Secrets.Provider {
	case "env", "file", "aws_secrets_manager", "vault", "azure_key_vault":
	default:
		return fmt.Errorf("secrets.provider must be one of env|file|aws_secrets_manager|vault|azure_key_vault, got %q", c.Secrets.Provider)
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}

	return nil
}
