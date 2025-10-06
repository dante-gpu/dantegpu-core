package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// JobHandler handles job operations
type JobHandler struct {
	db   *sql.DB
	nats *nats.Conn
}

// SubmitJobRequest represents job submission
type SubmitJobRequest struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	DockerImage     string            `json:"docker_image"`
	Command         []string          `json:"command"`
	Environment     map[string]string `json:"environment"`
	GPUCapabilityID string            `json:"gpu_capability_id"`
	MaxDuration     int               `json:"max_duration_hours"`
	DatasetURLs     []string          `json:"dataset_urls"`
}

// NewJobHandler creates a new job handler
func NewJobHandler(db *sql.DB, nats *nats.Conn) *JobHandler {
	return &JobHandler{db: db, nats: nats}
}

// SubmitJob submits a new job
func (h *JobHandler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req SubmitJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate GPU capability
	var providerID string
	var hourlyRate float64
	err := h.db.QueryRowContext(ctx, `
		SELECT provider_id, hourly_rate
		FROM gpu_capabilities
		WHERE id = $1 AND is_available = true
	`, req.GPUCapabilityID).Scan(&providerID, &hourlyRate)

	if err != nil {
		respondError(w, http.StatusBadRequest, "GPU not available")
		return
	}

	// Create job
	jobID := uuid.New().String()
	now := time.Now()

	envJSON, _ := json.Marshal(req.Environment)
	cmdJSON, _ := json.Marshal(req.Command)
	datasetsJSON, _ := json.Marshal(req.DatasetURLs)

	_, err = h.db.ExecContext(ctx, `
		INSERT INTO jobs (
			id, user_id, name, description, docker_image,
			command, environment, gpu_capability_id, provider_id,
			max_duration_hours, dataset_urls, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, jobID, userID, req.Name, req.Description, req.DockerImage,
		cmdJSON, envJSON, req.GPUCapabilityID, providerID,
		req.MaxDuration, datasetsJSON, "pending", now)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create job")
		return
	}

	// Publish job to NATS
	jobMsg, _ := json.Marshal(map[string]interface{}{
		"job_id":             jobID,
		"user_id":            userID,
		"provider_id":        providerID,
		"gpu_capability_id":  req.GPUCapabilityID,
		"docker_image":       req.DockerImage,
		"command":            req.Command,
		"environment":        req.Environment,
		"hourly_rate":        hourlyRate,
		"estimated_minutes":  req.MaxDuration * 60,
	})

	h.nats.Publish("JOBS.submit", jobMsg)

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":          jobID,
		"status":      "pending",
		"hourly_rate": hourlyRate,
		"created_at":  now,
	})
}

// GetJob gets job details
func (h *JobHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		respondError(w, http.StatusBadRequest, "Job ID required")
		return
	}

	var job struct {
		ID          string
		Name        string
		Description string
		DockerImage string
		Status      string
		CreatedAt   time.Time
		StartedAt   sql.NullTime
		CompletedAt sql.NullTime
		Error       sql.NullString
	}

	err := h.db.QueryRowContext(ctx, `
		SELECT id, name, description, docker_image, status,
		       created_at, started_at, completed_at, error
		FROM jobs
		WHERE id = $1 AND user_id = $2
	`, jobID, userID).Scan(&job.ID, &job.Name, &job.Description,
		&job.DockerImage, &job.Status, &job.CreatedAt,
		&job.StartedAt, &job.CompletedAt, &job.Error)

	if err != nil {
		respondError(w, http.StatusNotFound, "Job not found")
		return
	}

	response := map[string]interface{}{
		"id":           job.ID,
		"name":         job.Name,
		"description":  job.Description,
		"docker_image": job.DockerImage,
		"status":       job.Status,
		"created_at":   job.CreatedAt,
	}

	if job.StartedAt.Valid {
		response["started_at"] = job.StartedAt.Time
	}
	if job.CompletedAt.Valid {
		response["completed_at"] = job.CompletedAt.Time
	}
	if job.Error.Valid {
		response["error"] = job.Error.String
	}

	respondJSON(w, http.StatusOK, response)
}

// ListJobs lists user jobs
func (h *JobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	status := r.URL.Query().Get("status")

	query := `
		SELECT id, name, docker_image, status, created_at, started_at, completed_at
		FROM jobs
		WHERE user_id = $1
	`
	args := []interface{}{userID}

	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}

	query += ` ORDER BY created_at DESC LIMIT 100`

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list jobs")
		return
	}
	defer rows.Close()

	var jobs []map[string]interface{}
	for rows.Next() {
		var j struct {
			ID          string
			Name        string
			DockerImage string
			Status      string
			CreatedAt   time.Time
			StartedAt   sql.NullTime
			CompletedAt sql.NullTime
		}
		rows.Scan(&j.ID, &j.Name, &j.DockerImage, &j.Status,
			&j.CreatedAt, &j.StartedAt, &j.CompletedAt)

		job := map[string]interface{}{
			"id":           j.ID,
			"name":         j.Name,
			"docker_image": j.DockerImage,
			"status":       j.Status,
			"created_at":   j.CreatedAt,
		}

		if j.StartedAt.Valid {
			job["started_at"] = j.StartedAt.Time
		}
		if j.CompletedAt.Valid {
			job["completed_at"] = j.CompletedAt.Time
		}

		jobs = append(jobs, job)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// CancelJob cancels a job
func (h *JobHandler) CancelJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Update job status
	result, err := h.db.ExecContext(ctx, `
		UPDATE jobs
		SET status = 'cancelled', completed_at = $1
		WHERE id = $2 AND user_id = $3 AND status IN ('pending', 'running')
	`, time.Now(), req.JobID, userID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to cancel job")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Job not found or cannot be cancelled")
		return
	}

	// Publish cancellation event
	cancelMsg, _ := json.Marshal(map[string]interface{}{
		"job_id":  req.JobID,
		"user_id": userID,
		"action":  "cancel",
	})
	h.nats.Publish("JOBS.cancel", cancelMsg)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Job cancelled successfully",
	})
}

// GetJobLogs gets job logs
func (h *JobHandler) GetJobLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		respondError(w, http.StatusBadRequest, "Job ID required")
		return
	}

	// Verify job ownership
	var exists bool
	err := h.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM jobs WHERE id = $1 AND user_id = $2)
	`, jobID, userID).Scan(&exists)

	if err != nil || !exists {
		respondError(w, http.StatusNotFound, "Job not found")
		return
	}

	// Get logs
	rows, err := h.db.QueryContext(ctx, `
		SELECT log_level, message, created_at
		FROM job_logs
		WHERE job_id = $1
		ORDER BY created_at ASC
		LIMIT 1000
	`, jobID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get logs")
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var log struct {
			Level     string
			Message   string
			CreatedAt time.Time
		}
		rows.Scan(&log.Level, &log.Message, &log.CreatedAt)

		logs = append(logs, map[string]interface{}{
			"level":      log.Level,
			"message":    log.Message,
			"created_at": log.CreatedAt,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

