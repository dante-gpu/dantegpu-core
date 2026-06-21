package models

import (
	"time"
)

// ExecutionType defines the type of execution required for a task.
type ExecutionType string

const (
	// ExecutionTypeScript indicates the task should be run as a script.
	ExecutionTypeScript ExecutionType = "script"
	// ExecutionTypeDocker indicates the task should be run in a Docker container.
	ExecutionTypeDocker ExecutionType = "docker"
	// ExecutionTypeUndefined indicates the execution type was not specified.
	ExecutionTypeUndefined ExecutionType = ""
)

// Task represents a unit of work to be dispatched from the scheduler to a provider daemon.
// It contains essential details from the original job and any specific instructions
// or context needed by the daemon to execute the job.
// This structure MUST be kept in sync with scheduler-orchestrator-service/internal/models/task.go

// Task struct definition
type Task struct {
	JobID     string                 `json:"job_id"`     // Original Job ID from the user/API gateway
	UserID    string                 `json:"user_id"`    // User who submitted the job
	JobType   string                 `json:"job_type"`   // e.g., "ai-training", "data-processing", "script_execution"
	JobName   string                 `json:"job_name"`   // User-defined name for the job
	JobParams map[string]interface{} `json:"job_params"` // Job-specific parameters (e.g., script content, dataset URI, hyperparameters, docker_image)

	ExecutionType ExecutionType `json:"execution_type"` // Specifies whether to use ScriptExecutor or DockerExecutor

	// Resource requirements (can be used by daemon for validation or local scheduling if managing multiple local GPUs)
	GPUTypeNeeded  string `json:"gpu_type_needed,omitempty"`
	GPUCountNeeded int    `json:"gpu_count_needed,omitempty"`

	// Information about the assigned provider (this daemon instance)
	AssignedProviderID string `json:"assigned_provider_id"` // This should match the daemon's instance ID

	// Dispatch details
	DispatchedAt time.Time `json:"dispatched_at"` // Timestamp when the scheduler dispatched this task
	// ExecutionTimeout time.Duration `json:"execution_timeout,omitempty"` // Max time daemon should run this task
	SelectedGPU *SelectedGPUInfo `json:"selected_gpu,omitempty"` // Information about the specific GPU instance selected for this task
}

// SelectedGPUInfo holds details about the GPU instance assigned to a task.
type SelectedGPUInfo struct {
	InstanceID   string  `json:"instance_id"`    // Unique identifier for the GPU instance
	PricePerHour float64 `json:"price_per_hour"` // Billing price per hour for this GPU instance
	// Potentially other details like VRAM, Model, etc., if needed by the daemon directly
}

// Example JobParams structure for a script execution task:
// JobParams: {
//   "script_language": "python", // or "bash", etc.
//   "script_content": "print('Hello from Dante GPU Platform!')",
//   "docker_image": "python:3.9-slim", // Optional: if execution should be in a container
//   "requirements": ["numpy==1.23.0", "pandas"], // Optional: Python requirements
//   "input_data_urls": ["s3://my-bucket/data/input1.csv"], // Optional
//   "output_data_path": "s3://my-bucket/results/job_id/" // Optional
// }
