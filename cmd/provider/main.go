package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/dante-gpu/dante-backend/common"
)

// TaskExecutionType defines the type of execution
type TaskExecutionType string

const (
	ExecutionTypeDocker TaskExecutionType = "docker"
	ExecutionTypeScript TaskExecutionType = "script"
	ExecutionTypePython TaskExecutionType = "python"
	ExecutionTypeBash   TaskExecutionType = "bash"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusReceived   JobStatus = "received"
	JobStatusValidating JobStatus = "validating"
	JobStatusStarting   JobStatus = "starting"
	JobStatusRunning    JobStatus = "running"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCanceled   JobStatus = "canceled"
	JobStatusTimeout    JobStatus = "timeout"
)

// AppleGPUMetrics represents Apple GPU metrics from system_profiler
type AppleGPUMetrics struct {
	Utilization float64
	Temperature float64
}

// Task represents comprehensive work sent to the provider daemon
type Task struct {
	JobID          string                 `json:"job_id"`
	UserID         string                 `json:"user_id"`
	SessionID      uuid.UUID              `json:"session_id"`
	JobType        string                 `json:"job_type"`
	JobName        string                 `json:"job_name"`
	JobDescription string                 `json:"job_description,omitempty"`
	JobParams      map[string]interface{} `json:"job_params"`
	ExecutionType  TaskExecutionType      `json:"execution_type"`

	// Docker execution parameters
	DockerImage       string            `json:"docker_image,omitempty"`
	DockerCommand     []string          `json:"docker_command,omitempty"`
	DockerEnvironment map[string]string `json:"docker_environment,omitempty"`
	DockerVolumes     []VolumeMount     `json:"docker_volumes,omitempty"`
	DockerGPUAccess   bool              `json:"docker_gpu_access"`

	// Script execution parameters
	Script            string            `json:"script,omitempty"`
	ScriptLanguage    string            `json:"script_language,omitempty"`
	ScriptEnvironment map[string]string `json:"script_environment,omitempty"`

	// Resource requirements and constraints
	Requirements ResourceRequirements `json:"requirements"`
	Constraints  TaskConstraints      `json:"constraints"`

	// Execution metadata
	Priority           int       `json:"priority"`
	MaxDurationMinutes int       `json:"max_duration_minutes"`
	RetryCount         int       `json:"retry_count"`
	SubmittedAt        time.Time `json:"submitted_at"`
	DispatchedAt       time.Time `json:"dispatched_at"`

	// File transfer and storage
	InputFiles       []FileTransfer `json:"input_files,omitempty"`
	OutputFiles      []FileTransfer `json:"output_files,omitempty"`
	WorkspaceCleanup bool           `json:"workspace_cleanup"`

	// Billing and cost control
	MaxCostDGPU       decimal.Decimal `json:"max_cost_dgpu"`
	EstimatedCostDGPU decimal.Decimal `json:"estimated_cost_dgpu"`
}

// VolumeMount represents a Docker volume mount
type VolumeMount struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"read_only"`
	Type     string `json:"type"` // bind, volume, tmpfs
}

// ResourceRequirements specifies resource requirements for the task
type ResourceRequirements struct {
	GPUModel           string  `json:"gpu_model,omitempty"`
	GPUMemoryMB        uint64  `json:"gpu_memory_mb"`
	GPUComputeUnits    float64 `json:"gpu_compute_units"`
	CPUCores           int     `json:"cpu_cores"`
	MemoryMB           uint64  `json:"memory_mb"`
	DiskSpaceMB        uint64  `json:"disk_space_mb"`
	NetworkBandwidthMB uint64  `json:"network_bandwidth_mb"`
}

// TaskConstraints specifies execution constraints
type TaskConstraints struct {
	MaxCPUUsagePercent    float64 `json:"max_cpu_usage_percent"`
	MaxMemoryUsagePercent float64 `json:"max_memory_usage_percent"`
	MaxGPUUsagePercent    float64 `json:"max_gpu_usage_percent"`
	MaxNetworkUsageMB     uint64  `json:"max_network_usage_mb"`
	AllowNetworkAccess    bool    `json:"allow_network_access"`
	AllowFileSystemAccess bool    `json:"allow_filesystem_access"`
}

// FileTransfer represents a file to be transferred
type FileTransfer struct {
	URL         string            `json:"url"`
	Path        string            `json:"path"`
	Checksum    string            `json:"checksum,omitempty"`
	Compression string            `json:"compression,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// TaskStatusUpdate represents comprehensive task status updates
type TaskStatusUpdate struct {
	JobID      string           `json:"job_id"`
	ProviderID string           `json:"provider_id"`
	SessionID  uuid.UUID        `json:"session_id"`
	Status     JobStatus        `json:"status"`
	Progress   float32          `json:"progress"`
	Stage      string           `json:"stage"`
	Message    string           `json:"message"`
	Error      string           `json:"error,omitempty"`
	ErrorCode  string           `json:"error_code,omitempty"`
	Result     TaskResult       `json:"result,omitempty"`
	Metrics    ExecutionMetrics `json:"metrics"`
	Timestamp  time.Time        `json:"timestamp"`

	// Resource usage
	ResourceUsage ResourceUsage `json:"resource_usage"`
	GPUMetrics    []GPUMetrics  `json:"gpu_metrics"`

	// Execution details
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	DurationSeconds float64    `json:"duration_seconds"`
	ExitCode        *int       `json:"exit_code,omitempty"`

	// Logs and output
	LogsURL        string   `json:"logs_url,omitempty"`
	OutputFilesURL []string `json:"output_files_url,omitempty"`

	// Cost and billing
	ActualCostDGPU    decimal.Decimal `json:"actual_cost_dgpu"`
	EnergyConsumedKWh decimal.Decimal `json:"energy_consumed_kwh"`
}

// TaskResult represents the result of task execution
type TaskResult struct {
	Success     bool                   `json:"success"`
	ExitCode    int                    `json:"exit_code"`
	Output      string                 `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	OutputFiles []string               `json:"output_files,omitempty"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
	Artifacts   []Artifact             `json:"artifacts,omitempty"`
}

// Artifact represents a generated artifact from task execution
type Artifact struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Size      int64     `json:"size"`
	Checksum  string    `json:"checksum"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

// ExecutionMetrics provides detailed execution metrics
type ExecutionMetrics struct {
	CPUUsagePercent     float64 `json:"cpu_usage_percent"`
	MemoryUsageMB       uint64  `json:"memory_usage_mb"`
	MemoryUsagePercent  float64 `json:"memory_usage_percent"`
	DiskUsageMB         uint64  `json:"disk_usage_mb"`
	NetworkTxMB         uint64  `json:"network_tx_mb"`
	NetworkRxMB         uint64  `json:"network_rx_mb"`
	ProcessCount        int     `json:"process_count"`
	ThreadCount         int     `json:"thread_count"`
	FileDescriptorCount int     `json:"file_descriptor_count"`
	ContextSwitches     uint64  `json:"context_switches"`
	PageFaults          uint64  `json:"page_faults"`
}

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryMB      uint64    `json:"memory_mb"`
	MemoryPercent float64   `json:"memory_percent"`
	DiskReadMB    uint64    `json:"disk_read_mb"`
	DiskWriteMB   uint64    `json:"disk_write_mb"`
	NetworkTxMB   uint64    `json:"network_tx_mb"`
	NetworkRxMB   uint64    `json:"network_rx_mb"`
	Timestamp     time.Time `json:"timestamp"`
}

// GPUMetrics represents comprehensive GPU metrics
type GPUMetrics struct {
	Index              int          `json:"index"`
	UUID               string       `json:"uuid"`
	Name               string       `json:"name"`
	UtilizationGPU     uint8        `json:"utilization_gpu_percent"`
	UtilizationMemory  uint8        `json:"utilization_memory_percent"`
	UtilizationEncoder uint8        `json:"utilization_encoder_percent"`
	UtilizationDecoder uint8        `json:"utilization_decoder_percent"`
	MemoryTotal        uint64       `json:"memory_total_mb"`
	MemoryUsed         uint64       `json:"memory_used_mb"`
	MemoryFree         uint64       `json:"memory_free_mb"`
	Temperature        uint8        `json:"temperature_celsius"`
	PowerDraw          uint32       `json:"power_draw_watts"`
	PowerLimit         uint32       `json:"power_limit_watts"`
	ClockCore          uint32       `json:"clock_core_mhz"`
	ClockMemory        uint32       `json:"clock_memory_mhz"`
	ClockSM            uint32       `json:"clock_sm_mhz"`
	ClockVideo         uint32       `json:"clock_video_mhz"`
	FanSpeed           uint8        `json:"fan_speed_percent"`
	Processes          []GPUProcess `json:"processes"`
	Timestamp          time.Time    `json:"timestamp"`
}

// GPUProcess represents a process using the GPU
type GPUProcess struct {
	PID         uint32 `json:"pid"`
	ProcessName string `json:"process_name"`
	MemoryUsage uint64 `json:"memory_usage_mb"`
	Type        string `json:"type"` // C (Compute), G (Graphics), C+G
}

// BillingSessionRequest for starting a billing session
type BillingSessionRequest struct {
	UserID           string           `json:"user_id"`
	ProviderID       uuid.UUID        `json:"provider_id"`
	JobID            *string          `json:"job_id,omitempty"`
	SessionID        *uuid.UUID       `json:"session_id,omitempty"`
	GPUModel         string           `json:"gpu_model"`
	RequestedVRAM    uint64           `json:"requested_vram_mb"`
	EstimatedPowerW  uint32           `json:"estimated_power_w"`
	MaxHourlyRate    *decimal.Decimal `json:"max_hourly_rate,omitempty"`
	MaxDurationHours *int             `json:"max_duration_hours,omitempty"`
	MaxTotalCost     *decimal.Decimal `json:"max_total_cost,omitempty"`
}

// BillingSessionResponse represents the billing session response
type BillingSessionResponse struct {
	Session struct {
		ID               uuid.UUID       `json:"id"`
		UserID           string          `json:"user_id"`
		ProviderID       uuid.UUID       `json:"provider_id"`
		JobID            *string         `json:"job_id,omitempty"`
		Status           string          `json:"status"`
		GPUModel         string          `json:"gpu_model"`
		AllocatedVRAM    uint64          `json:"allocated_vram_mb"`
		TotalVRAM        uint64          `json:"total_vram_mb"`
		VRAMPercentage   decimal.Decimal `json:"vram_percentage"`
		HourlyRate       decimal.Decimal `json:"hourly_rate"`
		VRAMRate         decimal.Decimal `json:"vram_rate"`
		PowerRate        decimal.Decimal `json:"power_rate"`
		PlatformFeeRate  decimal.Decimal `json:"platform_fee_rate"`
		EstimatedPowerW  uint32          `json:"estimated_power_w"`
		ActualPowerW     *uint32         `json:"actual_power_w,omitempty"`
		StartedAt        time.Time       `json:"started_at"`
		EndedAt          *time.Time      `json:"ended_at,omitempty"`
		LastBilledAt     time.Time       `json:"last_billed_at"`
		TotalCost        decimal.Decimal `json:"total_cost"`
		PlatformFee      decimal.Decimal `json:"platform_fee"`
		ProviderEarnings decimal.Decimal `json:"provider_earnings"`
		CreatedAt        time.Time       `json:"created_at"`
		UpdatedAt        time.Time       `json:"updated_at"`
	} `json:"session"`
	CurrentCost         decimal.Decimal `json:"current_cost"`
	EstimatedHourlyCost decimal.Decimal `json:"estimated_hourly_rate"`
	RemainingBalance    decimal.Decimal `json:"remaining_balance"`
	EstimatedRuntime    decimal.Decimal `json:"estimated_runtime_hours"`
}

// UsageUpdateRequest for sending usage updates to billing
type UsageUpdateRequest struct {
	SessionID       uuid.UUID              `json:"session_id"`
	JobID           string                 `json:"job_id"`
	ProviderID      uuid.UUID              `json:"provider_id"`
	GPUUtilization  uint8                  `json:"gpu_utilization_percent"`
	VRAMUtilization uint8                  `json:"vram_utilization_percent"`
	PowerDraw       uint32                 `json:"power_draw_w"`
	Temperature     uint8                  `json:"temperature_c"`
	CPUUtilization  float64                `json:"cpu_utilization_percent"`
	MemoryUsageMB   uint64                 `json:"memory_usage_mb"`
	EnergyUsageKWh  decimal.Decimal        `json:"energy_usage_kwh"`
	Timestamp       time.Time              `json:"timestamp"`
	CustomMetrics   map[string]interface{} `json:"custom_metrics,omitempty"`
}

// SolanaWalletManager manages Solana wallet operations
type SolanaWalletManager struct {
	privateKey      solana.PrivateKey
	publicKey       solana.PublicKey
	rpcClient       *rpc.Client
	tokenMintPubkey solana.PublicKey
	logger          *zap.Logger
}

// ExecutionEnvironment manages the execution environment for tasks
type ExecutionEnvironment struct {
	dockerClient  *client.Client
	workspaceDir  string
	logger        *zap.Logger
	resourceLimit ResourceLimit
}

// ResourceLimit defines resource limits for task execution
type ResourceLimit struct {
	CPUCores     int
	MemoryMB     uint64
	DiskSpaceMB  uint64
	NetworkKBps  uint64
	GPUMemoryMB  uint64
	MaxProcesses int
	MaxFileDesc  int
}

// GPUProvider represents a comprehensive GPU provider instance
type GPUProvider struct {
	config         *common.ProviderConfig
	logger         *zap.Logger
	httpClient     *http.Client
	natsConn       *nats.Conn
	provider       *common.Provider
	gpus           []common.GPUDetail
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.RWMutex
	isShuttingDown bool

	// Advanced components
	walletManager *SolanaWalletManager
	executionEnv  *ExecutionEnvironment
	activeJobs    map[string]*ActiveJob
	jobMutex      sync.RWMutex

	// Monitoring and metrics
	systemMetrics *SystemMetrics
	alertManager  *AlertManager
	healthChecker *HealthChecker

	// Performance tracking
	performanceHistory []PerformanceSnapshot
	historyMutex       sync.RWMutex

	// Rate limiting and resource management
	resourceManager *ResourceManager
	jobQueue        chan *Task
	workerPool      []*TaskWorker
}

// ActiveJob tracks an active job execution
type ActiveJob struct {
	Task            *Task
	SessionID       uuid.UUID
	ContainerID     string
	WorkspaceDir    string
	StartTime       time.Time
	LastHeartbeat   time.Time
	Context         context.Context
	Cancel          context.CancelFunc
	Status          JobStatus
	Progress        float32
	ResourceUsage   ResourceUsage
	BillingSession  *BillingSessionResponse
	Metrics         ExecutionMetrics
	GPUMetrics      []GPUMetrics
	OutputCollector *OutputCollector
	ErrorCollector  *ErrorCollector
}

// OutputCollector manages stdout/stderr collection
type OutputCollector struct {
	Stdout    strings.Builder
	Stderr    strings.Builder
	LogFile   *os.File
	MaxSizeMB int
	mu        sync.Mutex
}

// ErrorCollector manages error tracking and reporting
type ErrorCollector struct {
	Errors []JobError
	mu     sync.Mutex
}

// JobError represents a job execution error
type JobError struct {
	Timestamp   time.Time `json:"timestamp"`
	Stage       string    `json:"stage"`
	ErrorType   string    `json:"error_type"`
	Message     string    `json:"message"`
	Stack       string    `json:"stack,omitempty"`
	Recoverable bool      `json:"recoverable"`
}

// SystemMetrics tracks comprehensive system metrics
type SystemMetrics struct {
	CPUUsage        float64      `json:"cpu_usage_percent"`
	MemoryUsage     uint64       `json:"memory_usage_mb"`
	MemoryTotal     uint64       `json:"memory_total_mb"`
	DiskUsage       uint64       `json:"disk_usage_mb"`
	DiskTotal       uint64       `json:"disk_total_mb"`
	NetworkTxMB     uint64       `json:"network_tx_mb"`
	NetworkRxMB     uint64       `json:"network_rx_mb"`
	LoadAverage     []float64    `json:"load_average"`
	ProcessCount    int          `json:"process_count"`
	ThreadCount     int          `json:"thread_count"`
	FileDescriptors int          `json:"file_descriptors"`
	Temperature     float64      `json:"temperature_celsius"`
	GPUMetrics      []GPUMetrics `json:"gpu_metrics"`
	LastUpdated     time.Time    `json:"last_updated"`
}

// AlertManager handles alerts and notifications
type AlertManager struct {
	logger        *zap.Logger
	alerts        []Alert
	mu            sync.Mutex
	webhookURL    string
	emailSettings EmailSettings
}

// Alert represents a system alert
type Alert struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Severity   string                 `json:"severity"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details"`
	Timestamp  time.Time              `json:"timestamp"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// EmailSettings for alert notifications
type EmailSettings struct {
	SMTPHost    string
	SMTPPort    int
	Username    string
	Password    string
	FromAddress string
	ToAddresses []string
}

// HealthChecker monitors system health
type HealthChecker struct {
	logger        *zap.Logger
	checks        []HealthCheck
	lastCheckTime time.Time
	overallHealth string
	mu            sync.Mutex
}

// HealthCheck represents a health check
type HealthCheck struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	LastCheck time.Time              `json:"last_check"`
	Duration  time.Duration          `json:"duration"`
	Details   map[string]interface{} `json:"details"`
}

// PerformanceSnapshot captures performance at a point in time
type PerformanceSnapshot struct {
	Timestamp            time.Time       `json:"timestamp"`
	JobsCompleted        int             `json:"jobs_completed"`
	JobsActive           int             `json:"jobs_active"`
	AverageExecutionTime time.Duration   `json:"average_execution_time"`
	CPUUsage             float64         `json:"cpu_usage"`
	MemoryUsage          float64         `json:"memory_usage"`
	GPUUsage             float64         `json:"gpu_usage"`
	EnergyEfficiency     float64         `json:"energy_efficiency"`
	ProviderEarnings     decimal.Decimal `json:"provider_earnings"`
}

// ResourceManager manages resource allocation and limits
type ResourceManager struct {
	maxConcurrentJobs int
	maxCPUUsage       float64
	maxMemoryUsage    float64
	maxGPUUsage       float64
	currentJobs       int
	mu                sync.RWMutex
}

// TaskWorker represents a worker that executes tasks
type TaskWorker struct {
	ID       int
	provider *GPUProvider
	logger   *zap.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// String returns the string representation of JobStatus
func (js JobStatus) String() string {
	return string(js)
}

// getDefaultProviderConfig returns comprehensive default configuration
func getDefaultProviderConfig() *common.ProviderConfig {
	return &common.ProviderConfig{
		ProviderName:        "Advanced GPU Provider",
		OwnerID:             os.Getenv("PROVIDER_OWNER_ID"),
		Location:            getLocationFromEnvironment(),
		APIGatewayURL:       getenvDefault("API_GATEWAY_URL", "http://localhost:8080"),
		ProviderRegistryURL: getenvDefault("PROVIDER_REGISTRY_URL", "http://localhost:8001"),
		BillingServiceURL:   getenvDefault("BILLING_SERVICE_URL", "http://localhost:8003"),
		NATSAddress:         getenvDefault("NATS_ADDRESS", "nats://localhost:4222"),
		SolanaWalletAddress: os.Getenv("SOLANA_WALLET_ADDRESS"),
		MaxConcurrentJobs:   getenvIntDefault("MAX_CONCURRENT_JOBS", 4),
		MinPricePerHour:     getenvDecimalDefault("MIN_PRICE_PER_HOUR", "1.0"),
		EnableDocker:        getenvBoolDefault("ENABLE_DOCKER", true),
		RequestTimeout:      30 * time.Second,
		HeartbeatInterval:   15 * time.Second,
		MetricsInterval:     5 * time.Second,
		WorkspaceDir:        getenvDefault("WORKSPACE_DIR", "/tmp/dante-workspace"),
	}
}

// Helper functions for environment variables
func getenvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getenvIntDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getenvDecimalDefault(key, defaultValue string) decimal.Decimal {
	if value := os.Getenv(key); value != "" {
		if decVal, err := decimal.NewFromString(value); err == nil {
			return decVal
		}
	}
	decVal, _ := decimal.NewFromString(defaultValue)
	return decVal
}

func getenvBoolDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getLocationFromEnvironment() string {
	if location := os.Getenv("PROVIDER_LOCATION"); location != "" {
		return location
	}

	// Try to detect location based on various sources
	if region := os.Getenv("AWS_REGION"); region != "" {
		return fmt.Sprintf("aws-%s", region)
	}
	if zone := os.Getenv("GCP_ZONE"); zone != "" {
		return fmt.Sprintf("gcp-%s", zone)
	}
	if region := os.Getenv("AZURE_REGION"); region != "" {
		return fmt.Sprintf("azure-%s", region)
	}

	// Default to hostname-based location
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}

	return "unknown"
}

func estimatePowerConsumption(gpuName string) uint32 {
	gpuName = strings.ToLower(gpuName)

	// NVIDIA RTX series power consumption estimates
	if strings.Contains(gpuName, "rtx 4090") {
		return 450
	} else if strings.Contains(gpuName, "rtx 4080") {
		return 320
	} else if strings.Contains(gpuName, "rtx 4070") {
		return 200
	} else if strings.Contains(gpuName, "rtx 3090") {
		return 350
	} else if strings.Contains(gpuName, "rtx 3080") {
		return 320
	} else if strings.Contains(gpuName, "rtx 3070") {
		return 220
	}

	// Datacenter cards
	if strings.Contains(gpuName, "a100") {
		return 400
	} else if strings.Contains(gpuName, "h100") {
		return 700
	} else if strings.Contains(gpuName, "v100") {
		return 300
	}

	// AMD cards
	if strings.Contains(gpuName, "6900 xt") {
		return 300
	} else if strings.Contains(gpuName, "6800 xt") {
		return 250
	}

	// Apple Silicon (very efficient)
	if strings.Contains(gpuName, "m3 ultra") {
		return 100
	} else if strings.Contains(gpuName, "m3 max") {
		return 70
	} else if strings.Contains(gpuName, "m2 ultra") {
		return 100
	} else if strings.Contains(gpuName, "m1 ultra") {
		return 90
	}

	return 150 // Conservative default
}

// NewGPUProvider creates a new GPU provider instance with comprehensive capabilities
func NewGPUProvider(config *common.ProviderConfig) (*GPUProvider, error) {
	// Create logger
	logger, err := common.SetupLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %w", err)
	}

	// Detect GPUs
	gpus, err := detectGPUs()
	if err != nil {
		return nil, fmt.Errorf("GPU detection failed: %w", err)
	}

	logger.Info("Detected GPUs", zap.Int("count", len(gpus)))
	for i, gpu := range gpus {
		logger.Info("GPU details",
			zap.Int("index", i),
			zap.String("model", gpu.ModelName),
			zap.Uint64("vram_mb", gpu.VRAM),
			zap.String("architecture", gpu.Architecture),
			zap.Bool("healthy", gpu.IsHealthy),
			zap.Bool("available", gpu.IsAvailable))
	}

	// Create HTTP client with timeouts
	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: false},
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	// Create context for provider lifecycle
	ctx, cancel := context.WithCancel(context.Background())

	// Create provider instance
	now := time.Now()
	providerInstance := &common.Provider{
		ID:         uuid.New(),
		OwnerID:    config.OwnerID,
		Name:       config.ProviderName,
		Location:   config.Location,
		Status:     "initializing",
		GPUs:       gpus,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastSeenAt: &now,
		Metadata: common.ProviderMetadata{
			MaxConcurrentJobs: config.MaxConcurrentJobs,
			MinPricePerHour:   config.MinPricePerHour,
			SolanaWallet:      config.SolanaWalletAddress,
			DockerEnabled:     config.EnableDocker,
		},
	}

	// Initialize Solana wallet manager
	walletManager, err := initializeSolanaWallet(config, logger)
	if err != nil {
		logger.Warn("Failed to initialize Solana wallet, continuing without it", zap.Error(err))
	}

	// Initialize execution environment
	executionEnv, err := initializeExecutionEnvironment(config, logger)
	if err != nil {
		logger.Error("Failed to initialize execution environment", zap.Error(err))
		cancel()
		return nil, fmt.Errorf("execution environment initialization failed: %w", err)
	}

	// Create resource manager
	resourceManager := &ResourceManager{
		maxConcurrentJobs: config.MaxConcurrentJobs,
		maxCPUUsage:       80.0,
		maxMemoryUsage:    85.0,
		maxGPUUsage:       90.0,
	}

	// Create alert manager
	alertManager := &AlertManager{
		logger: logger,
		alerts: make([]Alert, 0),
	}

	// Create health checker
	healthChecker := &HealthChecker{
		logger:        logger,
		checks:        make([]HealthCheck, 0),
		overallHealth: "unknown",
	}

	provider := &GPUProvider{
		config:             config,
		logger:             logger,
		httpClient:         httpClient,
		provider:           providerInstance,
		gpus:               gpus,
		ctx:                ctx,
		cancel:             cancel,
		activeJobs:         make(map[string]*ActiveJob),
		walletManager:      walletManager,
		executionEnv:       executionEnv,
		systemMetrics:      &SystemMetrics{},
		alertManager:       alertManager,
		healthChecker:      healthChecker,
		performanceHistory: make([]PerformanceSnapshot, 0, 1000),
		resourceManager:    resourceManager,
		jobQueue:           make(chan *Task, 100),
	}

	// Initialize worker pool
	provider.initializeWorkerPool()

	return provider, nil
}

// initializeSolanaWallet initializes the Solana wallet manager
func initializeSolanaWallet(config *common.ProviderConfig, logger *zap.Logger) (*SolanaWalletManager, error) {
	if config.SolanaWalletAddress == "" {
		return nil, fmt.Errorf("Solana wallet address not configured")
	}

	// Load private key from environment or generate one
	var privateKey solana.PrivateKey
	var err error

	if privKeyStr := os.Getenv("SOLANA_PRIVATE_KEY"); privKeyStr != "" {
		// Decode base58 private key
		privKeyBytes, err := solana.PrivateKeyFromBase58(privKeyStr)
		if err != nil {
			return nil, fmt.Errorf("invalid Solana private key: %w", err)
		}
		privateKey = privKeyBytes
	} else {
		// Generate new keypair
		account := solana.NewWallet()
		privateKey = account.PrivateKey
		logger.Warn("Generated new Solana keypair - save this private key",
			zap.String("public_key", account.PublicKey().String()),
			zap.String("private_key", account.PrivateKey.String()))
	}

	publicKey := privateKey.PublicKey()

	// Create RPC client
	rpcEndpoint := getenvDefault("SOLANA_RPC_URL", "https://api.mainnet-beta.solana.com")
	rpcClient := rpc.New(rpcEndpoint)

	// Get dGPU token mint address
	dGPUTokenMint := getenvDefault("DGPU_TOKEN_MINT", "7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump")
	tokenMintPubkey, err := solana.PublicKeyFromBase58(dGPUTokenMint)
	if err != nil {
		return nil, fmt.Errorf("invalid dGPU token mint address: %w", err)
	}

	walletManager := &SolanaWalletManager{
		privateKey:      privateKey,
		publicKey:       publicKey,
		rpcClient:       rpcClient,
		tokenMintPubkey: tokenMintPubkey,
		logger:          logger,
	}

	// Test connection
	if err := walletManager.testConnection(); err != nil {
		return nil, fmt.Errorf("Solana wallet connection test failed: %w", err)
	}

	logger.Info("Solana wallet initialized successfully",
		zap.String("public_key", publicKey.String()))

	return walletManager, nil
}

// testConnection tests the Solana RPC connection
func (swm *SolanaWalletManager) testConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get slot to test connection
	_, err := swm.rpcClient.GetSlot(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("failed to connect to Solana RPC: %w", err)
	}

	// Get account balance
	balance, err := swm.rpcClient.GetBalance(ctx, swm.publicKey, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("failed to get wallet balance: %w", err)
	}

	swm.logger.Info("Solana wallet balance",
		zap.Uint64("lamports", balance.Value),
		zap.Float64("sol", float64(balance.Value)/1e9))

	return nil
}

// initializeExecutionEnvironment sets up the Docker and execution environment
func initializeExecutionEnvironment(config *common.ProviderConfig, logger *zap.Logger) (*ExecutionEnvironment, error) {
	// Create workspace directory
	workspaceDir := config.WorkspaceDir
	if workspaceDir == "" {
		workspaceDir = "/tmp/dante-workspace"
	}

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Initialize Docker client if enabled
	var dockerClient *client.Client
	var err error

	if config.EnableDocker {
		dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			logger.Warn("Failed to initialize Docker client", zap.Error(err))
		} else {
			// Test Docker connection
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if _, err := dockerClient.Ping(ctx); err != nil {
				logger.Warn("Docker ping failed", zap.Error(err))
				dockerClient = nil
			} else {
				logger.Info("Docker client initialized successfully")
			}
		}
	}

	// Set resource limits based on system capabilities
	resourceLimit := ResourceLimit{
		CPUCores:     runtime.NumCPU(),
		MemoryMB:     8192,    // Default 8GB limit
		DiskSpaceMB:  10240,   // Default 10GB limit
		NetworkKBps:  1000000, // 1GB/s
		GPUMemoryMB:  0,       // Set based on available GPU memory
		MaxProcesses: 100,
		MaxFileDesc:  1000,
	}

	// Get system memory to set realistic limits
	if memInfo, err := mem.VirtualMemory(); err == nil {
		// Use up to 80% of system memory
		resourceLimit.MemoryMB = uint64(float64(memInfo.Total) * 0.8 / 1024 / 1024)
	}

	execEnv := &ExecutionEnvironment{
		dockerClient:  dockerClient,
		workspaceDir:  workspaceDir,
		logger:        logger,
		resourceLimit: resourceLimit,
	}

	return execEnv, nil
}

// initializeWorkerPool creates worker goroutines for task execution
func (p *GPUProvider) initializeWorkerPool() {
	workerCount := p.config.MaxConcurrentJobs
	p.workerPool = make([]*TaskWorker, workerCount)

	for i := 0; i < workerCount; i++ {
		ctx, cancel := context.WithCancel(p.ctx)
		worker := &TaskWorker{
			ID:       i,
			provider: p,
			logger:   p.logger.With(zap.Int("worker_id", i)),
			ctx:      ctx,
			cancel:   cancel,
		}
		p.workerPool[i] = worker

		// Start worker goroutine
		p.wg.Add(1)
		go worker.run()
	}

	p.logger.Info("Worker pool initialized", zap.Int("workers", workerCount))
}

// run is the main worker loop
func (w *TaskWorker) run() {
	defer w.provider.wg.Done()
	w.logger.Info("Worker started")

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("Worker stopping")
			return
		case task, ok := <-w.provider.jobQueue:
			if !ok {
				return
			}
			w.executeTask(task)
		}
	}
}

// executeTask executes a task with comprehensive monitoring
func (w *TaskWorker) executeTask(task *Task) {
	w.logger.Info("Executing task",
		zap.String("job_id", task.JobID),
		zap.String("user_id", task.UserID),
		zap.String("type", string(task.ExecutionType)))

	// Create active job tracking
	activeJob := &ActiveJob{
		Task:            task,
		SessionID:       task.SessionID,
		StartTime:       time.Now(),
		LastHeartbeat:   time.Now(),
		Status:          JobStatusStarting,
		Progress:        0.0,
		OutputCollector: &OutputCollector{MaxSizeMB: 100},
		ErrorCollector:  &ErrorCollector{Errors: make([]JobError, 0)},
	}

	ctx, cancel := context.WithTimeout(w.ctx, time.Duration(task.MaxDurationMinutes)*time.Minute)
	activeJob.Context = ctx
	activeJob.Cancel = cancel
	defer cancel()

	// Track active job
	w.provider.jobMutex.Lock()
	w.provider.activeJobs[task.JobID] = activeJob
	w.provider.jobMutex.Unlock()

	defer func() {
		w.provider.jobMutex.Lock()
		delete(w.provider.activeJobs, task.JobID)
		w.provider.jobMutex.Unlock()
	}()

	// Create workspace for this job
	jobWorkspace := filepath.Join(w.provider.executionEnv.workspaceDir, task.JobID)
	if err := os.MkdirAll(jobWorkspace, 0755); err != nil {
		w.handleTaskError(activeJob, "workspace_creation", err)
		return
	}
	activeJob.WorkspaceDir = jobWorkspace

	// Start billing session
	if err := w.startBillingSession(activeJob); err != nil {
		w.handleTaskError(activeJob, "billing_start", err)
		return
	}

	// Download input files
	if err := w.downloadInputFiles(activeJob); err != nil {
		w.handleTaskError(activeJob, "input_download", err)
		return
	}

	// Update status to running
	activeJob.Status = JobStatusRunning
	w.publishTaskStatus(activeJob, "Task execution started", "")

	// Start metrics collection
	go w.collectMetrics(activeJob)

	// Execute based on execution type
	var err error

	switch task.ExecutionType {
	case ExecutionTypeDocker:
		_, err = w.executeDockerTask(activeJob)
	case ExecutionTypeScript:
		_, err = w.executeScriptTask(activeJob)
	default:
		err = fmt.Errorf("unsupported execution type: %s", task.ExecutionType)
	}

	// Handle execution result
	if err != nil {
		w.handleTaskError(activeJob, "execution", err)
		return
	}

	// Upload output files
	if err := w.uploadOutputFiles(activeJob); err != nil {
		w.logger.Warn("Failed to upload output files", zap.Error(err))
		// Don't fail the task for output upload errors
	}

	// Finalize task
	activeJob.Status = JobStatusCompleted
	activeJob.Progress = 1.0
	w.publishTaskStatus(activeJob, "Task completed successfully", "")

	// End billing session
	if err := w.endBillingSession(activeJob); err != nil {
		w.logger.Error("Failed to end billing session", zap.Error(err))
	}

	// Cleanup workspace if requested
	if task.WorkspaceCleanup {
		if err := os.RemoveAll(jobWorkspace); err != nil {
			w.logger.Warn("Failed to cleanup workspace", zap.Error(err))
		}
	}

	w.logger.Info("Task completed successfully", zap.String("job_id", task.JobID))
}

// executeDockerTask executes a task using Docker
func (w *TaskWorker) executeDockerTask(activeJob *ActiveJob) (*TaskResult, error) {
	task := activeJob.Task

	if w.provider.executionEnv.dockerClient == nil {
		return nil, fmt.Errorf("Docker not available")
	}

	// Pull Docker image
	w.publishTaskStatus(activeJob, "Pulling Docker image", "")
	if err := w.pullDockerImage(task.DockerImage); err != nil {
		return nil, fmt.Errorf("failed to pull Docker image: %w", err)
	}

	// Prepare container configuration
	containerConfig := &container.Config{
		Image:        task.DockerImage,
		Cmd:          task.DockerCommand,
		Env:          mapToSlice(task.DockerEnvironment),
		WorkingDir:   "/workspace",
		AttachStdout: true,
		AttachStderr: true,
	}

	// Add GPU access if requested and available
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/workspace", activeJob.WorkspaceDir),
		},
		Resources: container.Resources{
			Memory:   int64(w.provider.executionEnv.resourceLimit.MemoryMB * 1024 * 1024),
			NanoCPUs: int64(w.provider.executionEnv.resourceLimit.CPUCores) * 1000000000,
		},
		NetworkMode: "bridge",
	}

	if task.DockerGPUAccess && w.hasAvailableGPU() {
		hostConfig.DeviceRequests = []container.DeviceRequest{
			{
				Driver:       "nvidia",
				Count:        -1, // All GPUs
				Capabilities: [][]string{{"gpu"}},
			},
		}
	}

	// Add custom volumes
	for _, volume := range task.DockerVolumes {
		hostConfig.Binds = append(hostConfig.Binds,
			fmt.Sprintf("%s:%s", volume.Source, volume.Target))
	}

	// Create and start container
	ctx := activeJob.Context
	resp, err := w.provider.executionEnv.dockerClient.ContainerCreate(
		ctx, containerConfig, hostConfig, &network.NetworkingConfig{}, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	activeJob.ContainerID = resp.ID
	defer w.cleanupContainer(resp.ID)

	// Start container
	if err := w.provider.executionEnv.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	w.publishTaskStatus(activeJob, "Container started", "")

	// Attach to container to collect logs
	go w.collectContainerLogs(activeJob, resp.ID)

	// Wait for container to finish
	statusCh, errCh := w.provider.executionEnv.dockerClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("container wait error: %w", err)
		}
	case status := <-statusCh:
		// Container finished
		result := &TaskResult{
			Success:  status.StatusCode == 0,
			ExitCode: int(status.StatusCode),
			Output:   activeJob.OutputCollector.Stdout.String(),
		}

		if status.StatusCode != 0 {
			result.Error = activeJob.OutputCollector.Stderr.String()
		}

		return result, nil
	case <-ctx.Done():
		// Timeout or cancellation
		return nil, fmt.Errorf("task execution timeout or cancelled")
	}

	return nil, fmt.Errorf("unexpected container execution end")
}

// executeScriptTask executes a script-based task
func (w *TaskWorker) executeScriptTask(activeJob *ActiveJob) (*TaskResult, error) {
	task := activeJob.Task

	if task.Script == "" {
		return nil, fmt.Errorf("no script provided")
	}

	// Determine script interpreter
	var interpreter string
	var scriptExt string

	switch task.ExecutionType {
	case ExecutionTypePython:
		interpreter = "python3"
		scriptExt = ".py"
	case ExecutionTypeBash:
		interpreter = "bash"
		scriptExt = ".sh"
	default:
		// Try to detect from script language
		switch strings.ToLower(task.ScriptLanguage) {
		case "python", "python3":
			interpreter = "python3"
			scriptExt = ".py"
		case "bash", "shell", "sh":
			interpreter = "bash"
			scriptExt = ".sh"
		case "node", "nodejs", "javascript":
			interpreter = "node"
			scriptExt = ".js"
		case "ruby":
			interpreter = "ruby"
			scriptExt = ".rb"
		default:
			interpreter = "bash"
			scriptExt = ".sh"
		}
	}

	// Check if interpreter is available
	if _, err := exec.LookPath(interpreter); err != nil {
		return nil, fmt.Errorf("interpreter %s not found", interpreter)
	}

	// Write script to file
	scriptPath := filepath.Join(activeJob.WorkspaceDir, "script"+scriptExt)
	if err := os.WriteFile(scriptPath, []byte(task.Script), 0755); err != nil {
		return nil, fmt.Errorf("failed to write script file: %w", err)
	}

	// Prepare execution environment
	cmd := exec.CommandContext(activeJob.Context, interpreter, scriptPath)
	cmd.Dir = activeJob.WorkspaceDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range task.ScriptEnvironment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set up stdout/stderr capture
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	w.publishTaskStatus(activeJob, "Starting script execution", "")

	// Run the script
	err := cmd.Run()

	// Prepare result
	result := &TaskResult{
		Success:  err == nil,
		Output:   stdout.String(),
		Error:    stderr.String(),
		ExitCode: 0,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	// Update output collectors
	activeJob.OutputCollector.Stdout.WriteString(result.Output)
	activeJob.OutputCollector.Stderr.WriteString(result.Error)

	return result, nil
}

// handleTaskError handles task execution errors
func (w *TaskWorker) handleTaskError(activeJob *ActiveJob, stage string, err error) {
	w.logger.Error("Task execution error",
		zap.String("job_id", activeJob.Task.JobID),
		zap.String("stage", stage),
		zap.Error(err))

	// Add error to collector
	jobError := JobError{
		Timestamp:   time.Now(),
		Stage:       stage,
		ErrorType:   "execution_error",
		Message:     err.Error(),
		Recoverable: false,
	}

	activeJob.ErrorCollector.mu.Lock()
	activeJob.ErrorCollector.Errors = append(activeJob.ErrorCollector.Errors, jobError)
	activeJob.ErrorCollector.mu.Unlock()

	// Update status
	activeJob.Status = JobStatusFailed
	w.publishTaskStatus(activeJob, fmt.Sprintf("Task failed at %s", stage), err.Error())

	// End billing session if it was started
	if activeJob.BillingSession != nil {
		if endErr := w.endBillingSession(activeJob); endErr != nil {
			w.logger.Error("Failed to end billing session after error", zap.Error(endErr))
		}
	}
}

// publishTaskStatus publishes task status updates via NATS
func (w *TaskWorker) publishTaskStatus(activeJob *ActiveJob, message, errorMsg string) {
	if w.provider.natsConn == nil {
		return
	}

	update := TaskStatusUpdate{
		JobID:           activeJob.Task.JobID,
		ProviderID:      w.provider.provider.ID.String(),
		SessionID:       activeJob.SessionID,
		Status:          activeJob.Status,
		Progress:        activeJob.Progress,
		Stage:           activeJob.Status.String(),
		Message:         message,
		Error:           errorMsg,
		Metrics:         activeJob.Metrics,
		Timestamp:       time.Now(),
		ResourceUsage:   activeJob.ResourceUsage,
		GPUMetrics:      activeJob.GPUMetrics,
		StartedAt:       &activeJob.StartTime,
		DurationSeconds: time.Since(activeJob.StartTime).Seconds(),
	}

	if activeJob.BillingSession != nil {
		update.ActualCostDGPU = activeJob.BillingSession.CurrentCost
	}

	if data, err := json.Marshal(update); err == nil {
		subject := fmt.Sprintf("task.status.%s", activeJob.Task.JobID)
		w.provider.natsConn.Publish(subject, data)
	}
}

// startBillingSession starts a billing session for the task
func (w *TaskWorker) startBillingSession(activeJob *ActiveJob) error {
	if w.provider.config.BillingServiceURL == "" {
		w.logger.Warn("Billing service URL not configured, skipping billing")
		return nil
	}

	task := activeJob.Task

	// Find appropriate GPU for the task
	var selectedGPU *common.GPUDetail
	for _, gpu := range w.provider.gpus {
		if gpu.IsAvailable && gpu.IsHealthy {
			if task.Requirements.GPUModel == "" ||
				strings.Contains(strings.ToLower(gpu.ModelName), strings.ToLower(task.Requirements.GPUModel)) {
				if gpu.VRAM >= task.Requirements.GPUMemoryMB {
					selectedGPU = &gpu
					break
				}
			}
		}
	}

	if selectedGPU == nil {
		return fmt.Errorf("no suitable GPU available for task requirements")
	}

	// Create billing session request
	request := BillingSessionRequest{
		UserID:          task.UserID,
		ProviderID:      w.provider.provider.ID,
		JobID:           &task.JobID,
		SessionID:       &activeJob.SessionID,
		GPUModel:        selectedGPU.ModelName,
		RequestedVRAM:   task.Requirements.GPUMemoryMB,
		EstimatedPowerW: selectedGPU.PowerConsumption,
		MaxTotalCost:    &task.MaxCostDGPU,
	}

	// Send request to billing service
	reqData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal billing request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/billing/sessions/start", w.provider.config.BillingServiceURL)
	resp, err := w.provider.httpClient.Post(url, "application/json", bytes.NewBuffer(reqData))
	if err != nil {
		return fmt.Errorf("failed to start billing session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("billing service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var billingResp BillingSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&billingResp); err != nil {
		return fmt.Errorf("failed to decode billing response: %w", err)
	}

	activeJob.BillingSession = &billingResp
	w.logger.Info("Billing session started",
		zap.String("session_id", billingResp.Session.ID.String()),
		zap.String("hourly_rate", billingResp.Session.HourlyRate.String()))

	return nil
}

// endBillingSession ends the billing session
func (w *TaskWorker) endBillingSession(activeJob *ActiveJob) error {
	if activeJob.BillingSession == nil {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/billing/sessions/%s/end",
		w.provider.config.BillingServiceURL,
		activeJob.BillingSession.Session.ID.String())

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create billing end request: %w", err)
	}

	resp, err := w.provider.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to end billing session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		w.logger.Error("Failed to end billing session",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)))
	}

	return nil
}

// downloadInputFiles downloads input files for the task
func (w *TaskWorker) downloadInputFiles(activeJob *ActiveJob) error {
	if len(activeJob.Task.InputFiles) == 0 {
		return nil
	}

	w.publishTaskStatus(activeJob, "Downloading input files", "")

	for i, file := range activeJob.Task.InputFiles {
		w.logger.Info("Downloading input file",
			zap.Int("index", i),
			zap.String("url", file.URL),
			zap.String("path", file.Path))

		if err := w.downloadFile(file, activeJob.WorkspaceDir); err != nil {
			return fmt.Errorf("failed to download file %s: %w", file.URL, err)
		}

		// Update progress
		activeJob.Progress = float32(i+1) / float32(len(activeJob.Task.InputFiles)) * 0.2 // 20% of total progress
		w.publishTaskStatus(activeJob, fmt.Sprintf("Downloaded %d/%d input files", i+1, len(activeJob.Task.InputFiles)), "")
	}

	return nil
}

// downloadFile downloads a single file
func (w *TaskWorker) downloadFile(file FileTransfer, workspaceDir string) error {
	// Create HTTP request
	req, err := http.NewRequest("GET", file.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers
	for key, value := range file.Headers {
		req.Header.Set(key, value)
	}

	// Perform request
	resp, err := w.provider.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create destination file
	destPath := filepath.Join(workspaceDir, file.Path)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy data
	_, err = io.Copy(destFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// uploadOutputFiles uploads output files
func (w *TaskWorker) uploadOutputFiles(activeJob *ActiveJob) error {
	if len(activeJob.Task.OutputFiles) == 0 {
		return nil
	}

	w.publishTaskStatus(activeJob, "Uploading output files", "")

	for i, file := range activeJob.Task.OutputFiles {
		sourcePath := filepath.Join(activeJob.WorkspaceDir, file.Path)

		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			w.logger.Warn("Output file not found", zap.String("path", sourcePath))
			continue
		}

		if err := w.uploadFile(file, sourcePath); err != nil {
			w.logger.Error("Failed to upload output file",
				zap.String("path", file.Path),
				zap.Error(err))
			// Continue with other files
		} else {
			w.logger.Info("Uploaded output file",
				zap.Int("index", i),
				zap.String("path", file.Path))
		}
	}

	return nil
}

// uploadFile uploads a single file
func (w *TaskWorker) uploadFile(file FileTransfer, sourcePath string) error {
	// This is a simplified implementation
	// In a real system, you'd upload to a storage service like S3, Google Cloud Storage, etc.

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create upload request
	req, err := http.NewRequest("PUT", file.URL, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	// Add headers
	for key, value := range file.Headers {
		req.Header.Set(key, value)
	}

	// Perform upload
	resp, err := w.provider.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	return nil
}

// collectMetrics collects system and GPU metrics during task execution
func (w *TaskWorker) collectMetrics(activeJob *ActiveJob) {
	ticker := time.NewTicker(w.provider.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-activeJob.Context.Done():
			return
		case <-ticker.C:
			// Collect CPU metrics
			if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
				activeJob.ResourceUsage.CPUPercent = cpuPercent[0]
			}

			// Collect memory metrics
			if memInfo, err := mem.VirtualMemory(); err == nil {
				activeJob.ResourceUsage.MemoryMB = (memInfo.Total - memInfo.Available) / 1024 / 1024
				activeJob.ResourceUsage.MemoryPercent = memInfo.UsedPercent
			}

			// Collect GPU metrics
			if gpuMetrics, err := w.collectGPUMetrics(); err == nil {
				activeJob.GPUMetrics = gpuMetrics
			}

			// Update timestamp
			activeJob.ResourceUsage.Timestamp = time.Now()
			activeJob.LastHeartbeat = time.Now()

			// Send usage update to billing service
			if activeJob.BillingSession != nil {
				w.sendUsageUpdate(activeJob)
			}
		}
	}
}

// collectGPUMetrics collects current GPU metrics
func (w *TaskWorker) collectGPUMetrics() ([]GPUMetrics, error) {
	var metrics []GPUMetrics

	// Try NVIDIA first
	if nvidiaMetrics, err := w.collectNVIDIAMetrics(); err == nil {
		metrics = append(metrics, nvidiaMetrics...)
	}

	// Add other GPU vendors as needed
	// TODO: Implement AMD, Intel, Apple metrics collection

	return metrics, nil
}

// collectNVIDIAMetrics collects NVIDIA GPU metrics
func (w *TaskWorker) collectNVIDIAMetrics() ([]GPUMetrics, error) {
	if !isCommandAvailable("nvidia-smi") {
		return nil, fmt.Errorf("nvidia-smi not available")
	}

	cmd := exec.Command("nvidia-smi",
		"--query-gpu=index,uuid,name,utilization.gpu,utilization.memory,memory.total,memory.used,memory.free,temperature.gpu,power.draw,clocks.gr,clocks.mem,fan.speed",
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi execution failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var metrics []GPUMetrics

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 13 {
			continue
		}

		index, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		uuid := strings.TrimSpace(parts[1])
		name := strings.TrimSpace(parts[2])
		utilizationGPU, _ := strconv.ParseUint(strings.TrimSpace(parts[3]), 10, 8)
		utilizationMemory, _ := strconv.ParseUint(strings.TrimSpace(parts[4]), 10, 8)
		memoryTotal, _ := strconv.ParseUint(strings.TrimSpace(parts[5]), 10, 64)
		memoryUsed, _ := strconv.ParseUint(strings.TrimSpace(parts[6]), 10, 64)
		memoryFree, _ := strconv.ParseUint(strings.TrimSpace(parts[7]), 10, 64)
		temperature, _ := strconv.ParseUint(strings.TrimSpace(parts[8]), 10, 8)
		powerDraw, _ := strconv.ParseUint(strings.TrimSpace(parts[9]), 10, 32)
		clockCore, _ := strconv.ParseUint(strings.TrimSpace(parts[10]), 10, 32)
		clockMemory, _ := strconv.ParseUint(strings.TrimSpace(parts[11]), 10, 32)
		fanSpeed, _ := strconv.ParseUint(strings.TrimSpace(parts[12]), 10, 8)

		metric := GPUMetrics{
			Index:             index,
			UUID:              uuid,
			Name:              name,
			UtilizationGPU:    uint8(utilizationGPU),
			UtilizationMemory: uint8(utilizationMemory),
			MemoryTotal:       memoryTotal,
			MemoryUsed:        memoryUsed,
			MemoryFree:        memoryFree,
			Temperature:       uint8(temperature),
			PowerDraw:         uint32(powerDraw),
			ClockCore:         uint32(clockCore),
			ClockMemory:       uint32(clockMemory),
			FanSpeed:          uint8(fanSpeed),
			Timestamp:         time.Now(),
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// sendUsageUpdate sends usage update to billing service
func (w *TaskWorker) sendUsageUpdate(activeJob *ActiveJob) {
	if w.provider.config.BillingServiceURL == "" || activeJob.BillingSession == nil {
		return
	}

	// Calculate energy usage (simplified)
	var energyUsage decimal.Decimal
	if len(activeJob.GPUMetrics) > 0 {
		totalPower := decimal.Zero
		for _, gpu := range activeJob.GPUMetrics {
			totalPower = totalPower.Add(decimal.NewFromInt(int64(gpu.PowerDraw)))
		}
		// Convert watts to kWh (power in watts * time in hours / 1000)
		hours := decimal.NewFromFloat(w.provider.config.MetricsInterval.Hours())
		energyUsage = totalPower.Mul(hours).Div(decimal.NewFromInt(1000))
	}

	request := UsageUpdateRequest{
		SessionID:      activeJob.BillingSession.Session.ID,
		JobID:          activeJob.Task.JobID,
		ProviderID:     w.provider.provider.ID,
		CPUUtilization: activeJob.ResourceUsage.CPUPercent,
		MemoryUsageMB:  activeJob.ResourceUsage.MemoryMB,
		EnergyUsageKWh: energyUsage,
		Timestamp:      time.Now(),
	}

	// Add GPU metrics
	if len(activeJob.GPUMetrics) > 0 {
		gpu := activeJob.GPUMetrics[0] // Use first GPU for simplicity
		request.GPUUtilization = gpu.UtilizationGPU
		request.VRAMUtilization = gpu.UtilizationMemory
		request.PowerDraw = gpu.PowerDraw
		request.Temperature = gpu.Temperature
	}

	// Send update
	reqData, err := json.Marshal(request)
	if err != nil {
		w.logger.Error("Failed to marshal usage update", zap.Error(err))
		return
	}

	url := fmt.Sprintf("%s/api/v1/billing/sessions/%s/usage",
		w.provider.config.BillingServiceURL,
		activeJob.BillingSession.Session.ID.String())

	resp, err := w.provider.httpClient.Post(url, "application/json", bytes.NewBuffer(reqData))
	if err != nil {
		w.logger.Error("Failed to send usage update", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.logger.Warn("Usage update returned non-OK status", zap.Int("status", resp.StatusCode))
	}
}

// pullDockerImage pulls a Docker image
func (w *TaskWorker) pullDockerImage(image string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	reader, err := w.provider.executionEnv.dockerClient.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Read the pull output (optional, for logging)
	_, err = io.Copy(io.Discard, reader)
	return err
}

// hasAvailableGPU checks if there's an available GPU
func (w *TaskWorker) hasAvailableGPU() bool {
	for _, gpu := range w.provider.gpus {
		if gpu.IsAvailable && gpu.IsHealthy {
			return true
		}
	}
	return false
}

// collectContainerLogs collects logs from a Docker container
func (w *TaskWorker) collectContainerLogs(activeJob *ActiveJob, containerID string) {
	ctx := activeJob.Context

	logs, err := w.provider.executionEnv.dockerClient.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	})
	if err != nil {
		w.logger.Error("Failed to get container logs", zap.Error(err))
		return
	}
	defer logs.Close()

	// Read logs and add to output collector
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := logs.Read(buf)
			if err != nil {
				if err != io.EOF {
					w.logger.Error("Error reading container logs", zap.Error(err))
				}
				return
			}

			if n > 0 {
				// Docker logs have 8-byte header, strip it
				logData := buf[8:n]

				activeJob.OutputCollector.mu.Lock()
				activeJob.OutputCollector.Stdout.Write(logData)
				activeJob.OutputCollector.mu.Unlock()
			}
		}
	}
}

// cleanupContainer removes a Docker container
func (w *TaskWorker) cleanupContainer(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop container
	timeout := 10
	w.provider.executionEnv.dockerClient.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})

	// Remove container
	w.provider.executionEnv.dockerClient.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force: true,
	})
}

// mapToSlice converts a map to a slice of key=value strings
func mapToSlice(m map[string]string) []string {
	result := make([]string, 0, len(m))
	for k, v := range m {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

// Main function
func main() {
	// Load configuration
	config := getDefaultProviderConfig()

	// Create provider
	provider, err := NewGPUProvider(config)
	if err != nil {
		fmt.Printf("Failed to create GPU provider: %v\n", err)
		os.Exit(1)
	}

	// Initialize provider
	if err := provider.Initialize(); err != nil {
		fmt.Printf("Failed to initialize GPU provider: %v\n", err)
		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	provider.logger.Info("GPU Provider started successfully")
	fmt.Println("GPU Provider is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	<-sigChan

	// Graceful shutdown
	if err := provider.Shutdown(); err != nil {
		provider.logger.Error("Error during shutdown", zap.Error(err))
		os.Exit(1)
	}

	fmt.Println("GPU Provider stopped")
}

// isCommandAvailable checks if a command exists in PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// detectGPUs detects available GPUs on the system
func detectGPUs() ([]common.GPUDetail, error) {
	var gpus []common.GPUDetail

	// Detect NVIDIA GPUs
	if isCommandAvailable("nvidia-smi") {
		nvidiaGPUs, err := detectNVIDIAGPUs()
		if err == nil {
			gpus = append(gpus, nvidiaGPUs...)
		}
	}

	// Detect AMD GPUs
	if runtime.GOOS == "linux" {
		amdGPUs, err := detectAMDGPUs()
		if err == nil {
			gpus = append(gpus, amdGPUs...)
		}
	}

	// Detect Intel GPUs
	if isCommandAvailable("intel_gpu_top") {
		intelGPUs, err := detectIntelGPUs()
		if err == nil {
			gpus = append(gpus, intelGPUs...)
		}
	}

	// Detect Apple Silicon GPUs
	if runtime.GOOS == "darwin" {
		appleGPUs, err := detectAppleGPUs()
		if err == nil {
			gpus = append(gpus, appleGPUs...)
		}
	}

	if len(gpus) == 0 {
		return nil, fmt.Errorf("no GPUs detected")
	}

	return gpus, nil
}

// detectNVIDIAGPUs detects NVIDIA GPUs using nvidia-smi
func detectNVIDIAGPUs() ([]common.GPUDetail, error) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,memory.total,driver_version,compute_cap", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi command failed: %w", err)
	}

	var gpus []common.GPUDetail
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 5 {
			continue
		}

		// Parse memory
		memoryStr := strings.TrimSpace(fields[2])
		memoryMB, err := strconv.ParseUint(memoryStr, 10, 64)
		if err != nil {
			memoryMB = 0
		}

		gpu := common.GPUDetail{
			ModelName:         strings.TrimSpace(fields[1]),
			VRAM:              memoryMB,
			DriverVersion:     strings.TrimSpace(fields[3]),
			ComputeCapability: strings.TrimSpace(fields[4]),
			IsHealthy:         true,
			IsAvailable:       true,
			LastCheckAt:       time.Now(),
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// detectAMDGPUs detects AMD GPUs (Linux only)
func detectAMDGPUs() ([]common.GPUDetail, error) {
	var gpus []common.GPUDetail

	// Check for ROCm devices
	devices, err := filepath.Glob("/sys/class/drm/card*/device/vendor")
	if err != nil {
		return gpus, nil
	}

	for _, device := range devices {
		vendorBytes, err := os.ReadFile(device)
		if err != nil {
			continue
		}

		vendor := strings.TrimSpace(string(vendorBytes))
		if vendor == "0x1002" { // AMD vendor ID
			// This is a basic AMD GPU detection
			gpu := common.GPUDetail{
				ModelName:   "AMD GPU",
				VRAM:        8192, // Default assumption
				IsHealthy:   true,
				IsAvailable: true,
				LastCheckAt: time.Now(),
			}
			gpus = append(gpus, gpu)
		}
	}

	return gpus, nil
}

// detectIntelGPUs detects Intel GPUs
func detectIntelGPUs() ([]common.GPUDetail, error) {
	var gpus []common.GPUDetail

	// Basic Intel GPU detection
	if runtime.GOOS == "linux" {
		devices, err := filepath.Glob("/sys/class/drm/card*/device/vendor")
		if err != nil {
			return gpus, nil
		}

		for _, device := range devices {
			vendorBytes, err := os.ReadFile(device)
			if err != nil {
				continue
			}

			vendor := strings.TrimSpace(string(vendorBytes))
			if vendor == "0x8086" { // Intel vendor ID
				gpu := common.GPUDetail{
					ModelName:   "Intel GPU",
					VRAM:        4096, // Default assumption
					IsHealthy:   true,
					IsAvailable: true,
					LastCheckAt: time.Now(),
				}
				gpus = append(gpus, gpu)
			}
		}
	}

	return gpus, nil
}

// detectAppleGPUs detects Apple Silicon GPUs (macOS only)
func detectAppleGPUs() ([]common.GPUDetail, error) {
	var gpus []common.GPUDetail

	if runtime.GOOS != "darwin" {
		return gpus, nil
	}

	cmd := exec.Command("system_profiler", "SPDisplaysDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return gpus, nil
	}

	// Parse system_profiler output for GPU info
	// This is a simplified implementation
	if strings.Contains(string(output), "Apple") {
		gpu := common.GPUDetail{
			ModelName:   "Apple Silicon GPU",
			VRAM:        8192, // Unified memory assumption
			IsHealthy:   true,
			IsAvailable: true,
			LastCheckAt: time.Now(),
		}
		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// Initialize sets up the GPU provider
func (p *GPUProvider) Initialize() error {
	p.logger.Info("Initializing GPU provider", zap.String("provider_id", p.provider.ID.String()))

	// Initialize job queue
	p.jobQueue = make(chan *Task, 100)

	// Initialize worker pool
	p.initializeWorkerPool()

	// Start background services
	go p.startHeartbeat()
	go p.startMetricsCollection()
	go p.startHealthChecks()

	p.logger.Info("GPU provider initialized successfully")
	return nil
}

// Shutdown gracefully shuts down the GPU provider
func (p *GPUProvider) Shutdown() error {
	p.logger.Info("Shutting down GPU provider")

	p.mu.Lock()
	p.isShuttingDown = true
	p.mu.Unlock()

	// Cancel context to stop all operations
	p.cancel()

	// Close job queue
	close(p.jobQueue)

	// Wait for all goroutines to finish
	p.wg.Wait()

	// Close NATS connection
	if p.natsConn != nil {
		p.natsConn.Close()
	}

	// Clean up execution environment
	if p.executionEnv != nil && p.executionEnv.dockerClient != nil {
		p.executionEnv.dockerClient.Close()
	}

	p.logger.Info("GPU provider shutdown complete")
	return nil
}

// startHeartbeat sends periodic heartbeats to the registry
func (p *GPUProvider) startHeartbeat() {
	p.wg.Add(1)
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if err := p.sendHeartbeat(); err != nil {
				p.logger.Error("Failed to send heartbeat", zap.Error(err))
			}
		}
	}
}

// sendHeartbeat sends a heartbeat to the provider registry
func (p *GPUProvider) sendHeartbeat() error {
	// Update GPU metrics
	for i := range p.gpus {
		// Simple availability check
		p.gpus[i].IsAvailable = true
		p.gpus[i].LastCheckAt = time.Now()
	}

	// Send heartbeat to registry
	heartbeatData := map[string]interface{}{
		"provider_id": p.provider.ID,
		"status":      "online",
		"gpu_metrics": p.gpus,
		"timestamp":   time.Now(),
	}

	data, err := json.Marshal(heartbeatData)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat data: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/providers/%s/heartbeat", p.config.ProviderRegistryURL, p.provider.ID)
	req, err := http.NewRequestWithContext(p.ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	return nil
}

// startMetricsCollection starts periodic metrics collection
func (p *GPUProvider) startMetricsCollection() {
	p.wg.Add(1)
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.collectSystemMetrics()
		}
	}
}

// collectSystemMetrics collects system and GPU metrics
func (p *GPUProvider) collectSystemMetrics() {
	p.systemMetrics = &SystemMetrics{
		LastUpdated: time.Now(),
	}

	// Collect CPU metrics
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		p.systemMetrics.CPUUsage = cpuPercent[0]
	}

	// Collect memory metrics
	if memInfo, err := mem.VirtualMemory(); err == nil {
		p.systemMetrics.MemoryUsage = memInfo.Used / 1024 / 1024
		p.systemMetrics.MemoryTotal = memInfo.Total / 1024 / 1024
	}
}

// startHealthChecks starts periodic health checks
func (p *GPUProvider) startHealthChecks() {
	p.wg.Add(1)
	defer p.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.performHealthChecks()
		}
	}
}

// performHealthChecks performs various health checks
func (p *GPUProvider) performHealthChecks() {
	// Check GPU availability
	for i := range p.gpus {
		p.gpus[i].IsHealthy = true
		p.gpus[i].LastCheckAt = time.Now()
	}

	// Check Docker availability
	if p.executionEnv != nil && p.executionEnv.dockerClient != nil {
		_, err := p.executionEnv.dockerClient.Ping(p.ctx)
		if err != nil {
			p.logger.Warn("Docker health check failed", zap.Error(err))
		}
	}
}
