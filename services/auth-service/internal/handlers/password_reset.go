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

	"golang.org/x/crypto/bcrypt"
)

// RequestPasswordResetRequest represents password reset request payload
type RequestPasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest represents password reset payload
type ResetPasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
}

// ChangePasswordRequest represents password change payload
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
}

// PasswordResetHandler handles password reset operations
type PasswordResetHandler struct {
	db           *sql.DB
	emailService EmailService
	config       *PasswordResetConfig
}

// PasswordResetConfig holds password reset configuration
type PasswordResetConfig struct {
	TokenTTL              time.Duration
	ResetCooldown         time.Duration
	PasswordMinLength     int
	PasswordRequireUpper  bool
	PasswordRequireLower  bool
	PasswordRequireNumber bool
	PasswordRequireSpecial bool
	BcryptCost            int
}

// NewPasswordResetHandler creates a new password reset handler
func NewPasswordResetHandler(db *sql.DB, emailService EmailService, config *PasswordResetConfig) *PasswordResetHandler {
	return &PasswordResetHandler{
		db:           db,
		emailService: emailService,
		config:       config,
	}
}

// RequestPasswordReset initiates password reset process
func (h *PasswordResetHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RequestPasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if !isValidEmail(req.Email) {
		respondError(w, http.StatusBadRequest, "Invalid email format")
		return
	}

	email := strings.ToLower(req.Email)

	// Get user (don't reveal if user exists)
	var userID, firstName string
	var isActive bool
	err := h.db.QueryRowContext(ctx, `
		SELECT id, first_name, is_active
		FROM users
		WHERE email = $1
	`, email).Scan(&userID, &firstName, &isActive)

	if err == sql.ErrNoRows {
		// Don't reveal if user exists
		respondJSON(w, http.StatusOK, map[string]string{
			"message": "If the email exists, a password reset link has been sent",
		})
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process request")
		return
	}

	if !isActive {
		// Don't reveal account status
		respondJSON(w, http.StatusOK, map[string]string{
			"message": "If the email exists, a password reset link has been sent",
		})
		return
	}

	// Check cooldown period
	canRequest, err := h.canRequestReset(ctx, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process request")
		return
	}
	if !canRequest {
		respondError(w, http.StatusTooManyRequests, "Please wait before requesting another password reset")
		return
	}

	// Generate reset token
	token, tokenHash, err := h.generateResetToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate reset token")
		return
	}

	now := time.Now()
	expiresAt := now.Add(h.config.TokenTTL)

	// Store reset token
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO password_reset_tokens (
			user_id, email, token_hash, expires_at, 
			ip_address, user_agent, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, email, tokenHash, expiresAt, getIPAddress(r), r.UserAgent(), now)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create reset token")
		return
	}

	// Send reset email (async)
	go h.emailService.SendPasswordResetEmail(context.Background(), email, token, firstName)

	// Log audit event
	go h.logAuditEvent(context.Background(), userID, "user.password_reset_requested", email, r)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword resets password using reset token
func (h *PasswordResetHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate passwords match
	if req.Password != req.ConfirmPassword {
		respondError(w, http.StatusBadRequest, "Passwords do not match")
		return
	}

	// Validate password strength
	if err := h.validatePasswordStrength(req.Password); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Hash the token to compare with stored hash
	tokenHash := h.hashToken(req.Token)

	// Find and validate token
	var userID, email string
	var expiresAt time.Time
	var usedAt sql.NullTime

	err := h.db.QueryRowContext(ctx, `
		SELECT user_id, email, expires_at, used_at
		FROM password_reset_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(&userID, &email, &expiresAt, &usedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Invalid or expired reset token")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify token")
		return
	}

	// Check if already used
	if usedAt.Valid {
		respondError(w, http.StatusConflict, "Reset token has already been used")
		return
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		respondError(w, http.StatusGone, "Reset token has expired")
		return
	}

	// Hash new password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.config.BcryptCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	// Start transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Update password
	now := time.Now()
	_, err = tx.ExecContext(ctx, `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`, string(passwordHash), now, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	// Mark token as used
	_, err = tx.ExecContext(ctx, `
		UPDATE password_reset_tokens
		SET used_at = $1
		WHERE token_hash = $2
	`, now, tokenHash)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update token")
		return
	}

	// Revoke all active sessions for security
	_, err = tx.ExecContext(ctx, `
		UPDATE active_sessions
		SET revoked_at = $1
		WHERE user_id = $2 AND revoked_at IS NULL
	`, now, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to revoke sessions")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to complete password reset")
		return
	}

	// Send confirmation email
	var firstName string
	h.db.QueryRowContext(ctx, `SELECT first_name FROM users WHERE id = $1`, userID).Scan(&firstName)
	go h.emailService.SendPasswordChangedEmail(context.Background(), email, firstName)

	// Log audit event
	go h.logAuditEvent(context.Background(), userID, "user.password_reset", email, r)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Password reset successfully",
		"reset_at":   now,
	})
}

// ChangePassword changes password for authenticated user
func (h *PasswordResetHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by auth middleware)
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate passwords match
	if req.NewPassword != req.ConfirmPassword {
		respondError(w, http.StatusBadRequest, "Passwords do not match")
		return
	}

	// Validate password strength
	if err := h.validatePasswordStrength(req.NewPassword); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get current password hash
	var currentHash, email, firstName string
	err := h.db.QueryRowContext(ctx, `
		SELECT password_hash, email, first_name
		FROM users
		WHERE id = $1
	`, userID).Scan(&currentHash, &email, &firstName)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
		respondError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	// Check if new password is same as current
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.NewPassword)); err == nil {
		respondError(w, http.StatusBadRequest, "New password must be different from current password")
		return
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), h.config.BcryptCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	// Update password
	now := time.Now()
	_, err = h.db.ExecContext(ctx, `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`, string(newHash), now, userID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	// Send confirmation email
	go h.emailService.SendPasswordChangedEmail(context.Background(), email, firstName)

	// Log audit event
	go h.logAuditEvent(context.Background(), userID, "user.password_changed", email, r)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Password changed successfully",
		"changed_at": now,
	})
}

// Helper functions

func (h *PasswordResetHandler) canRequestReset(ctx context.Context, userID string) (bool, error) {
	var lastRequest sql.NullTime
	err := h.db.QueryRowContext(ctx, `
		SELECT MAX(created_at)
		FROM password_reset_tokens
		WHERE user_id = $1
	`, userID).Scan(&lastRequest)

	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	if !lastRequest.Valid {
		return true, nil
	}

	return time.Since(lastRequest.Time) > h.config.ResetCooldown, nil
}

func (h *PasswordResetHandler) generateResetToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.URLEncoding.EncodeToString(b)
	hash = h.hashToken(token)
	return token, hash, nil
}

func (h *PasswordResetHandler) hashToken(token string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(token), bcrypt.MinCost)
	return string(hash)
}

func (h *PasswordResetHandler) validatePasswordStrength(password string) error {
	if len(password) < h.config.PasswordMinLength {
		return ErrPasswordTooShort
	}
	if h.config.PasswordRequireUpper && !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return ErrPasswordNoUppercase
	}
	if h.config.PasswordRequireLower && !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		return ErrPasswordNoLowercase
	}
	if h.config.PasswordRequireNumber && !strings.ContainsAny(password, "0123456789") {
		return ErrPasswordNoNumber
	}
	if h.config.PasswordRequireSpecial && !strings.ContainsAny(password, "!@#$%^&*(),.?\":{}|<>") {
		return ErrPasswordNoSpecial
	}
	return nil
}

func (h *PasswordResetHandler) logAuditEvent(ctx context.Context, userID, action, email string, r *http.Request) {
	h.db.ExecContext(ctx, `
		INSERT INTO audit_logs (
			user_id, actor_type, action, resource_type, resource_id,
			ip_address, user_agent, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, userID, "user", action, "user", userID, getIPAddress(r), r.UserAgent(), "success", time.Now())
}

