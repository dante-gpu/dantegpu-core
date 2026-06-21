package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ServerConfig holds the configuration for the HTTP server.
type ServerConfig struct {
	Host string `yaml:"host"` // e.g., "0.0.0.0" or "" for all interfaces
	Port int    `yaml:"port"` // e.g., 8082
}

// ConsulRegistrationConfig holds details for how this service registers with Consul.
type ConsulRegistrationConfig struct {
	ServiceName         string        `yaml:"service_name"`
	ServiceIDPrefix     string        `yaml:"service_id_prefix"`
	ServiceTags         []string      `yaml:"service_tags"`
	HealthCheckPath     string        `yaml:"health_check_path"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout"`
}

// ConsulConfig holds general Consul client configuration and registration details.
type ConsulConfig struct {
	Enabled      bool                     `yaml:"enabled"`
	Address      string                   `yaml:"address"` // Consul agent address
	Registration ConsulRegistrationConfig `yaml:"registration"`
}

// MinioConfig holds the configuration for MinIO.
type MinioConfig struct {
	Endpoint                string `yaml:"endpoint"`
	AccessKeyID             string `yaml:"accessKeyID"`
	SecretAccessKey         string `yaml:"secretAccessKey"`
	UseSSL                  bool   `yaml:"useSSL"`
	Region                  string `yaml:"region"`
	DefaultBucket           string `yaml:"defaultBucket"`
	AutoCreateDefaultBucket bool   `yaml:"autoCreateDefaultBucket"`
}

// Config holds the overall application configuration.
type Config struct {
	InstanceID     string        `yaml:"instance_id"`     // Unique ID for this service instance
	LogLevel       string        `yaml:"log_level"`       // e.g., "debug", "info", "warn", "error"
	RequestTimeout time.Duration `yaml:"request_timeout"` // Default timeout for HTTP server requests

	Server ServerConfig `yaml:"server"`
	Consul ConsulConfig `yaml:"consul"`
	Minio  MinioConfig  `yaml:"minio"`

	Logger *zap.Logger `yaml:"-"` // Logger is not read from YAML
}

// LoadConfig reads configuration from the given YAML file path.
// It creates a default config file if it doesn't exist and applies defaults for missing fields.
func LoadConfig(configPath string, logger *zap.Logger) (*Config, error) {
	if logger == nil {
		var err error
		logger, err = zap.NewDevelopment() // Fallback logger
		if err != nil {
			return nil, fmt.Errorf("failed to initialize fallback logger: %w", err)
		}
		logger.Warn("No logger provided to LoadConfig, using temporary development logger.")
	}

	defaultConfig := getDefaultConfig()

	absPath, err := GetAbsPath(configPath)
	if err != nil {
		logger.Error("Failed to get absolute path for config file", zap.String("path", configPath), zap.Error(err))
		return nil, err
	}

	logger.Info("Loading configuration", zap.String("path", absPath))

	_, err = os.Stat(absPath)
	if os.IsNotExist(err) {
		logger.Info("Configuration file not found, creating with default values", zap.String("path", absPath))
		data, marshalErr := yaml.Marshal(defaultConfig)
		if marshalErr != nil {
			logger.Error("Failed to marshal default config", zap.Error(marshalErr))
			return nil, fmt.Errorf("failed to marshal default config: %w", marshalErr)
		}
		if mkdirErr := os.MkdirAll(filepath.Dir(absPath), 0755); mkdirErr != nil {
			logger.Error("Failed to create config directory", zap.String("path", filepath.Dir(absPath)), zap.Error(mkdirErr))
			return nil, fmt.Errorf("failed to create config directory: %w", mkdirErr)
		}
		if writeErr := os.WriteFile(absPath, data, 0644); writeErr != nil {
			logger.Error("Failed to write default config file", zap.String("path", absPath), zap.Error(writeErr))
			return nil, fmt.Errorf("failed to write default config file: %w", writeErr)
		}
		// Use a copy of defaultConfig to avoid modifying the global one if it's a pointer
		cfgToReturn := *defaultConfig
		cfgToReturn.Logger = logger
		if cfgToReturn.InstanceID == "" { // Generate InstanceID if default didn't set one
			cfgToReturn.InstanceID = uuid.New().String()
		}
		logger.Info("Default configuration file created and loaded", zap.String("instance_id", cfgToReturn.InstanceID))
		return &cfgToReturn, nil
	} else if err != nil {
		logger.Error("Error checking config file status", zap.String("path", absPath), zap.Error(err))
		return nil, fmt.Errorf("failed to check config file %s: %w", absPath, err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		logger.Error("Failed to read configuration file", zap.String("path", absPath), zap.Error(err))
		return nil, fmt.Errorf("failed to read config file %s: %w", absPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Error("Failed to unmarshal configuration YAML", zap.String("path", absPath), zap.Error(err))
		return nil, fmt.Errorf("failed to unmarshal config YAML from %s: %w", absPath, err)
	}

	// Apply defaults for any fields that might be missing in the loaded config
	applyDefaultsIfNotSet(&cfg, defaultConfig)

	if cfg.InstanceID == "" {
		cfg.InstanceID = uuid.New().String()
		logger.Info("Generated new InstanceID for service", zap.String("instance_id", cfg.InstanceID))
	}
	cfg.Logger = logger

	logger.Info("Configuration loaded and parsed successfully", zap.String("instance_id", cfg.InstanceID))
	return &cfg, nil
}

func getDefaultConfig() *Config {
	return &Config{
		InstanceID:     uuid.New().String(), // Generate a default instance ID
		LogLevel:       "info",
		RequestTimeout: 30 * time.Second,
		Server: ServerConfig{
			Host: "", // Listen on all interfaces by default
			Port: 8082,
		},
		Consul: ConsulConfig{
			Enabled: true,
			Address: "localhost:8500",
			Registration: ConsulRegistrationConfig{
				ServiceName:         "storage-service",
				ServiceIDPrefix:     "storage-svc-",
				ServiceTags:         []string{"dante", "storage"},
				HealthCheckPath:     "/health",
				HealthCheckInterval: 10 * time.Second,
				HealthCheckTimeout:  5 * time.Second,
			},
		},
		Minio: MinioConfig{
			Endpoint:                "localhost:9000",
			AccessKeyID:             "YOUR_MINIO_ACCESS_KEY", // Placeholder
			SecretAccessKey:         "YOUR_MINIO_SECRET_KEY", // Placeholder
			UseSSL:                  false,
			Region:                  "us-east-1", // Default MinIO region
			DefaultBucket:           "dante-storage",
			AutoCreateDefaultBucket: true,
		},
	}
}

// applyDefaultsIfNotSet applies default values from `defaults` to `cfg` for zero-valued fields.
func applyDefaultsIfNotSet(cfg *Config, defaults *Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = defaults.LogLevel
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = defaults.RequestTimeout
	}

	// Server defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = defaults.Server.Port
	}
	// Host defaults to empty string (listen on all interfaces), which is fine.

	// Consul defaults
	if cfg.Consul.Address == "" {
		cfg.Consul.Address = defaults.Consul.Address
	}
	// Note: Consul.Enabled defaults to false (Go default for bool) if not set in YAML.
	// If we want default to true, it should be set explicitly in getDefaultConfig() and checked here if needed.
	// Or, the user must always specify `enabled: true` or `enabled: false` in their YAML.
	// For now, we rely on the `getDefaultConfig()` having `Enabled: true`.

	// Consul Registration defaults
	if cfg.Consul.Registration.ServiceName == "" {
		cfg.Consul.Registration.ServiceName = defaults.Consul.Registration.ServiceName
	}
	if cfg.Consul.Registration.ServiceIDPrefix == "" {
		cfg.Consul.Registration.ServiceIDPrefix = defaults.Consul.Registration.ServiceIDPrefix
	}
	if len(cfg.Consul.Registration.ServiceTags) == 0 {
		cfg.Consul.Registration.ServiceTags = defaults.Consul.Registration.ServiceTags
	}
	if cfg.Consul.Registration.HealthCheckPath == "" {
		cfg.Consul.Registration.HealthCheckPath = defaults.Consul.Registration.HealthCheckPath
	}
	if cfg.Consul.Registration.HealthCheckInterval == 0 {
		cfg.Consul.Registration.HealthCheckInterval = defaults.Consul.Registration.HealthCheckInterval
	}
	if cfg.Consul.Registration.HealthCheckTimeout == 0 {
		cfg.Consul.Registration.HealthCheckTimeout = defaults.Consul.Registration.HealthCheckTimeout
	}

	// Minio defaults
	if cfg.Minio.Endpoint == "" {
		cfg.Minio.Endpoint = defaults.Minio.Endpoint
	}
	if cfg.Minio.AccessKeyID == "" {
		cfg.Minio.AccessKeyID = defaults.Minio.AccessKeyID
	}
	if cfg.Minio.SecretAccessKey == "" {
		cfg.Minio.SecretAccessKey = defaults.Minio.SecretAccessKey
	}
	// UseSSL defaults to false.
	if cfg.Minio.Region == "" {
		cfg.Minio.Region = defaults.Minio.Region
	}
	if cfg.Minio.DefaultBucket == "" {
		cfg.Minio.DefaultBucket = defaults.Minio.DefaultBucket
	}
	// AutoCreateDefaultBucket defaults to false. If we want default true, handle as with Consul.Enabled.

	// InstanceID is handled separately after loading if still empty.
}

// GetAbsPath returns the absolute path for a given relative path to the config file.
func GetAbsPath(relativePath string) (string, error) {
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", relativePath, err)
	}
	return absPath, nil
}
