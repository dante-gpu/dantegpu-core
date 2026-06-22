package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// isAccountLocked checks if an account is locked due to failed login attempts
func (h *LoginHandler) isAccountLocked(ctx context.Context, email string) (bool, error) {
	var lockedUntil sql.NullTime
	
	err := h.db.QueryRowContext(ctx, `
		SELECT locked_until
		FROM login_attempts
		WHERE email = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, email).Scan(&lockedUntil)
	
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	
	if lockedUntil.Valid && lockedUntil.Time.After(time.Now()) {
		return true, nil
	}
	
	return false, nil
}

// recordLoginAttempt records a login attempt
func (h *LoginHandler) recordLoginAttempt(ctx context.Context, email string, success bool, ipAddress, userAgent string) error {
	// Count recent failed attempts
	var failedCount int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM login_attempts
		WHERE email = $1
		AND success = false
		AND created_at > NOW() - INTERVAL '15 minutes'
	`, email).Scan(&failedCount)
	
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	
	// Determine if account should be locked
	var lockedUntil *time.Time
	if !success && failedCount >= h.config.MaxLoginAttempts-1 {
		lockTime := time.Now().Add(h.config.LockoutDuration)
		lockedUntil = &lockTime
	}
	
	// Insert login attempt
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO login_attempts (email, success, ip_address, user_agent, locked_until, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, email, success, ipAddress, userAgent, lockedUntil, time.Now())
	
	return err
}

// getUserRoles gets all roles for a user
func (h *LoginHandler) getUserRoles(ctx context.Context, userID string) ([]string, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT role
		FROM user_roles
		WHERE user_id = $1 AND is_active = true
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	
	if len(roles) == 0 {
		roles = []string{"user"} // Default role
	}
	
	return roles, nil
}

// has2FAEnabled checks if user has 2FA enabled
func (h *LoginHandler) has2FAEnabled(ctx context.Context, userID string) (bool, error) {
	var enabled bool
	err := h.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM two_factor_auth
			WHERE user_id = $1 AND is_enabled = true
		)
	`, userID).Scan(&enabled)
	
	return enabled, err
}

// create2FASession creates a temporary session for 2FA verification
func (h *LoginHandler) create2FASession(ctx context.Context, userID, ipAddress, userAgent string) (string, error) {
	sessionToken := generateSecureToken()
	expiresAt := time.Now().Add(10 * time.Minute) // 10 minute expiry
	
	_, err := h.db.ExecContext(ctx, `
		INSERT INTO two_factor_sessions (user_id, session_token, ip_address, user_agent, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, sessionToken, ipAddress, userAgent, expiresAt, time.Now())
	
	if err != nil {
		return "", err
	}
	
	return sessionToken, nil
}

// createSession creates a new user session
func (h *LoginHandler) createSession(ctx context.Context, userID, refreshToken, ipAddress, userAgent string, duration time.Duration) (string, error) {
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(duration)
	
	_, err := h.db.ExecContext(ctx, `
		INSERT INTO active_sessions (
			id, user_id, refresh_token, ip_address, user_agent, 
			expires_at, last_activity, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, sessionID, userID, refreshToken, ipAddress, userAgent, expiresAt, time.Now(), time.Now())
	
	if err != nil {
		return "", err
	}
	
	return sessionID, nil
}

// isSessionValid checks if a session is valid
func (h *LoginHandler) isSessionValid(ctx context.Context, userID, refreshToken string) (bool, error) {
	var expiresAt time.Time
	var revokedAt sql.NullTime
	
	err := h.db.QueryRowContext(ctx, `
		SELECT expires_at, revoked_at
		FROM active_sessions
		WHERE user_id = $1 AND refresh_token = $2
	`, userID, refreshToken).Scan(&expiresAt, &revokedAt)
	
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	
	// Check if revoked
	if revokedAt.Valid {
		return false, nil
	}
	
	// Check if expired
	if time.Now().After(expiresAt) {
		return false, nil
	}
	
	return true, nil
}

// updateSession updates a session with new refresh token
func (h *LoginHandler) updateSession(ctx context.Context, userID, oldRefreshToken, newRefreshToken string) error {
	_, err := h.db.ExecContext(ctx, `
		UPDATE active_sessions
		SET refresh_token = $1, last_activity = $2
		WHERE user_id = $3 AND refresh_token = $4
	`, newRefreshToken, time.Now(), userID, oldRefreshToken)
	
	return err
}

// revokeSession revokes a specific session
func (h *LoginHandler) revokeSession(ctx context.Context, userID, refreshToken string) error {
	_, err := h.db.ExecContext(ctx, `
		UPDATE active_sessions
		SET revoked_at = $1
		WHERE user_id = $2 AND refresh_token = $3
	`, time.Now(), userID, refreshToken)
	
	return err
}

// revokeAllSessions revokes all sessions for a user
func (h *LoginHandler) revokeAllSessions(ctx context.Context, userID string) error {
	_, err := h.db.ExecContext(ctx, `
		UPDATE active_sessions
		SET revoked_at = $1
		WHERE user_id = $2 AND revoked_at IS NULL
	`, time.Now(), userID)
	
	return err
}

// sendLoginAlert sends a login alert email
func (h *LoginHandler) sendLoginAlert(ctx context.Context, email, firstName, ipAddress string) {
	// Get location from IP (in production, use a geolocation service)
	location := "Unknown Location"
	
	// Send email
	h.emailService.SendLoginAlertEmail(ctx, email, firstName, ipAddress, location)
}

// logAuditEvent logs an audit event
func (h *LoginHandler) logAuditEvent(ctx context.Context, userID, action, email string, r *http.Request) {
	h.db.ExecContext(ctx, `
		INSERT INTO audit_logs (
			user_id, actor_type, action, resource_type, resource_id,
			ip_address, user_agent, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, userID, "user", action, "session", userID, getIPAddress(r), r.UserAgent(), "success", time.Now())
}

// GetActiveSessions returns all active sessions for a user
func (h *LoginHandler) GetActiveSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context (set by auth middleware)
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	rows, err := h.db.QueryContext(ctx, `
		SELECT id, ip_address, user_agent, last_activity, created_at
		FROM active_sessions
		WHERE user_id = $1 
		AND revoked_at IS NULL 
		AND expires_at > NOW()
		ORDER BY last_activity DESC
	`, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get sessions")
		return
	}
	defer rows.Close()
	
	type Session struct {
		ID           string    `json:"id"`
		IPAddress    string    `json:"ip_address"`
		UserAgent    string    `json:"user_agent"`
		LastActivity time.Time `json:"last_activity"`
		CreatedAt    time.Time `json:"created_at"`
		IsCurrent    bool      `json:"is_current"`
	}
	
	var sessions []Session
	currentIP := getIPAddress(r)
	
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.IPAddress, &s.UserAgent, &s.LastActivity, &s.CreatedAt); err != nil {
			continue
		}
		s.IsCurrent = s.IPAddress == currentIP
		sessions = append(sessions, s)
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// RevokeSession revokes a specific session by ID
func (h *LoginHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	
	// Get session ID from request
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	
	// Revoke the session
	result, err := h.db.ExecContext(ctx, `
		UPDATE active_sessions
		SET revoked_at = $1
		WHERE id = $2 AND user_id = $3 AND revoked_at IS NULL
	`, time.Now(), req.SessionID, userID)
	
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to revoke session")
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Session not found")
		return
	}
	
	// Log audit event
	go h.logAuditEvent(context.Background(), userID, "user.session_revoked", "", r)
	
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Session revoked successfully",
	})
}

// generateSecureToken generates a secure random token
func generateSecureToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

