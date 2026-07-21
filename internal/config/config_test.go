package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("")
	require.NoError(t, err)

	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "0.0.0.0:8080", cfg.Server.Addr())
	assert.Equal(t, 10*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, "envelope", cfg.Crypto.Mode)
	assert.Equal(t, "local", cfg.Crypto.KMSProvider)
	assert.Equal(t, "env", cfg.Secrets.Provider)
}

func TestLoad_FromYAMLFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.yaml"
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  port: 9090
crypto:
  mode: plaintext
  kms_provider: local
`), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "plaintext", cfg.Crypto.Mode)
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.yaml"
	require.NoError(t, os.WriteFile(path, []byte("server:\n  port: 9090\n"), 0o600))

	t.Setenv("APP_SERVER_PORT", "7070")

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, 7070, cfg.Server.Port)
}

func TestLoad_MissingFileFallsBackToDefaults(t *testing.T) {
	cfg, err := Load("does/not/exist.yaml")
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestDatabaseConfig_DSN(t *testing.T) {
	d := DatabaseConfig{
		Host: "db", Port: 5432, User: "u", Password: "p", Name: "n", SSLMode: "disable",
	}
	assert.Equal(t, "host=db port=5432 user=u password=p dbname=n sslmode=disable", d.DSN())
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name:    "invalid crypto mode",
			mutate:  func(c *Config) { c.Crypto.Mode = "bogus" },
			wantErr: "crypto.mode",
		},
		{
			name:    "invalid kms provider",
			mutate:  func(c *Config) { c.Crypto.KMSProvider = "bogus" },
			wantErr: "crypto.kms_provider",
		},
		{
			name: "aws kms without key id",
			mutate: func(c *Config) {
				c.Crypto.KMSProvider = "aws"
				c.Crypto.AWSKeyID = ""
			},
			wantErr: "crypto.aws_key_id",
		},
		{
			name:    "invalid secrets provider",
			mutate:  func(c *Config) { c.Secrets.Provider = "bogus" },
			wantErr: "secrets.provider",
		},
		{
			name:    "invalid port",
			mutate:  func(c *Config) { c.Server.Port = 70000 },
			wantErr: "server.port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load("")
			require.NoError(t, err)
			tt.mutate(cfg)

			err = cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestValidate_DefaultsAreValid(t *testing.T) {
	cfg, err := Load("")
	require.NoError(t, err)
	assert.NoError(t, cfg.Validate())
}
