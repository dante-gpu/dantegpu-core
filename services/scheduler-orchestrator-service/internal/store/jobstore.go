package store

import (
	"context"

	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/models"
)

// JobStore defines the interface for storing and retrieving job state information.
// This allows for different backend implementations (e.g., in-memory, PostgreSQL).
type JobStore interface {
	// SaveJob saves the complete state of a job. This can be used for initial creation or full updates.
	SaveJob(ctx context.Context, jobRecord *models.JobRecord) error

	// GetJob retrieves a job by its ID.
	GetJob(ctx context.Context, jobID string) (*models.JobRecord, error)

	// UpdateJobState updates specific fields of a job: its state, assigned provider ID, last error, and increments attempts.
	// It also updates the UpdatedAt timestamp.
	UpdateJobState(ctx context.Context, jobID string, newState models.SchedulerJobState, providerID string, lastError string, attempts int) error

	// GetJobsByState retrieves a list of jobs matching a specific state.
	// This could be useful for re-processing pending jobs on startup or for monitoring.
	GetJobsByState(ctx context.Context, state models.SchedulerJobState, limit int) ([]*models.JobRecord, error)

	// GetPendingJobs retrieves jobs that are in a pending, searching, or previously failed state (for retry purposes).
	// This is a more specific query that might be useful on startup.
	GetRetryableJobs(ctx context.Context, limit int) ([]*models.JobRecord, error)

	// DeleteJob removes a job from the store (e.g., after successful completion and archival, or for cleanup).
	// This might be a less frequently used operation in the scheduler itself.
	DeleteJob(ctx context.Context, jobID string) error

	// Initialize is called to set up the store, e.g., create tables if they don't exist.
	Initialize(ctx context.Context) error

	// Close is called to release any resources held by the store, like DB connections.
	Close() error
}
