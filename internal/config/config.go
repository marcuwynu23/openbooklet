package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// Secret is a string type that refuses String() to prevent accidental logging.
type Secret string

func (s Secret) String() string {
	return "<redacted>"
}

// Config holds the application configuration.
type Config struct {
	Port             int
	Host             string
	DatabasePath     string
	StoragePath      string
	ProviderName     string
	ProviderModel    string
	ProviderEndpoint string
	ProviderAPIKey   Secret
	LogLevel         string
	Timeout          time.Duration
}

// NewDefaultConfig creates a configuration with sensible defaults.
func NewDefaultConfig() *Config {
	return &Config{
		Port:          8080,
		Host:          "localhost",
		DatabasePath:  "./data/openbooklet.db",
		StoragePath:   "./data",
		ProviderName:  "",
		ProviderModel: "",
		LogLevel:      "info",
		Timeout:       30 * time.Second,
	}
}

// Validate checks the configuration for completeness and validity.
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.DatabasePath == "" {
		return errors.New("database path is required")
	}
	if c.StoragePath == "" {
		return errors.New("storage path is required")
	}
	if c.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	return nil
}

// LoadFromEnv reads configuration from environment variables.
// Missing optional fields fall back to defaults.
func LoadFromEnv() *Config {
	cfg := NewDefaultConfig()

	if port := os.Getenv("OPENBOOKLET_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Port)
	}
	if host := os.Getenv("OPENBOOKLET_HOST"); host != "" {
		cfg.Host = host
	}
	if dbPath := os.Getenv("OPENBOOKLET_DB_PATH"); dbPath != "" {
		cfg.DatabasePath = dbPath
	}
	if storagePath := os.Getenv("OPENBOOKLET_STORAGE_PATH"); storagePath != "" {
		cfg.StoragePath = storagePath
	}
	if provider := os.Getenv("OPENBOOKLET_PROVIDER"); provider != "" {
		cfg.ProviderName = provider
	}
	if model := os.Getenv("OPENBOOKLET_MODEL"); model != "" {
		cfg.ProviderModel = model
	}
	if endpoint := os.Getenv("OPENBOOKLET_PROVIDER_ENDPOINT"); endpoint != "" {
		cfg.ProviderEndpoint = endpoint
	}
	if apiKey := os.Getenv("OPENBOOKLET_API_KEY"); apiKey != "" {
		cfg.ProviderAPIKey = Secret(apiKey)
	}
	if logLevel := os.Getenv("OPENBOOKLET_LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}
	if timeoutStr := os.Getenv("OPENBOOKLET_TIMEOUT"); timeoutStr != "" {
		fmt.Sscanf(timeoutStr, "%d", &cfg.Timeout)
	}

	return cfg
}
