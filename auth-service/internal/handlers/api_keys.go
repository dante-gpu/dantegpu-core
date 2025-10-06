package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateAPIKeyRequest represents API key creation request
type CreateAPIKeyRequest struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description,omitempty"`
	Scopes      []string `json:"scopes" validate:"required"`
	ExpiresIn   int      `json:"expires_in,omitempty"` // days, 0 = never
}

// APIKeyResponse represents API key response
type APIKeyResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Key         string    `json:"key,omitempty"` // Only returned on creation
	KeyPrefix   string    `json:"key_prefix"`
	Description string    `json:"description"`
	Scopes      []string  `json:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

// APIKeyHandler handles API key operations
type APIKeyHandler struct {
	db *sql.DB
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(db *sql.DB) *APIKeyHandler {
	return &APIKeyHandler{db: db}
}

// CreateAPIKey creates a new API key
func (h *APIKeyHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate scopes
	if len(req.Scopes) == 0 {
		respondError(w, http.StatusBadRequest, "At least one scope is required")
		return
	}

	// Generate API key
	apiKey, keyHash, keyPrefix, err := h.generateAPIKey()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate API key")
		return
	}

	// Calculate expiration
	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		exp := time.Now().AddDate(0, 0, req.ExpiresIn)
		expiresAt = &exp
	}

	// Insert API key
	keyID := uuid.New().String()
	now := time.Now()

	_, err = h.db.ExecContext(ctx, `
		INSERT INTO api_keys (
			id, user_id, name, description, key_hash, key_prefix,
			scopes, expires_at, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, keyID, userID, req.Name, req.Description, keyHash, keyPrefix,
		strings.Join(req.Scopes, ","), expiresAt, true, now, now)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create API key")
		return
	}

	// Log audit event
	go h.logAuditEvent(context.Background(), userID, "api_key.created", keyID, r)

	respondJSON(w, http.StatusCreated, APIKeyResponse{
		ID:          keyID,
		Name:        req.Name,
		Key:         apiKey, // Only shown once
		KeyPrefix:   keyPrefix,
		Description: req.Description,
		Scopes:      req.Scopes,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
	})
}

// ListAPIKeys lists all API keys for the user
func (h *APIKeyHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	rows, err := h.db.QueryContext(ctx, `
		SELECT id, name, description, key_prefix, scopes, expires_at, 
		       is_active, created_at, last_used_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list API keys")
		return
	}
	defer rows.Close()

	var keys []APIKeyResponse
	for rows.Next() {
		var key APIKeyResponse
		var scopes string
		var lastUsedAt sql.NullTime

		err := rows.Scan(&key.ID, &key.Name, &key.Description, &key.KeyPrefix,
			&scopes, &key.ExpiresAt, &key.CreatedAt, &lastUsedAt)
		if err != nil {
			continue
		}

		key.Scopes = strings.Split(scopes, ",")
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}

		keys = append(keys, key)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"keys":  keys,
		"count": len(keys),
	})
}

// RevokeAPIKey revokes an API key
func (h *APIKeyHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		KeyID string `json:"key_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Revoke key
	result, err := h.db.ExecContext(ctx, `
		UPDATE api_keys
		SET is_active = false, revoked_at = $1, updated_at = $1
		WHERE id = $2 AND user_id = $3
	`, time.Now(), req.KeyID, userID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to revoke API key")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "API key not found")
		return
	}

	// Log audit event
	go h.logAuditEvent(context.Background(), userID, "api_key.revoked", req.KeyID, r)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "API key revoked successfully",
	})
}

// ValidateAPIKey validates an API key (used by middleware)
func (h *APIKeyHandler) ValidateAPIKey(ctx context.Context, apiKey string) (string, []string, error) {
	// Extract prefix
	parts := strings.Split(apiKey, "_")
	if len(parts) < 2 {
		return "", nil, ErrInvalidAPIKey
	}
	prefix := parts[0] + "_" + parts[1]

	// Get key by prefix
	var keyHash, userID, scopes string
	var expiresAt sql.NullTime
	var isActive bool

	err := h.db.QueryRowContext(ctx, `
		SELECT key_hash, user_id, scopes, expires_at, is_active
		FROM api_keys
		WHERE key_prefix = $1
	`, prefix).Scan(&keyHash, &userID, &scopes, &expiresAt, &isActive)

	if err == sql.ErrNoRows {
		return "", nil, ErrInvalidAPIKey
	}
	if err != nil {
		return "", nil, err
	}

	// Check if active
	if !isActive {
		return "", nil, ErrAPIKeyRevoked
	}

	// Check expiration
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		return "", nil, ErrAPIKeyExpired
	}

	// Verify key hash
	if err := bcrypt.CompareHashAndPassword([]byte(keyHash), []byte(apiKey)); err != nil {
		return "", nil, ErrInvalidAPIKey
	}

	// Update last used
	go h.updateLastUsed(context.Background(), prefix)

	return userID, strings.Split(scopes, ","), nil
}

// Helper functions

func (h *APIKeyHandler) generateAPIKey() (key, hash, prefix string, err error) {
	// Generate random bytes
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", err
	}

	// Create key with prefix
	keySecret := base64.URLEncoding.EncodeToString(b)
	prefix = "dgpu_" + keySecret[:8]
	key = prefix + "_" + keySecret[8:]

	// Hash the full key
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return "", "", "", err
	}

	return key, string(hashBytes), prefix, nil
}

func (h *APIKeyHandler) updateLastUsed(ctx context.Context, prefix string) {
	h.db.ExecContext(ctx, `
		UPDATE api_keys
		SET last_used_at = $1
		WHERE key_prefix = $2
	`, time.Now(), prefix)
}

func (h *APIKeyHandler) logAuditEvent(ctx context.Context, userID, action, resourceID string, r *http.Request) {
	h.db.ExecContext(ctx, `
		INSERT INTO audit_logs (user_id, actor_type, action, resource_type, resource_id, ip_address, user_agent, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, userID, "user", action, "api_key", resourceID, getIPAddress(r), r.UserAgent(), "success", time.Now())
}

