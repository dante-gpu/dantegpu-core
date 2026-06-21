package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PostgresJobStore implements the JobStore interface using a PostgreSQL database.
type PostgresJobStore struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresJobStore creates a new PostgresJobStore.
// It expects a connected pgxpool.Pool.
func NewPostgresJobStore(db *pgxpool.Pool, logger *zap.Logger) *PostgresJobStore {
	return &PostgresJobStore{
		db:     db,
		logger: logger,
	}
}

// Initialize creates the necessary 'jobs' table if it doesn't already exist.
func (pjs *PostgresJobStore) Initialize(ctx context.Context) error {
	// Best practice: Use a migrations tool (like goose, sql-migrate, or Alembic for Python).
	// For this example, a simple CREATE TABLE IF NOT EXISTS is used for brevity.
	// In a production system, you'd have versioned migrations.
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS jobs (
		job_id VARCHAR(255) PRIMARY KEY,
		user_id VARCHAR(255) NOT NULL,
		job_details JSONB NOT NULL,
		state VARCHAR(50) NOT NULL,
		provider_id VARCHAR(255),
		attempts INTEGER DEFAULT 0,
		last_error TEXT,
		received_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL,
		submitted_at TIMESTAMPTZ NOT NULL, 
		job_name VARCHAR(255),            
		job_type VARCHAR(100),           
		gpu_type_requested VARCHAR(100), 
		priority INTEGER
	);

	-- Optional: Add indexes for commonly queried columns
	CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs (state);
	CREATE INDEX IF NOT EXISTS idx_jobs_user_id ON jobs (user_id);
	CREATE INDEX IF NOT EXISTS idx_jobs_updated_at ON jobs (updated_at);
	CREATE INDEX IF NOT EXISTS idx_jobs_provider_id ON jobs (provider_id) WHERE provider_id IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_jobs_priority ON jobs (priority DESC); -- For ordering by priority
	CREATE INDEX IF NOT EXISTS idx_jobs_job_type ON jobs (job_type);
	CREATE INDEX IF NOT EXISTS idx_jobs_gpu_type_requested ON jobs (gpu_type_requested);
	`

	_, err := pjs.db.Exec(ctx, createTableSQL)
	if err != nil {
		pjs.logger.Error("Failed to create 'jobs' table", zap.Error(err))
		return fmt.Errorf("initializing jobs table: %w", err)
	}
	pjs.logger.Info("'jobs' table checked/created successfully")
	return nil
}

// SaveJob saves the complete state of a job to the database.
// It uses an UPSERT (INSERT ON CONFLICT DO UPDATE) operation.
func (pjs *PostgresJobStore) SaveJob(ctx context.Context, jobRecord *models.JobRecord) error {
	jobRecord.UpdatedAt = time.Now().UTC() // Ensure UpdatedAt is always current

	// Marshal JobDetails to JSON manually because models.JobDetailsDB.Value() is for driver interaction,
	// and pgx might not automatically pick it up for direct struct embedding in Exec.
	jobDetailsJSON, err := json.Marshal(jobRecord.JobDetails)
	if err != nil {
		return fmt.Errorf("marshalling job_details for SaveJob: %w", err)
	}

	sqlQuery := `
	INSERT INTO jobs (
		job_id, user_id, job_details, state, provider_id, attempts, 
		last_error, received_at, updated_at, submitted_at, job_name, 
		job_type, gpu_type_requested, priority
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	ON CONFLICT (job_id) DO UPDATE SET
		user_id = EXCLUDED.user_id,
		job_details = EXCLUDED.job_details,
		state = EXCLUDED.state,
		provider_id = EXCLUDED.provider_id,
		attempts = EXCLUDED.attempts,
		last_error = EXCLUDED.last_error,
		-- received_at should not be updated on conflict
		updated_at = EXCLUDED.updated_at,
		submitted_at = EXCLUDED.submitted_at, 
		job_name = EXCLUDED.job_name, 
		job_type = EXCLUDED.job_type, 
		gpu_type_requested = EXCLUDED.gpu_type_requested, 
		priority = EXCLUDED.priority
	`

	_, err = pjs.db.Exec(ctx, sqlQuery,
		jobRecord.JobID,
		jobRecord.UserID,
		jobDetailsJSON, // Use marshalled JSON
		jobRecord.State,
		sql.NullString{String: jobRecord.ProviderID, Valid: jobRecord.ProviderID != ""},
		jobRecord.Attempts,
		sql.NullString{String: jobRecord.LastError, Valid: jobRecord.LastError != ""},
		jobRecord.ReceivedAt,
		jobRecord.UpdatedAt,
		jobRecord.SubmittedAt,
		jobRecord.JobName,
		jobRecord.JobType,
		jobRecord.GPUType,
		jobRecord.Priority,
	)

	if err != nil {
		pjs.logger.Error("Failed to save job state to DB", zap.String("job_id", jobRecord.JobID), zap.Error(err))
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("saving job %s (SQL state %s): %w", jobRecord.JobID, pgErr.Code, err)
		}
		return fmt.Errorf("saving job %s: %w", jobRecord.JobID, err)
	}
	pjs.logger.Debug("Successfully saved job state to DB", zap.String("job_id", jobRecord.JobID))
	return nil
}

// GetJob retrieves a job by its ID from the database.
func (pjs *PostgresJobStore) GetJob(ctx context.Context, jobID string) (*models.JobRecord, error) {
	sqlQuery := `
	SELECT 
		job_id, user_id, job_details, state, provider_id, attempts, 
		last_error, received_at, updated_at, submitted_at, job_name, 
		job_type, gpu_type_requested, priority
	FROM jobs WHERE job_id = $1
	`
	jobRecord := &models.JobRecord{}
	var providerIDNullable sql.NullString
	var lastErrorNullable sql.NullString
	var jobDetailsBytes []byte

	err := pjs.db.QueryRow(ctx, sqlQuery, jobID).Scan(
		&jobRecord.JobID,
		&jobRecord.UserID,
		&jobDetailsBytes,
		&jobRecord.State,
		&providerIDNullable,
		&jobRecord.Attempts,
		&lastErrorNullable,
		&jobRecord.ReceivedAt,
		&jobRecord.UpdatedAt,
		&jobRecord.SubmittedAt,
		&jobRecord.JobName,
		&jobRecord.JobType,
		&jobRecord.GPUType,
		&jobRecord.Priority,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			pjs.logger.Debug("Job not found in DB", zap.String("job_id", jobID))
			return nil, nil
		}
		pjs.logger.Error("Failed to get job from DB", zap.String("job_id", jobID), zap.Error(err))
		return nil, fmt.Errorf("getting job %s: %w", jobID, err)
	}

	// Unmarshal job_details from []byte
	if err := json.Unmarshal(jobDetailsBytes, &jobRecord.JobDetails); err != nil {
		pjs.logger.Error("Failed to unmarshal job_details for job from DB", zap.String("job_id", jobID), zap.Error(err))
		return nil, fmt.Errorf("unmarshalling job_details for job %s: %w", jobID, err)
	}

	if providerIDNullable.Valid {
		jobRecord.ProviderID = providerIDNullable.String
	}
	if lastErrorNullable.Valid {
		jobRecord.LastError = lastErrorNullable.String
	}

	pjs.logger.Debug("Successfully retrieved job from DB", zap.String("job_id", jobRecord.JobID))
	return jobRecord, nil
}

// UpdateJobState updates specific fields of a job in the database.
func (pjs *PostgresJobStore) UpdateJobState(ctx context.Context, jobID string, newState models.SchedulerJobState, providerID string, lastError string, attempts int) error {
	sqlQuery := `
	UPDATE jobs 
	SET state = $1, provider_id = $2, last_error = $3, attempts = $4, updated_at = $5
	WHERE job_id = $6
	`
	updatedAt := time.Now().UTC()

	cmdTag, err := pjs.db.Exec(ctx, sqlQuery,
		newState,
		sql.NullString{String: providerID, Valid: providerID != ""},
		sql.NullString{String: lastError, Valid: lastError != ""},
		attempts,
		updatedAt,
		jobID,
	)

	if err != nil {
		pjs.logger.Error("Failed to update job state in DB", zap.String("job_id", jobID), zap.Error(err))
		return fmt.Errorf("updating job state for %s: %w", jobID, err)
	}
	if cmdTag.RowsAffected() == 0 {
		pjs.logger.Warn("UpdateJobState affected no rows, job might not exist", zap.String("job_id", jobID))
		// Return nil or a specific error like ErrNotFound? For update, usually nil if no error and 0 rows affected is okay.
		return nil // Or a custom ErrNotFound error
	}

	pjs.logger.Debug("Successfully updated job state in DB", zap.String("job_id", jobID), zap.String("new_state", string(newState)))
	return nil
}

// scanJobRows is a helper function to scan multiple rows into a slice of JobRecord.
func (pjs *PostgresJobStore) scanJobRows(rows pgx.Rows) ([]*models.JobRecord, error) {
	var jobs []*models.JobRecord
	defer rows.Close()

	for rows.Next() {
		jobRecord := &models.JobRecord{}
		var providerIDNullable sql.NullString
		var lastErrorNullable sql.NullString
		var jobDetailsBytes []byte

		err := rows.Scan(
			&jobRecord.JobID,
			&jobRecord.UserID,
			&jobDetailsBytes,
			&jobRecord.State,
			&providerIDNullable,
			&jobRecord.Attempts,
			&lastErrorNullable,
			&jobRecord.ReceivedAt,
			&jobRecord.UpdatedAt,
			&jobRecord.SubmittedAt,
			&jobRecord.JobName,
			&jobRecord.JobType,
			&jobRecord.GPUType,
			&jobRecord.Priority,
		)
		if err != nil {
			pjs.logger.Error("Error scanning job row", zap.Error(err))
			return nil, fmt.Errorf("scanning job row: %w", err)
		}

		if err := json.Unmarshal(jobDetailsBytes, &jobRecord.JobDetails); err != nil {
			pjs.logger.Error("Failed to unmarshal job_details from scanned row", zap.String("job_id", jobRecord.JobID), zap.Error(err))
			return nil, fmt.Errorf("unmarshalling job_details for job %s from scan: %w", jobRecord.JobID, err)
		}

		if providerIDNullable.Valid {
			jobRecord.ProviderID = providerIDNullable.String
		}
		if lastErrorNullable.Valid {
			jobRecord.LastError = lastErrorNullable.String
		}
		jobs = append(jobs, jobRecord)
	}

	if rows.Err() != nil {
		pjs.logger.Error("Error iterating over job rows", zap.Error(rows.Err()))
		return nil, fmt.Errorf("iterating job rows: %w", rows.Err())
	}
	return jobs, nil
}

// GetJobsByState retrieves a list of jobs matching a specific state.
func (pjs *PostgresJobStore) GetJobsByState(ctx context.Context, state models.SchedulerJobState, limit int) ([]*models.JobRecord, error) {
	sqlQuery := `
	SELECT 
		job_id, user_id, job_details, state, provider_id, attempts, 
		last_error, received_at, updated_at, submitted_at, job_name, 
		job_type, gpu_type_requested, priority
	FROM jobs 
	WHERE state = $1 
	ORDER BY updated_at ASC -- Process older pending jobs first, for example
	LIMIT $2
	`
	rows, err := pjs.db.Query(ctx, sqlQuery, state, limit)
	if err != nil {
		pjs.logger.Error("Failed to get jobs by state from DB", zap.String("state", string(state)), zap.Error(err))
		return nil, fmt.Errorf("getting jobs by state %s: %w", state, err)
	}
	return pjs.scanJobRows(rows)
}

// GetRetryableJobs retrieves jobs that are in a pending or searching state, or failed with few attempts.
// This is a simplified example; more complex retry logic might be needed.
func (pjs *PostgresJobStore) GetRetryableJobs(ctx context.Context, limit int) ([]*models.JobRecord, error) {
	// Example: Retry jobs that are pending, searching, or failed less than 3 times
	// and were last updated recently (e.g., within the last hour, to avoid retrying very old stuck jobs indefinitely without review)
	// This logic can be made more sophisticated.
	maxAttempts := 3
	// lookbackTime := time.Now().UTC().Add(-1 * time.Hour) // Example lookback

	sqlQuery := `
	SELECT 
		job_id, user_id, job_details, state, provider_id, attempts, 
		last_error, received_at, updated_at, submitted_at, job_name, 
		job_type, gpu_type_requested, priority
	FROM jobs 
	WHERE (state = $1 OR state = $2 OR (state = $3 AND attempts < $4)) 
	-- AND updated_at > $5 -- Optional: only retry recently updated ones
	ORDER BY priority DESC, updated_at ASC -- Prioritize by user priority then by oldest update
	LIMIT $5
	`
	rows, err := pjs.db.Query(ctx, sqlQuery,
		models.JobStatePending,
		models.JobStateSearching,
		models.JobStateFailed,
		maxAttempts,
		// lookbackTime, // if lookback is used
		limit,
	)
	if err != nil {
		pjs.logger.Error("Failed to get retryable jobs from DB", zap.Error(err))
		return nil, fmt.Errorf("getting retryable jobs: %w", err)
	}
	return pjs.scanJobRows(rows)
}

// DeleteJob removes a job from the store.
func (pjs *PostgresJobStore) DeleteJob(ctx context.Context, jobID string) error {
	sqlQuery := `DELETE FROM jobs WHERE job_id = $1`
	cmdTag, err := pjs.db.Exec(ctx, sqlQuery, jobID)
	if err != nil {
		pjs.logger.Error("Failed to delete job from DB", zap.String("job_id", jobID), zap.Error(err))
		return fmt.Errorf("deleting job %s: %w", jobID, err)
	}
	if cmdTag.RowsAffected() == 0 {
		pjs.logger.Warn("DeleteJob affected no rows, job might not exist or already deleted", zap.String("job_id", jobID))
		// Usually not an error if the intent is to ensure it's gone.
	}
	pjs.logger.Info("Successfully deleted job from DB (or it was already gone)", zap.String("job_id", jobID))
	return nil
}

// Close closes the database connection pool.
func (pjs *PostgresJobStore) Close() error {
	if pjs.db != nil {
		pjs.logger.Info("Closing PostgresJobStore database connection pool...")
		pjs.db.Close() // pgxpool.Pool.Close() waits for all connections to be returned to the pool and closes them.
		pjs.logger.Info("PostgresJobStore database connection pool closed.")
	}
	return nil
}

// Ensure pgx.Rows is imported for scanJobRows
// If it's not directly, it might be through pgxpool or another pgx subpackage.
// For explicit import if needed: import "github.com/jackc/pgx/v5"
// However, pgxpool.Rows is typically what Query returns, which is compatible.
// Let's adjust scanJobRows to use pgxpool.Rows, which Query returns.
// Update: pgxpool.Rows is an alias for pgx.Rows, so it should be fine.
// The type is actually pgx.Rows, from Query method of pgxpool.Pool. The helper looks correct.
