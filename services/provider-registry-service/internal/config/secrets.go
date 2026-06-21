package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// SecretLoader defines the interface for loading secrets from various sources
type SecretLoader interface {
	LoadSecret(key string) (string, error)
}

// EnvSecretLoader loads secrets from environment variables
type EnvSecretLoader struct {
	Prefix string // Optional prefix for environment variables
}

// LoadSecret loads a secret from an environment variable
func (e *EnvSecretLoader) LoadSecret(key string) (string, error) {
	// Convert key to uppercase, replace dots with underscores
	envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))

	// Add prefix if specified
	if e.Prefix != "" {
		envKey = e.Prefix + "_" + envKey
	}

	value := os.Getenv(envKey)
	if value == "" {
		return "", fmt.Errorf("environment variable %s not found or empty", envKey)
	}

	return value, nil
}

// FileSecretLoader loads secrets from a JSON file
type FileSecretLoader struct {
	FilePath string
	secrets  map[string]string
	loaded   bool
}

// LoadSecret loads a secret from the secrets file
func (f *FileSecretLoader) LoadSecret(key string) (string, error) {
	// Load secrets file if not already loaded
	if !f.loaded {
		if err := f.loadSecretsFile(); err != nil {
			return "", err
		}
	}

	// Look up the key in the secrets map
	value, ok := f.secrets[key]
	if !ok {
		return "", fmt.Errorf("secret key %s not found in secrets file", key)
	}

	return value, nil
}

// loadSecretsFile reads and parses the secrets file
func (f *FileSecretLoader) loadSecretsFile() error {
	// Check if file exists
	if _, err := os.Stat(f.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("secrets file not found: %s", f.FilePath)
	}

	// Read file
	data, err := os.ReadFile(f.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read secrets file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, &f.secrets); err != nil {
		return fmt.Errorf("failed to parse secrets file: %w", err)
	}

	f.loaded = true
	return nil
}

// ChainedSecretLoader tries multiple secret loaders in sequence
type ChainedSecretLoader struct {
	Loaders []SecretLoader
}

// LoadSecret tries each loader in sequence until it finds the secret
func (c *ChainedSecretLoader) LoadSecret(key string) (string, error) {
	var lastErr error

	for _, loader := range c.Loaders {
		value, err := loader.LoadSecret(key)
		if err == nil {
			return value, nil
		}

		lastErr = err
	}

	return "", fmt.Errorf("secret %s not found in any loader: %w", key, lastErr)
}

// DefaultSecretLoader returns a chained secret loader with environment variables
// and an optional secrets file
func DefaultSecretLoader(secretsFilePath string) SecretLoader {
	loaders := []SecretLoader{
		&EnvSecretLoader{Prefix: "DANTE"},
	}

	// Add file loader if a path is provided
	if secretsFilePath != "" {
		loaders = append(loaders, &FileSecretLoader{FilePath: secretsFilePath})
	}

	return &ChainedSecretLoader{Loaders: loaders}
}

// BuildDatabaseURL constructs a database URL from individual components
// This allows us to store credentials separately from the connection string
func BuildDatabaseURL(host, port, dbname, user, password string, sslmode string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)
}

// GetDatabaseCredentials retrieves database credentials from the secret loader
func GetDatabaseCredentials(loader SecretLoader) (host, port, dbname, user, password, sslmode string, err error) {
	host, err = loader.LoadSecret("db.host")
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("failed to load db.host: %w", err)
	}

	port, err = loader.LoadSecret("db.port")
	if err != nil {
		// Default to standard PostgreSQL port if not specified
		port = "5432"
	}

	dbname, err = loader.LoadSecret("db.name")
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("failed to load db.name: %w", err)
	}

	user, err = loader.LoadSecret("db.user")
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("failed to load db.user: %w", err)
	}

	password, err = loader.LoadSecret("db.password")
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("failed to load db.password: %w", err)
	}

	sslmode, err = loader.LoadSecret("db.sslmode")
	if err != nil {
		// Default to require sslmode if not specified
		sslmode = "require"
	}

	return host, port, dbname, user, password, sslmode, nil
}
