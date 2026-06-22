package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dante-gpu/dante-backend/provider-daemon/internal/billing"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// NatsConfig holds NATS specific configuration.
type NatsConfig struct {
	URL                        string        `yaml:"url"`
	ConnectTimeout             time.Duration `yaml:"connect_timeout"`
	ReconnectWait              time.Duration `yaml:"reconnect_wait"`
	MaxReconnects              int           `yaml:"max_reconnects"`
	SubjectPrefix              string        `yaml:"subject_prefix"`
	TaskDispatchSubjectPattern string        `yaml:"task_dispatch_subject_pattern"`
	JobStatusUpdateSubjectPrefix string      `yaml:"job_status_update_subject_prefix"`
	StreamNamePrefix           string        `yaml:"stream_name_prefix"`
	ConsumerNamePrefix         string        `yaml:"consumer_name_prefix"`
	PullMaxWaiting             int           `yaml:"pull_max_waiting"`
	AckWait                    time.Duration `yaml:"ack_wait"`
	MaxDeliver                 int           `yaml:"max_deliver"`
}

// ExecutorSettings holds executor specific configuration.
type ExecutorSettings struct {
	Type           string `yaml:"type"` // "docker" or "script"
	DockerEndpoint string `yaml:"docker_endpoint,omitempty"`
	// WorkspaceDir is now at the top level Config as it's shared
}

// GPUDetectorSettings holds GPU detector specific configuration.
type GPUDetectorSettings struct {
	DetectionInterval time.Duration `yaml:"detection_interval"`
	NvidiaSmiPath     string        `yaml:"nvidia_smi_path"`
	// Other paths like rocm-smi if needed
}

// Config holds the application configuration for the provider daemon.
type Config struct {
	InstanceID string `yaml:"instance_id"`
	LogLevel   string `yaml:"log_level"`
	// General request timeout, e.g., for HTTP calls to other services
	RequestTimeout time.Duration `yaml:"request_timeout"`

	// Configurations for sub-components, now defined locally
	NatsConfig        NatsConfig          `yaml:"nats"`
	ExecutorConfig    ExecutorSettings    `yaml:"executor"`     // Uses locally defined ExecutorSettings
	GPUDetectorConfig GPUDetectorSettings `yaml:"gpu_detector"` // Uses locally defined GPUDetectorSettings

	ProviderRegistryServiceName string        `yaml:"provider_registry_service_name"`
	ProviderRegistryURL         string        `yaml:"provider_registry_url,omitempty"`
	ProviderHeartbeatInterval   time.Duration `yaml:"provider_heartbeat_interval"`

	WorkspaceDir      string   `yaml:"workspace_dir"` // Moved here, shared by executors
	ManagedGPUIDs     []string `yaml:"managed_gpu_ids,omitempty"`
	MaxConcurrentJobs uint32   `yaml:"max_concurrent_jobs"`
	PreferredCurrency string   `yaml:"preferred_currency"`

	// Provider specific settings (for GUI interaction and default behaviors)
	DefaultHourlyRateDGPU float64 `yaml:"default_hourly_rate_dgpu"`
	MinJobDurationMinutes uint32  `yaml:"min_job_duration_minutes"`

	GpuRentalConfigs []GpuRentalConfigEntry `yaml:"gpu_rental_configs,omitempty"`

	Logger              *zap.Logger    `yaml:"-"`
	BillingClientConfig billing.Config `yaml:"billing_client"`

	shutdownTimeout time.Duration
}

// GpuRentalConfigEntry defines the structure for an individual GPU's rental settings.
type GpuRentalConfigEntry struct {
	GpuID                 string  `yaml:"gpu_id"`
	IsAvailableForRent    bool    `yaml:"is_available_for_rent"`
	CurrentHourlyRateDGPU float32 `yaml:"current_hourly_rate_dgpu"`
}

// LoadConfig reads configuration from the given YAML file path.
// It creates a default config file if it doesn't exist.
func LoadConfig(path string, logger *zap.Logger) (*Config, error) {
	hostname, _ := os.Hostname()
	defaultInstanceID := "provider-" + hostname
	if defaultInstanceID == "provider-" {
		defaultInstanceID = "provider-daemon-unknown"
	}

	defaultConfig := &Config{
		InstanceID:     defaultInstanceID,
		LogLevel:       "info",
		RequestTimeout: 30 * time.Second,
		NatsConfig: NatsConfig{ // Initialize locally defined NatsConfig
			URL:                        "nats://localhost:4222",
			ConnectTimeout:             5 * time.Second,
			ReconnectWait:              5 * time.Second,
			MaxReconnects:              -1, // Infinite
			SubjectPrefix:              "tasks.dispatch",
			// Must contain a single %s for the instance id (fmt.Sprintf) and live
			// under the TASKS JetStream stream's subjects (tasks.>). The previous
			// default "dante.tasks.dispatch.>" had no %s (so Sprintf garbled it)
			// and the wrong prefix (no stream matched), breaking task dispatch.
			TaskDispatchSubjectPattern: "tasks.dispatch.%s.*",
			JobStatusUpdateSubjectPrefix: "dante.provider.status",
			StreamNamePrefix:           "DANTE_TASKS_",
			ConsumerNamePrefix:         "provider_daemon_",
			PullMaxWaiting:             5,
			AckWait:                    30 * time.Second,
			MaxDeliver:                 5,
		},
		ExecutorConfig: ExecutorSettings{ // Initialize locally defined ExecutorSettings
			Type:           "docker",
			DockerEndpoint: "unix:///var/run/docker.sock",
		},
		GPUDetectorConfig: GPUDetectorSettings{ // Initialize locally defined GPUDetectorSettings
			DetectionInterval: 1 * time.Minute,
			NvidiaSmiPath:     "nvidia-smi",
		},
		ProviderRegistryServiceName: "provider-registry",
		ProviderHeartbeatInterval:   30 * time.Second,
		WorkspaceDir:                filepath.Join(os.TempDir(), "dante_tasks"), // Default WorkspaceDir
		MaxConcurrentJobs:           1,
		PreferredCurrency:           "DGPU",
		DefaultHourlyRateDGPU:       1.0, // Default value
		MinJobDurationMinutes:       5,   // Default value
		GpuRentalConfigs:            make([]GpuRentalConfigEntry, 0),
		BillingClientConfig: billing.Config{
			BaseURL: "http://localhost:8081/api/v1/billing",
		},
		shutdownTimeout: 10 * time.Second,
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
		fmt.Printf("Default configuration file created at %s\n", path)
		// For a new default config, ensure Logger is set before returning
		defaultConfig.Logger = logger
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

	cfg.Logger = logger

	return &cfg, nil
}

// applyDefaultsIfNotSet applies default values to cfg fields if they are zero-valued.
func applyDefaultsIfNotSet(cfg *Config, defaults *Config) {
	if cfg.InstanceID == "" {
		cfg.InstanceID = defaults.InstanceID
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = defaults.LogLevel
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = defaults.RequestTimeout
	}
	// NATS Config
	if cfg.NatsConfig.URL == "" {
		cfg.NatsConfig.URL = defaults.NatsConfig.URL
	}
	if cfg.NatsConfig.ConnectTimeout == 0 {
		cfg.NatsConfig.ConnectTimeout = defaults.NatsConfig.ConnectTimeout
	}
	if cfg.NatsConfig.ReconnectWait == 0 {
		cfg.NatsConfig.ReconnectWait = defaults.NatsConfig.ReconnectWait
	}
	// MaxReconnects can be 0 or -1, so only default if not explicitly set (how to check if not set vs set to 0?)
	// For now, assume if it's 0 and default isn't, apply. This logic might need refinement if 0 is a valid user setting.
	if cfg.NatsConfig.MaxReconnects == 0 && defaults.NatsConfig.MaxReconnects != 0 {
		cfg.NatsConfig.MaxReconnects = defaults.NatsConfig.MaxReconnects
	}
	if cfg.NatsConfig.SubjectPrefix == "" {
		cfg.NatsConfig.SubjectPrefix = defaults.NatsConfig.SubjectPrefix
	}
	if cfg.NatsConfig.TaskDispatchSubjectPattern == "" {
		cfg.NatsConfig.TaskDispatchSubjectPattern = defaults.NatsConfig.TaskDispatchSubjectPattern
	}
	if cfg.NatsConfig.JobStatusUpdateSubjectPrefix == "" {
		cfg.NatsConfig.JobStatusUpdateSubjectPrefix = defaults.NatsConfig.JobStatusUpdateSubjectPrefix
	}
	if cfg.NatsConfig.StreamNamePrefix == "" {
		cfg.NatsConfig.StreamNamePrefix = defaults.NatsConfig.StreamNamePrefix
	}
	if cfg.NatsConfig.ConsumerNamePrefix == "" {
		cfg.NatsConfig.ConsumerNamePrefix = defaults.NatsConfig.ConsumerNamePrefix
	}
	if cfg.NatsConfig.PullMaxWaiting == 0 {
		cfg.NatsConfig.PullMaxWaiting = defaults.NatsConfig.PullMaxWaiting
	}
	if cfg.NatsConfig.AckWait == 0 {
		cfg.NatsConfig.AckWait = defaults.NatsConfig.AckWait
	}
	if cfg.NatsConfig.MaxDeliver == 0 {
		cfg.NatsConfig.MaxDeliver = defaults.NatsConfig.MaxDeliver
	}

	// Executor Config
	if cfg.ExecutorConfig.Type == "" {
		cfg.ExecutorConfig.Type = defaults.ExecutorConfig.Type
	}
	if cfg.ExecutorConfig.DockerEndpoint == "" && defaults.ExecutorConfig.DockerEndpoint != "" {
		cfg.ExecutorConfig.DockerEndpoint = defaults.ExecutorConfig.DockerEndpoint
	}

	// GPU Detector Config
	if cfg.GPUDetectorConfig.DetectionInterval == 0 {
		cfg.GPUDetectorConfig.DetectionInterval = defaults.GPUDetectorConfig.DetectionInterval
	}
	if cfg.GPUDetectorConfig.NvidiaSmiPath == "" {
		cfg.GPUDetectorConfig.NvidiaSmiPath = defaults.GPUDetectorConfig.NvidiaSmiPath
	}

	if cfg.ProviderRegistryServiceName == "" && cfg.ProviderRegistryURL == "" {
		cfg.ProviderRegistryServiceName = defaults.ProviderRegistryServiceName
	}
	if cfg.ProviderHeartbeatInterval == 0 {
		cfg.ProviderHeartbeatInterval = defaults.ProviderHeartbeatInterval
	}
	if cfg.WorkspaceDir == "" { // Applied WorkspaceDir default
		cfg.WorkspaceDir = defaults.WorkspaceDir
	}
	if cfg.MaxConcurrentJobs == 0 {
		cfg.MaxConcurrentJobs = defaults.MaxConcurrentJobs
	}
	if cfg.PreferredCurrency == "" {
		cfg.PreferredCurrency = defaults.PreferredCurrency
	}
	// For GpuRentalConfigs, if nil, use default (empty slice). If user provides empty slice, it's fine.
	if cfg.GpuRentalConfigs == nil {
		cfg.GpuRentalConfigs = defaults.GpuRentalConfigs
	}
	if cfg.BillingClientConfig.BaseURL == "" {
		cfg.BillingClientConfig.BaseURL = defaults.BillingClientConfig.BaseURL
	}
	if cfg.shutdownTimeout == 0 {
		cfg.shutdownTimeout = defaults.shutdownTimeout
	}
	if cfg.DefaultHourlyRateDGPU == 0 && defaults.DefaultHourlyRateDGPU != 0 { // Check for 0.0 for float
		cfg.DefaultHourlyRateDGPU = defaults.DefaultHourlyRateDGPU
	}
	if cfg.MinJobDurationMinutes == 0 && defaults.MinJobDurationMinutes != 0 {
		cfg.MinJobDurationMinutes = defaults.MinJobDurationMinutes
	}
}

// SaveConfig saves the current configuration to the specified path.
// NOTE: This will overwrite the existing config file.
func SaveConfig(cfg *Config, path string) error {
	cfg.Logger.Info("Attempting to save configuration", zap.String("path", path))

	// Ensure the Logger field is not marshaled into the YAML.
	// A common way is to have a separate struct for marshalling or ensure yaml:"-" is effective.
	// Given Logger is already yaml:"-", this should be fine.

	data, err := yaml.Marshal(cfg)
	if err != nil {
		cfg.Logger.Error("Failed to marshal config for saving", zap.Error(err))
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		cfg.Logger.Error("Failed to write config file", zap.String("path", path), zap.Error(err))
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}

	cfg.Logger.Info("Configuration saved successfully", zap.String("path", path))
	return nil
}
