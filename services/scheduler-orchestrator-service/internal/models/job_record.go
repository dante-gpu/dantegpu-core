package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
	// Assuming job IDs can be UUIDs or strings. For now, string.
)

// JobDetailsDB is a wrapper for Job to handle JSONB storage in PostgreSQL.
type JobDetailsDB Job // Use the existing Job model

// Value implements the driver.Valuer interface for JobDetailsDB.
// This tells the SQL driver how to store the JobDetailsDB struct in the database.
func (jd JobDetailsDB) Value() (driver.Value, error) {
	return json.Marshal(jd)
}

// Scan implements the sql.Scanner interface for JobDetailsDB.
// This tells the SQL driver how to read the JobDetailsDB struct from the database.
func (jd *JobDetailsDB) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for JobDetailsDB")
	}
	return json.Unmarshal(b, &jd)
}

// JobRecord represents the structure of a job as stored in the database.
// This is derived from InternalJobRepresentation but adapted for DB storage,
// particularly for handling the JobDetails as JSONB.
type JobRecord struct {
	JobID       string            `db:"job_id"` // Primary Key
	UserID      string            `db:"user_id"`
	JobDetails  JobDetailsDB      `db:"job_details"` // Stored as JSONB
	State       SchedulerJobState `db:"state"`
	ProviderID  string            `db:"provider_id"` // Nullable in DB
	Attempts    int               `db:"attempts"`
	LastError   string            `db:"last_error"` // Nullable in DB
	ReceivedAt  time.Time         `db:"received_at"`
	UpdatedAt   time.Time         `db:"updated_at"`
	SubmittedAt time.Time         `db:"submitted_at"`       // From original job details
	JobName     string            `db:"job_name"`           // For easier querying/indexing
	JobType     string            `db:"job_type"`           // For easier querying/indexing
	GPUType     string            `db:"gpu_type_requested"` // For easier querying/indexing
	Priority    int               `db:"priority"`           // For easier querying/indexing
}

// ToInternalJobRepresentation converts a JobRecord from the database back to an InternalJobRepresentation.
func (jr *JobRecord) ToInternalJobRepresentation() *InternalJobRepresentation {
	// The JobDetails in JobRecord is already of type Job (via JobDetailsDB alias),
	// so we can directly assign it.
	internalJob := &InternalJobRepresentation{
		JobDetails: Job(jr.JobDetails), // Convert JobDetailsDB back to Job
		State:      jr.State,
		ProviderID: jr.ProviderID,
		Attempts:   jr.Attempts,
		LastError:  jr.LastError,
		ReceivedAt: jr.ReceivedAt,
		UpdatedAt:  jr.UpdatedAt,
	}
	// Ensure original Job fields are correctly populated in JobDetails
	internalJob.JobDetails.ID = jr.JobID
	internalJob.JobDetails.UserID = jr.UserID
	// SubmittedAt, Name, Type, GPUType, Priority should be part of jr.JobDetails when unmarshalled.
	// If not, ensure they are explicitly mapped if they were stored as separate columns.
	// For this example, we assume JobDetailsDB correctly populates these from the JSONB.
	// However, if we denormalized some fields (like JobName, JobType for querying),
	// we need to make sure they are consistent or primarily rely on the JSONB source of truth.

	// Let's ensure the core fields of JobDetails are set from the JobRecord if they were denormalized
	// This handles cases where JobDetails might not have all fields if it was a partial unmarshal,
	// or if we prefer to use the top-level denormalized fields from JobRecord.
	if internalJob.JobDetails.ID == "" { // If JobID was not in the JSONB, use the record's JobID
		internalJob.JobDetails.ID = jr.JobID
	}
	if internalJob.JobDetails.UserID == "" {
		internalJob.JobDetails.UserID = jr.UserID
	}
	if internalJob.JobDetails.Name == "" {
		internalJob.JobDetails.Name = jr.JobName
	}
	if internalJob.JobDetails.Type == "" {
		internalJob.JobDetails.Type = jr.JobType
	}
	if internalJob.JobDetails.GPUType == "" && jr.GPUType != "" {
		internalJob.JobDetails.GPUType = jr.GPUType
	}
	if internalJob.JobDetails.Priority == 0 && jr.Priority != 0 {
		internalJob.JobDetails.Priority = jr.Priority
	}
	if internalJob.JobDetails.SubmittedAt.IsZero() && !jr.SubmittedAt.IsZero() {
		internalJob.JobDetails.SubmittedAt = jr.SubmittedAt
	}

	return internalJob
}

// FromInternalJobRepresentation converts an InternalJobRepresentation to a JobRecord for database storage.
func FromInternalJobRepresentation(internalJob *InternalJobRepresentation) *JobRecord {
	jobRecord := &JobRecord{
		JobID:      internalJob.JobDetails.ID,
		UserID:     internalJob.JobDetails.UserID,
		JobDetails: JobDetailsDB(internalJob.JobDetails), // Convert Job to JobDetailsDB
		State:      internalJob.State,
		ProviderID: internalJob.ProviderID,
		Attempts:   internalJob.Attempts,
		LastError:  internalJob.LastError,
		ReceivedAt: internalJob.ReceivedAt,
		UpdatedAt:  internalJob.UpdatedAt,
		// Denormalized fields for querying/indexing:
		SubmittedAt: internalJob.JobDetails.SubmittedAt,
		JobName:     internalJob.JobDetails.Name,
		JobType:     internalJob.JobDetails.Type,
		GPUType:     internalJob.JobDetails.GPUType,
		Priority:    internalJob.JobDetails.Priority,
	}
	return jobRecord
}

// Note: `db:"..."` tags are for libraries like sqlx, which can simplify mapping.
// If using standard database/sql, these tags are just for documentation/reference unless
// a helper library uses them. For pgx, we'll manually map fields.
