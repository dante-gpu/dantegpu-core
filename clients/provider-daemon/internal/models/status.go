package models

import (
	"fmt"
	"time"
)

// JobStatus represents the lifecycle state of a task being executed by the provider daemon.
// These are distinct from any job states managed by the scheduler.

type JobStatus string

// Constants for Task Execution Status by the Provider Daemon
const (
	StatusPreparing  JobStatus = "preparing"   // Task received, daemon is setting up
	StatusInProgress JobStatus = "in_progress" // Task is actively running
	StatusCompleted  JobStatus = "completed"   // Task finished successfully
	StatusFailed     JobStatus = "failed"      // Task failed to complete
	StatusCancelled  JobStatus = "cancelled"   // Task was cancelled (either by request or internal decision)
	StatusTimeout    JobStatus = "timeout"     // Task exceeded its execution time limit
)

// TaskStatusUpdate is sent by the provider daemon to report the current status of a task execution.
type TaskStatusUpdate struct {
	JobID        string    `json:"job_id"`
	ProviderID   string    `json:"provider_id"` // ID of the daemon instance reporting the status
	Status       JobStatus `json:"status"`
	Timestamp    time.Time `json:"timestamp"`               // When this status was recorded
	Message      string    `json:"message,omitempty"`       // Optional: human-readable message or error details
	Progress     float32   `json:"progress,omitempty"`      // Optional: 0.0 to 1.0 for progress if applicable
	ExitCode     *int      `json:"exit_code,omitempty"`     // Optional: Exit code of the task if it was a process
	ExecutionLog string    `json:"execution_log,omitempty"` // Optional: Snippet of recent logs or full log URI

	// Optional: Resource utilization at the time of this status update
	// CPUUsagePercent    float32 `json:"cpu_usage_percent,omitempty"`
	// MemoryUsageMB      float32 `json:"memory_usage_mb,omitempty"`
	// GPUUtilization   []GPUUtilizationMetrics `json:"gpu_utilization,omitempty"`
}

// NewTaskStatusUpdate creates a new TaskStatusUpdate with the current timestamp.
func NewTaskStatusUpdate(jobID, providerID string, status JobStatus, message string) *TaskStatusUpdate {
	return &TaskStatusUpdate{
		JobID:      jobID,
		ProviderID: providerID,
		Status:     status,
		Message:    message,
		Timestamp:  time.Now().UTC(),
	}
}

// String returns a human-readable string representation of the TaskStatusUpdate.
func (tsu *TaskStatusUpdate) String() string {
	return fmt.Sprintf("JobID: %s, Provider: %s, Status: %s, Time: %s, Msg: %s",
		tsu.JobID, tsu.ProviderID, tsu.Status, tsu.Timestamp.Format(time.RFC3339), tsu.Message)
}

// GPUUtilizationMetrics might be part of status updates in future if detailed per-GPU metrics are reported.
// type GPUUtilizationMetrics struct {
// 	GPUID string  `json:"gpu_id"`
// 	Usage float32 `json:"usage_percent"` // e.g., GPU core utilization
// 	MemoryUsedMB float32 `json:"memory_used_mb"`
// 	TemperatureC float32 `json:"temperature_c"`
// }
