package models

import (
	"time"
)

// Task represents a unit of work to be dispatched to a provider daemon.
// It contains essential details from the original job and any specific instructions
// or context needed by the daemon to execute the job.
type Task struct {
	JobID          string                 `json:"job_id"`     // Original Job ID from the user/API gateway
	UserID         string                 `json:"user_id"`    // User who submitted the job
	JobType        string                 `json:"job_type"`   // e.g., "ai-training", "data-processing"
	JobName        string                 `json:"job_name"`   // User-defined name for the job
	JobParams      map[string]interface{} `json:"job_params"` // Job-specific parameters (script, dataset, hyperparameters)
	GPUTypeNeeded  string                 `json:"gpu_type_needed,omitempty"`
	GPUCountNeeded int                    `json:"gpu_count_needed,omitempty"`

	// Information about the assigned provider (optional, but useful for the daemon)
	AssignedProviderID string `json:"assigned_provider_id,omitempty"`

	// Dispatch details
	DispatchedAt time.Time `json:"dispatched_at"`
	// ExecutionTimeout time.Duration `json:"execution_timeout,omitempty"` // Max time daemon should run this task

	// Other fields might include:
	// - URI for input data/models
	// - URI for output storage
	// - Environment variables to set
	// - Docker image to use
}

// NewTask creates a new Task from a Job and an assigned provider ID.
func NewTask(job *Job, assignedProviderID string) *Task {
	return &Task{
		JobID:              job.ID,
		UserID:             job.UserID,
		JobType:            job.Type,
		JobName:            job.Name,
		JobParams:          job.Params,
		GPUTypeNeeded:      job.GPUType,
		GPUCountNeeded:     job.GPUCount,
		AssignedProviderID: assignedProviderID,
		DispatchedAt:       time.Now().UTC(),
	}
}
