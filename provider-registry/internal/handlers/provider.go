package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ProviderHandler handles provider operations
type ProviderHandler struct {
	db *sql.DB
}

// RegisterProviderRequest represents provider registration
type RegisterProviderRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Email       string `json:"email"`
	Location    string `json:"location"`
	WalletAddr  string `json:"wallet_address"`
}

// GPUCapabilityRequest represents GPU capability registration
type GPUCapabilityRequest struct {
	GPUModel      string  `json:"gpu_model"`
	GPUMemory     int     `json:"gpu_memory_gb"`
	CPUCores      int     `json:"cpu_cores"`
	RAMSize       int     `json:"ram_size_gb"`
	StorageSize   int     `json:"storage_size_gb"`
	HourlyRate    float64 `json:"hourly_rate"`
	MaxJobDuration int    `json:"max_job_duration_hours"`
}

// NewProviderHandler creates a new provider handler
func NewProviderHandler(db *sql.DB) *ProviderHandler {
	return &ProviderHandler{db: db}
}

// RegisterProvider registers a new provider
func (h *ProviderHandler) RegisterProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req RegisterProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Check if already registered
	var exists bool
	err := h.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM providers WHERE user_id = $1)
	`, userID).Scan(&exists)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check provider")
		return
	}
	if exists {
		respondError(w, http.StatusConflict, "Provider already registered")
		return
	}

	// Create provider
	providerID := uuid.New().String()
	now := time.Now()

	_, err = h.db.ExecContext(ctx, `
		INSERT INTO providers (
			id, user_id, name, description, email, location,
			wallet_address, status, verification_status,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, providerID, userID, req.Name, req.Description, req.Email, req.Location,
		req.WalletAddr, "active", "pending", now, now)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to register provider")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":                  providerID,
		"name":                req.Name,
		"verification_status": "pending",
		"created_at":          now,
	})
}

// AddGPUCapability adds GPU capability
func (h *ProviderHandler) AddGPUCapability(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req GPUCapabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Get provider ID
	var providerID string
	err := h.db.QueryRowContext(ctx, `
		SELECT id FROM providers WHERE user_id = $1
	`, userID).Scan(&providerID)

	if err != nil {
		respondError(w, http.StatusNotFound, "Provider not found")
		return
	}

	// Get GPU model ID
	var gpuModelID string
	err = h.db.QueryRowContext(ctx, `
		SELECT id FROM gpu_models WHERE name = $1
	`, req.GPUModel).Scan(&gpuModelID)

	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid GPU model")
		return
	}

	// Create capability
	capabilityID := uuid.New().String()
	now := time.Now()

	_, err = h.db.ExecContext(ctx, `
		INSERT INTO gpu_capabilities (
			id, provider_id, gpu_model_id, gpu_memory_gb, cpu_cores,
			ram_size_gb, storage_size_gb, hourly_rate, max_job_duration_hours,
			is_available, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, capabilityID, providerID, gpuModelID, req.GPUMemory, req.CPUCores,
		req.RAMSize, req.StorageSize, req.HourlyRate, req.MaxJobDuration,
		true, now, now)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to add capability")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":          capabilityID,
		"gpu_model":   req.GPUModel,
		"hourly_rate": req.HourlyRate,
		"created_at":  now,
	})
}

// ListGPUs lists available GPUs
func (h *ProviderHandler) ListGPUs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	minMemory := r.URL.Query().Get("min_memory")
	maxRate := r.URL.Query().Get("max_rate")
	gpuModel := r.URL.Query().Get("gpu_model")

	query := `
		SELECT gc.id, gc.gpu_memory_gb, gc.cpu_cores, gc.ram_size_gb,
		       gc.storage_size_gb, gc.hourly_rate, gc.is_available,
		       gm.name as gpu_model, gm.compute_capability,
		       p.name as provider_name, p.rating
		FROM gpu_capabilities gc
		JOIN gpu_models gm ON gm.id = gc.gpu_model_id
		JOIN providers p ON p.id = gc.provider_id
		WHERE gc.is_available = true AND p.status = 'active'
	`

	args := []interface{}{}
	argCount := 1

	if minMemory != "" {
		query += ` AND gc.gpu_memory_gb >= $` + string(rune(argCount+'0'))
		args = append(args, minMemory)
		argCount++
	}

	if maxRate != "" {
		query += ` AND gc.hourly_rate <= $` + string(rune(argCount+'0'))
		args = append(args, maxRate)
		argCount++
	}

	if gpuModel != "" {
		query += ` AND gm.name ILIKE $` + string(rune(argCount+'0'))
		args = append(args, "%"+gpuModel+"%")
	}

	query += ` ORDER BY gc.hourly_rate ASC LIMIT 100`

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list GPUs")
		return
	}
	defer rows.Close()

	var gpus []map[string]interface{}
	for rows.Next() {
		var gpu struct {
			ID                string
			GPUMemory         int
			CPUCores          int
			RAMSize           int
			StorageSize       int
			HourlyRate        float64
			IsAvailable       bool
			GPUModel          string
			ComputeCapability string
			ProviderName      string
			Rating            float64
		}
		rows.Scan(&gpu.ID, &gpu.GPUMemory, &gpu.CPUCores, &gpu.RAMSize,
			&gpu.StorageSize, &gpu.HourlyRate, &gpu.IsAvailable,
			&gpu.GPUModel, &gpu.ComputeCapability, &gpu.ProviderName, &gpu.Rating)

		gpus = append(gpus, map[string]interface{}{
			"id":                 gpu.ID,
			"gpu_model":          gpu.GPUModel,
			"gpu_memory_gb":      gpu.GPUMemory,
			"cpu_cores":          gpu.CPUCores,
			"ram_size_gb":        gpu.RAMSize,
			"storage_size_gb":    gpu.StorageSize,
			"hourly_rate":        gpu.HourlyRate,
			"compute_capability": gpu.ComputeCapability,
			"provider_name":      gpu.ProviderName,
			"provider_rating":    gpu.Rating,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"gpus":  gpus,
		"count": len(gpus),
	})
}

// UpdateAvailability updates GPU availability
func (h *ProviderHandler) UpdateAvailability(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		CapabilityID string `json:"capability_id"`
		IsAvailable  bool   `json:"is_available"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Update availability
	result, err := h.db.ExecContext(ctx, `
		UPDATE gpu_capabilities gc
		SET is_available = $1, updated_at = $2
		FROM providers p
		WHERE gc.id = $3 AND gc.provider_id = p.id AND p.user_id = $4
	`, req.IsAvailable, time.Now(), req.CapabilityID, userID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update availability")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "GPU capability not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Availability updated",
		"is_available": req.IsAvailable,
	})
}

// GetProviderStats gets provider statistics
func (h *ProviderHandler) GetProviderStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get provider ID
	var providerID string
	err := h.db.QueryRowContext(ctx, `
		SELECT id FROM providers WHERE user_id = $1
	`, userID).Scan(&providerID)

	if err != nil {
		respondError(w, http.StatusNotFound, "Provider not found")
		return
	}

	// Get stats
	var stats struct {
		TotalEarnings   float64
		TotalJobs       int
		ActiveJobs      int
		AverageRating   float64
		TotalReviews    int
		GPUCount        int
		AvailableGPUs   int
	}

	h.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) as total_earnings
		FROM provider_payouts
		WHERE provider_id = $1 AND status = 'completed'
	`, providerID).Scan(&stats.TotalEarnings)

	h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) as total_jobs
		FROM rental_sessions
		WHERE provider_id = $1
	`, providerID).Scan(&stats.TotalJobs)

	h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) as active_jobs
		FROM rental_sessions
		WHERE provider_id = $1 AND status = 'active'
	`, providerID).Scan(&stats.ActiveJobs)

	h.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(rating), 0) as avg_rating, COUNT(*) as total_reviews
		FROM provider_reviews
		WHERE provider_id = $1
	`, providerID).Scan(&stats.AverageRating, &stats.TotalReviews)

	h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) as gpu_count,
		       SUM(CASE WHEN is_available THEN 1 ELSE 0 END) as available_gpus
		FROM gpu_capabilities
		WHERE provider_id = $1
	`, providerID).Scan(&stats.GPUCount, &stats.AvailableGPUs)

	respondJSON(w, http.StatusOK, stats)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

