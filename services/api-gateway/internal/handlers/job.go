package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dante-gpu/dante-backend/api-gateway/internal/auth"
	"github.com/dante-gpu/dante-backend/api-gateway/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// JobHandler holds dependencies for job-related handlers.
// I need the logger, config, and NATS connection.
type JobHandler struct {
	Logger   *zap.Logger
	Config   *config.Config
	NatsConn *nats.Conn
	// NatsJS nats.JetStreamContext // I might need JetStream later for guaranteed delivery
}

// NewJobHandler creates a new JobHandler.
func NewJobHandler(logger *zap.Logger, cfg *config.Config, nc *nats.Conn) *JobHandler {
	return &JobHandler{Logger: logger, Config: cfg, NatsConn: nc}
}

// SubmitJobRequest defines the structure for the job submission request body.
// Based on the provided example.
type SubmitJobRequest struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	GPUType     string                 `json:"gpu_type,omitempty"`
	GPUCount    int                    `json:"gpu_count,omitempty"`
	Priority    int                    `json:"priority,omitempty"`
	Params      map[string]interface{} `json:"params"`
	Tags        []string               `json:"tags,omitempty"`
	// I might add UserID from context later
	UserID string `json:"-"` // Added internally from JWT
}

// SubmitJobResponse defines the structure for the job submission response body.
type SubmitJobResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// SubmitJob handles requests to submit a new job.
// It publishes the job request to a NATS subject.
func (h *JobHandler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	var req SubmitJobRequest
	// I need to decode the request body.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("Failed to decode job submission request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// I should perform basic validation.
	if req.Type == "" || req.Name == "" || len(req.Params) == 0 {
		http.Error(w, "Type, name, and params are required fields", http.StatusBadRequest)
		return
	}

	// I should get the UserID from the JWT claims in the context.
	claims, ok := r.Context().Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok || claims == nil {
		h.Logger.Error("Claims not found in context for job submission")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	req.UserID = claims.UserID

	// I need to generate a unique Job ID.
	jobID := uuid.New().String()

	// I should marshal the job request (including UserID and JobID) into JSON for NATS.
	jobData, err := json.Marshal(struct {
		SubmitJobRequest
		JobID string `json:"job_id"`
	}{SubmitJobRequest: req, JobID: jobID})

	if err != nil {
		h.Logger.Error("Failed to marshal job data for NATS", zap.Error(err))
		http.Error(w, "Failed to process job submission", http.StatusInternalServerError)
		return
	}

	// I need to determine the NATS subject (e.g., based on job type or priority).
	// Using a simple subject for now.
	natsSubject := "jobs.submitted"

	// I should publish the job data to NATS.
	if err := h.NatsConn.Publish(natsSubject, jobData); err != nil {
		h.Logger.Error("Failed to publish job to NATS",
			zap.String("subject", natsSubject),
			zap.Error(err))
		http.Error(w, "Failed to submit job via message queue", http.StatusInternalServerError)
		return
	}

	h.Logger.Info("Job submitted successfully to NATS",
		zap.String("job_id", jobID),
		zap.String("subject", natsSubject),
		zap.String("user_id", req.UserID),
	)

	// Respond with success message.
	resp := SubmitJobResponse{
		JobID:     jobID,
		Status:    "queued", // Initial status
		Timestamp: time.Now(),
		Message:   "Job submitted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted for async processing
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.Logger.Error("Failed to encode job submission response", zap.Error(err))
	}
}

// GetJobStatus handles requests to get the status of a specific job.
// NOTE: This is a placeholder. The API Gateway might not be the ideal place
// to query job status directly. This might involve querying the
// scheduler-orchestrator-service or a dedicated job status service via REST/gRPC,
// or potentially using NATS request-reply if the responsible service listens.
// For now, it just returns a mock response.
func (h *JobHandler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	h.Logger.Info("Received request for job status", zap.String("jobID", jobID))

	// --- Placeholder Logic ---
	// In a real implementation:
	// 1. Extract jobID from URL.
	// 2. Make a request (e.g., gRPC) to the scheduler/job service to get the status.
	// 3. Handle potential errors (not found, service unavailable).
	// 4. Return the actual status.
	// --- End Placeholder ---

	mockStatus := "processing" // Or "queued", "completed", "failed", "cancelled"
	if jobID == "known-completed-id" {
		mockStatus = "completed"
	}

	resp := map[string]interface{}{ // Using a map for flexibility in mock response
		"job_id":    jobID,
		"status":    mockStatus,
		"timestamp": time.Now(),
		"message":   fmt.Sprintf("Mock status for job %s is %s", jobID, mockStatus),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.Logger.Error("Failed to encode job status response", zap.Error(err))
	}
}

// CancelJob handles requests to cancel a running job.
// NOTE: Similar to GetJobStatus, this is a placeholder.
// Cancellation likely involves sending a command via NATS or gRPC
// to the scheduler/orchestrator or the daemon running the job.
func (h *JobHandler) CancelJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	h.Logger.Info("Received request to cancel job", zap.String("jobID", jobID))

	// --- Placeholder Logic ---
	// In a real implementation:
	// 1. Extract jobID.
	// 2. Send a cancellation request (e.g., NATS message `jobs.cancel` with jobID)
	//    to the scheduler/orchestrator.
	// 3. The scheduler would then attempt to stop the job.
	// 4. Return an appropriate response (e.g., "cancellation requested").
	// --- End Placeholder ---

	resp := map[string]interface{}{ // Using a map for flexibility
		"job_id":    jobID,
		"status":    "cancelling", // Indicate request is being processed
		"timestamp": time.Now(),
		"message":   fmt.Sprintf("Job cancellation requested for %s", jobID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted - request is being processed
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.Logger.Error("Failed to encode job cancellation response", zap.Error(err))
	}
}
