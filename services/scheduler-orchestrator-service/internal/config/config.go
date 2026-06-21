package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Config holds the application configuration for the scheduler-orchestrator service.
// It includes settings for its own HTTP server, Consul, NATS, and interaction
// with the Provider Registry service.
type Config struct {
	Port           string        `yaml:"port"`
	LogLevel       string        `yaml:"log_level"`
	RequestTimeout time.Duration `yaml:"request_timeout"`

	// Database Configuration
	DatabaseURL string `yaml:"database_url"`

	// Consul Configuration
	ConsulAddress       string        `yaml:"consul_address"`
	ServiceName         string        `yaml:"service_name"`
	ServiceIDPrefix     string        `yaml:"service_id_prefix"`
	ServiceTags         []string      `yaml:"service_tags"`
	HealthCheckPath     string        `yaml:"health_check_path"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout"`

	// NATS Configuration
	NatsAddress                      string `yaml:"nats_address"`
	NatsJobSubmissionSubject         string `yaml:"nats_job_submission_subject"`
	NatsJobQueueGroup                string `yaml:"nats_job_queue_group"`
	NatsTaskDispatchSubjectPrefix    string `yaml:"nats_task_dispatch_subject_prefix"`
	NatsJobStatusUpdateSubjectPrefix string `yaml:"nats_job_status_update_subject_prefix"`

	// Provider Registry Service Configuration
	ProviderRegistryServiceName string `yaml:"provider_registry_service_name"`
	// ProviderRegistryURL string `yaml:"provider_registry_url,omitempty"` // Alternative if not using Consul discovery

	// Scheduling Algorithm Configuration
	SchedulingStrategy string `yaml:"scheduling_strategy"`
	JobDefaultPriority int    `yaml:"job_default_priority"`

	// Resource Query Configuration
	ProviderQueryTimeout time.Duration `yaml:"provider_query_timeout"`
}

// LoadConfig reads configuration from the given YAML file path.
// It creates a default config file if it doesn't exist.
func LoadConfig(path string) (*Config, error) {
	defaultConfig := &Config{
		Port:           ":8003",
		LogLevel:       "info",
		RequestTimeout: 30 * time.Second,

		DatabaseURL: "postgresql://user:password@localhost:5432/dante_scheduler?sslmode=disable", // Default DB URL

		ConsulAddress:       "localhost:8500",
		ServiceName:         "scheduler-orchestrator",
		ServiceIDPrefix:     "scheduler-orchestrator-",
		ServiceTags:         []string{"dante", "scheduler"},
		HealthCheckPath:     "/health",
		HealthCheckInterval: 10 * time.Second,
		HealthCheckTimeout:  2 * time.Second,

		NatsAddress:                      "nats://localhost:4222",
		NatsJobSubmissionSubject:         "jobs.submitted",
		NatsJobQueueGroup:                "scheduler-group",
		NatsTaskDispatchSubjectPrefix:    "tasks.dispatch",
		NatsJobStatusUpdateSubjectPrefix: "jobs.status",

		ProviderRegistryServiceName: "provider-registry",

		SchedulingStrategy: "round-robin",
		JobDefaultPriority: 5,

		ProviderQueryTimeout: 5 * time.Second,
	}

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		data, marshalErr := yaml.Marshal(defaultConfig)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal default config: %w", marshalErr)
		}
		if mkdirErr := os.MkdirAll(filepath.Dir(path), 0755); mkdirErr != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", mkdirErr)
		}
		if writeErr := os.WriteFile(path, data, 0644); writeErr != nil {
			return nil, fmt.Errorf("failed to write default config file: %w", writeErr)
		}
		return defaultConfig, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to check config file: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config data: %w", err)
	}

	applyDefaultsIfNotSet(&cfg, defaultConfig)

	return &cfg, nil
}

func applyDefaultsIfNotSet(cfg *Config, defaults *Config) {
	if cfg.Port == "" {
		cfg.Port = defaults.Port
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = defaults.LogLevel
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = defaults.RequestTimeout
	}
	if cfg.DatabaseURL == "" { // Apply default for DatabaseURL if missing
		cfg.DatabaseURL = defaults.DatabaseURL
	}
	if cfg.ConsulAddress == "" {
		cfg.ConsulAddress = defaults.ConsulAddress
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = defaults.ServiceName
	}
	if cfg.ServiceIDPrefix == "" {
		cfg.ServiceIDPrefix = defaults.ServiceIDPrefix
	}
	if len(cfg.ServiceTags) == 0 {
		cfg.ServiceTags = defaults.ServiceTags
	}
	if cfg.HealthCheckPath == "" {
		cfg.HealthCheckPath = defaults.HealthCheckPath
	}
	if cfg.HealthCheckInterval == 0 {
		cfg.HealthCheckInterval = defaults.HealthCheckInterval
	}
	if cfg.HealthCheckTimeout == 0 {
		cfg.HealthCheckTimeout = defaults.HealthCheckTimeout
	}
	if cfg.NatsAddress == "" {
		cfg.NatsAddress = defaults.NatsAddress
	}
	if cfg.NatsJobSubmissionSubject == "" {
		cfg.NatsJobSubmissionSubject = defaults.NatsJobSubmissionSubject
	}
	if cfg.NatsJobQueueGroup == "" {
		cfg.NatsJobQueueGroup = defaults.NatsJobQueueGroup
	}
	if cfg.NatsTaskDispatchSubjectPrefix == "" {
		cfg.NatsTaskDispatchSubjectPrefix = defaults.NatsTaskDispatchSubjectPrefix
	}
	if cfg.NatsJobStatusUpdateSubjectPrefix == "" {
		cfg.NatsJobStatusUpdateSubjectPrefix = defaults.NatsJobStatusUpdateSubjectPrefix
	}
	if cfg.ProviderRegistryServiceName == "" {
		cfg.ProviderRegistryServiceName = defaults.ProviderRegistryServiceName
	}
	if cfg.SchedulingStrategy == "" {
		cfg.SchedulingStrategy = defaults.SchedulingStrategy
	}
	if cfg.JobDefaultPriority == 0 { // Assuming 0 is not a valid priority, so it acts as unset
		cfg.JobDefaultPriority = defaults.JobDefaultPriority
	}
	if cfg.ProviderQueryTimeout == 0 {
		cfg.ProviderQueryTimeout = defaults.ProviderQueryTimeout
	}
}

func GenerateServiceID(prefix string) string {
	return prefix + uuid.New().String()
}
