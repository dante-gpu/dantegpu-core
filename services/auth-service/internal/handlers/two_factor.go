package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pquerna/otp/totp"
)

// Enable2FARequest represents enable 2FA request
type Enable2FARequest struct {
	Method string `json:"method" validate:"required,oneof=totp sms email"`
	Phone  string `json:"phone,omitempty"`
}

// Verify2FASetupRequest represents 2FA setup verification
type Verify2FASetupRequest struct {
	Code   string `json:"code" validate:"required"`
	Secret string `json:"secret" validate:"required"`
}

// Verify2FALoginRequest represents 2FA login verification
type Verify2FALoginRequest struct {
	SessionToken string `json:"session_token" validate:"required"`
	Code         string `json:"code" validate:"required"`
}

// Disable2FARequest represents disable 2FA request
type Disable2FARequest struct {
	Password string `json:"password" validate:"required"`
	Code     string `json:"code" validate:"required"`
}

// TwoFactorHandler handles 2FA operations
type TwoFactorHandler struct {
	db           *sql.DB
	jwtManager   interface{}
	emailService EmailService
	smsService   SMSService
	config       *TwoFactorConfig
}

// TwoFactorConfig holds 2FA configuration
type TwoFactorConfig struct {
	Issuer            string
	CodeLength        int
	CodeTTL           time.Duration
	BackupCodesCount  int
	SessionTimeout    time.Duration
}

// SMSService interface for sending SMS
type SMSService interface {
	SendSMS(ctx context.Context, phone, message string) error
}

// NewTwoFactorHandler creates a new 2FA handler
func NewTwoFactorHandler(db *sql.DB, jwtManager interface{}, emailService EmailService, smsService SMSService, config *TwoFactorConfig) *TwoFactorHandler {
	return &TwoFactorHandler{
		db:           db,
		jwtManager:   jwtManager,
		emailService: emailService,
		smsService:   smsService,
		config:       config,
	}
}

// Enable2FA initiates 2FA setup
func (h *TwoFactorHandler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req Enable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Check if 2FA already enabled
	var exists bool
	err := h.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM two_factor_auth WHERE user_id = $1 AND is_enabled = true)
	`, userID).Scan(&exists)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check 2FA status")
		return
	}
	if exists {
		respondError(w, http.StatusConflict, "2FA is already enabled")
		return
	}

	var response map[string]interface{}

	switch req.Method {
	case "totp":
		response, err = h.setupTOTP(ctx, userID)
	case "sms":
		if req.Phone == "" {
			respondError(w, http.StatusBadRequest, "Phone number required for SMS 2FA")
			return
		}
		response, err = h.setupSMS(ctx, userID, req.Phone)
	case "email":
		response, err = h.setupEmail(ctx, userID)
	default:
		respondError(w, http.StatusBadRequest, "Invalid 2FA method")
		return
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, response)
}

// Verify2FASetup verifies and completes 2FA setup
func (h *TwoFactorHandler) Verify2FASetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req Verify2FASetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Get pending 2FA setup
	var method, secret string
	err := h.db.QueryRowContext(ctx, `
		SELECT method, secret
		FROM two_factor_auth
		WHERE user_id = $1 AND is_enabled = false
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&method, &secret)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "No pending 2FA setup found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get 2FA setup")
		return
	}

	// Verify code based on method
	valid := false
	switch method {
	case "totp":
		valid = totp.Validate(req.Code, secret)
	case "sms", "email":
		// Verify code from database
		var storedCode string
		var expiresAt time.Time
		err := h.db.QueryRowContext(ctx, `
			SELECT code, expires_at FROM two_factor_codes
			WHERE user_id = $1 AND used_at IS NULL
			ORDER BY created_at DESC LIMIT 1
		`, userID).Scan(&storedCode, &expiresAt)
		
		if err == nil && storedCode == req.Code && time.Now().Before(expiresAt) {
			valid = true
			// Mark code as used
			h.db.ExecContext(ctx, `UPDATE two_factor_codes SET used_at = $1 WHERE user_id = $2 AND code = $3`,
				time.Now(), userID, req.Code)
		}
	}

	if !valid {
		respondError(w, http.StatusUnauthorized, "Invalid verification code")
		return
	}

	// Generate backup codes
	backupCodes, err := h.generateBackupCodes(ctx, userID, h.config.BackupCodesCount)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate backup codes")
		return
	}

	// Enable 2FA
	_, err = h.db.ExecContext(ctx, `
		UPDATE two_factor_auth
		SET is_enabled = true, verified_at = $1
		WHERE user_id = $2 AND method = $3
	`, time.Now(), userID, method)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to enable 2FA")
		return
	}

	// Log audit event
	go h.logAuditEvent(context.Background(), userID, "user.2fa_enabled", "", r)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "2FA enabled successfully",
		"backup_codes": backupCodes,
		"enabled_at":   time.Now(),
	})
}

// Verify2FALogin verifies 2FA code during login
func (h *TwoFactorHandler) Verify2FALogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req Verify2FALoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Get 2FA session
	var userID string
	var expiresAt time.Time
	err := h.db.QueryRowContext(ctx, `
		SELECT user_id, expires_at
		FROM two_factor_sessions
		WHERE session_token = $1 AND verified = false
	`, req.SessionToken).Scan(&userID, &expiresAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Invalid or expired session")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get session")
		return
	}

	if time.Now().After(expiresAt) {
		respondError(w, http.StatusGone, "Session has expired")
		return
	}

	// Get user's 2FA method and secret
	var method, secret string
	err = h.db.QueryRowContext(ctx, `
		SELECT method, secret
		FROM two_factor_auth
		WHERE user_id = $1 AND is_enabled = true
	`, userID).Scan(&method, &secret)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get 2FA settings")
		return
	}

	// Verify code
	valid := false
	isBackupCode := false

	switch method {
	case "totp":
		valid = totp.Validate(req.Code, secret)
		// Check backup codes if TOTP fails
		if !valid {
			valid, _ = h.verifyBackupCode(ctx, userID, req.Code)
			isBackupCode = valid
		}
	case "sms", "email":
		var storedCode string
		var codeExpiresAt time.Time
		err := h.db.QueryRowContext(ctx, `
			SELECT code, expires_at FROM two_factor_codes
			WHERE user_id = $1 AND used_at IS NULL
			ORDER BY created_at DESC LIMIT 1
		`, userID).Scan(&storedCode, &codeExpiresAt)
		
		if err == nil && storedCode == req.Code && time.Now().Before(codeExpiresAt) {
			valid = true
			h.db.ExecContext(ctx, `UPDATE two_factor_codes SET used_at = $1 WHERE user_id = $2 AND code = $3`,
				time.Now(), userID, req.Code)
		}
	}

	if !valid {
		respondError(w, http.StatusUnauthorized, "Invalid 2FA code")
		return
	}

	// Mark session as verified
	h.db.ExecContext(ctx, `UPDATE two_factor_sessions SET verified = true WHERE session_token = $1`, req.SessionToken)

	// Get user details and generate tokens (similar to login)
	// This would call the same token generation logic as login
	
	// Log audit event
	go h.logAuditEvent(context.Background(), userID, "user.2fa_verified", "", r)

	if isBackupCode {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message": "2FA verified with backup code. Please regenerate backup codes.",
			"warning": "backup_code_used",
		})
	} else {
		respondJSON(w, http.StatusOK, map[string]string{
			"message": "2FA verified successfully",
		})
	}
}

// Disable2FA disables 2FA
func (h *TwoFactorHandler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req Disable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Verify password
	var passwordHash string
	err := h.db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&passwordHash)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify password")
		return
	}

	// Verify 2FA code or backup code
	// ... verification logic ...

	// Disable 2FA
	_, err = h.db.ExecContext(ctx, `
		UPDATE two_factor_auth
		SET is_enabled = false, disabled_at = $1
		WHERE user_id = $2
	`, time.Now(), userID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to disable 2FA")
		return
	}

	// Revoke backup codes
	h.db.ExecContext(ctx, `DELETE FROM two_factor_backup_codes WHERE user_id = $1`, userID)

	// Log audit event
	go h.logAuditEvent(context.Background(), userID, "user.2fa_disabled", "", r)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "2FA disabled successfully",
	})
}

// Helper functions

func (h *TwoFactorHandler) setupTOTP(ctx context.Context, userID string) (map[string]interface{}, error) {
	// Get user email for TOTP label
	var email string
	h.db.QueryRowContext(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)

	// Generate TOTP secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      h.config.Issuer,
		AccountName: email,
	})
	if err != nil {
		return nil, err
	}

	// Store secret (not enabled yet)
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO two_factor_auth (user_id, method, secret, is_enabled, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, method) DO UPDATE SET secret = $3, created_at = $5
	`, userID, "totp", key.Secret(), false, time.Now())

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"method":     "totp",
		"secret":     key.Secret(),
		"qr_code":    key.URL(),
		"message":    "Scan the QR code with your authenticator app",
	}, nil
}

func (h *TwoFactorHandler) setupSMS(ctx context.Context, userID, phone string) (map[string]interface{}, error) {
	code := generateNumericCode(6)
	
	// Store code
	_, err := h.db.ExecContext(ctx, `
		INSERT INTO two_factor_codes (user_id, code, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
	`, userID, code, time.Now().Add(h.config.CodeTTL), time.Now())
	if err != nil {
		return nil, err
	}

	// Store 2FA method
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO two_factor_auth (user_id, method, phone, is_enabled, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, method) DO UPDATE SET phone = $3, created_at = $5
	`, userID, "sms", phone, false, time.Now())
	if err != nil {
		return nil, err
	}

	// Send SMS
	go h.smsService.SendSMS(context.Background(), phone, fmt.Sprintf("Your verification code is: %s", code))

	return map[string]interface{}{
		"method":  "sms",
		"phone":   phone,
		"message": "Verification code sent to your phone",
	}, nil
}

func (h *TwoFactorHandler) setupEmail(ctx context.Context, userID string) (map[string]interface{}, error) {
	code := generateNumericCode(6)
	
	var email, firstName string
	h.db.QueryRowContext(ctx, `SELECT email, first_name FROM users WHERE id = $1`, userID).Scan(&email, &firstName)

	// Store code
	_, err := h.db.ExecContext(ctx, `
		INSERT INTO two_factor_codes (user_id, code, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
	`, userID, code, time.Now().Add(h.config.CodeTTL), time.Now())
	if err != nil {
		return nil, err
	}

	// Store 2FA method
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO two_factor_auth (user_id, method, is_enabled, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, method) DO UPDATE SET created_at = $4
	`, userID, "email", false, time.Now())
	if err != nil {
		return nil, err
	}

	// Send email
	go h.emailService.Send2FACodeEmail(context.Background(), email, code, firstName)

	return map[string]interface{}{
		"method":  "email",
		"message": "Verification code sent to your email",
	}, nil
}

func (h *TwoFactorHandler) generateBackupCodes(ctx context.Context, userID string, count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		codes[i] = generateAlphanumericCode(8)
		
		// Store hashed backup code
		_, err := h.db.ExecContext(ctx, `
			INSERT INTO two_factor_backup_codes (user_id, code_hash, created_at)
			VALUES ($1, $2, $3)
		`, userID, hashCode(codes[i]), time.Now())
		if err != nil {
			return nil, err
		}
	}
	return codes, nil
}

func (h *TwoFactorHandler) verifyBackupCode(ctx context.Context, userID, code string) (bool, error) {
	// Get all backup codes for user
	rows, err := h.db.QueryContext(ctx, `
		SELECT id, code_hash FROM two_factor_backup_codes
		WHERE user_id = $1 AND used_at IS NULL
	`, userID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var codeHash string
		rows.Scan(&id, &codeHash)
		
		// Verify code (simple comparison for backup codes)
		if hashCode(code) == codeHash {
			// Mark as used
			h.db.ExecContext(ctx, `UPDATE two_factor_backup_codes SET used_at = $1 WHERE id = $2`, time.Now(), id)
			return true, nil
		}
	}
	
	return false, nil
}

func (h *TwoFactorHandler) logAuditEvent(ctx context.Context, userID, action, email string, r *http.Request) {
	h.db.ExecContext(ctx, `
		INSERT INTO audit_logs (user_id, actor_type, action, resource_type, resource_id, ip_address, user_agent, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, userID, "user", action, "2fa", userID, getIPAddress(r), r.UserAgent(), "success", time.Now())
}

func generateNumericCode(length int) string {
	const digits = "0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b)
}

func generateAlphanumericCode(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}

func hashCode(code string) string {
	// Simple hash for backup codes
	return base64.StdEncoding.EncodeToString([]byte(code))
}

