package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// RegistrationRequest represents the user registration payload
type RegistrationRequest struct {
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
	FirstName       string `json:"first_name" validate:"required,min=2,max=100"`
	LastName        string `json:"last_name" validate:"required,min=2,max=100"`
	Phone           string `json:"phone,omitempty"`
	Organization    string `json:"organization,omitempty"`
	AcceptTerms     bool   `json:"accept_terms" validate:"required"`
}

// RegistrationResponse represents the registration response
type RegistrationResponse struct {
	UserID            string    `json:"user_id"`
	Email             string    `json:"email"`
	VerificationSent  bool      `json:"verification_sent"`
	Message           string    `json:"message"`
	CreatedAt         time.Time `json:"created_at"`
}

// VerifyEmailRequest represents email verification payload
type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// ResendVerificationRequest represents resend verification payload
type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// RegistrationHandler handles user registration
type RegistrationHandler struct {
	db           *sql.DB
	emailService EmailService
	config       *RegistrationConfig
}

// RegistrationConfig holds registration configuration
type RegistrationConfig struct {
	RequireEmailVerification bool
	PasswordMinLength        int
	PasswordRequireUppercase bool
	PasswordRequireLowercase bool
	PasswordRequireNumber    bool
	PasswordRequireSpecial   bool
	BcryptCost              int
	VerificationTokenTTL    time.Duration
	EnableRegistration      bool
}

// EmailService interface for sending emails
type EmailService interface {
	SendVerificationEmail(ctx context.Context, email, token, firstName string) error
	SendWelcomeEmail(ctx context.Context, email, firstName string) error
	SendPasswordResetEmail(ctx context.Context, email, token, firstName string) error
	SendPasswordChangedEmail(ctx context.Context, email, firstName string) error
	Send2FACodeEmail(ctx context.Context, email, code, firstName string) error
	SendLoginAlertEmail(ctx context.Context, email, firstName, ipAddress, location string) error
}

// NewRegistrationHandler creates a new registration handler
func NewRegistrationHandler(db *sql.DB, emailService EmailService, config *RegistrationConfig) *RegistrationHandler {
	return &RegistrationHandler{
		db:           db,
		emailService: emailService,
		config:       config,
	}
}

// Register handles user registration
func (h *RegistrationHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if registration is enabled
	if !h.config.EnableRegistration {
		respondError(w, http.StatusForbidden, "Registration is currently disabled")
		return
	}

	// Parse request
	var req RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate request
	if err := h.validateRegistrationRequest(&req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if user already exists
	exists, err := h.userExists(ctx, req.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check user existence")
		return
	}
	if exists {
		respondError(w, http.StatusConflict, "User with this email already exists")
		return
	}

	// Hash password
	passwordHash, err := h.hashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	// Create user
	userID := uuid.New().String()
	username := h.generateUsername(req.Email)
	
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Insert user
	query := `
		INSERT INTO users (
			id, email, username, password_hash, first_name, last_name,
			phone, organization, is_active, is_verified, role,
			subscription_tier, credit_balance, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	
	now := time.Now()
	_, err = tx.ExecContext(ctx, query,
		userID,
		strings.ToLower(req.Email),
		username,
		passwordHash,
		req.FirstName,
		req.LastName,
		req.Phone,
		req.Organization,
		true,  // is_active
		false, // is_verified (will be verified via email)
		"user",
		"basic",
		0, // initial balance
		now,
		now,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Assign default user role
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_roles (user_id, role, granted_by, granted_at, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, "user", "system", now, true)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to assign user role")
		return
	}

	// Create user preferences
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_preferences (user_id, created_at, updated_at)
		VALUES ($1, $2, $3)
	`, userID, now, now)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create user preferences")
		return
	}

	// Generate verification token if email verification is required
	var verificationSent bool
	if h.config.RequireEmailVerification {
		token, tokenHash, err := h.generateVerificationToken()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate verification token")
			return
		}

		expiresAt := now.Add(h.config.VerificationTokenTTL)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO email_verification_tokens (user_id, email, token_hash, expires_at, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, req.Email, tokenHash, expiresAt, now)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to create verification token")
			return
		}

		// Send verification email (async, don't block registration)
		go func() {
			if err := h.emailService.SendVerificationEmail(context.Background(), req.Email, token, req.FirstName); err != nil {
				// Log error but don't fail registration
				// In production, use proper logging
			}
		}()
		
		verificationSent = true
	} else {
		// Mark user as verified immediately
		_, err = tx.ExecContext(ctx, `UPDATE users SET is_verified = true WHERE id = $1`, userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to verify user")
			return
		}

		// Send welcome email
		go func() {
			if err := h.emailService.SendWelcomeEmail(context.Background(), req.Email, req.FirstName); err != nil {
				// Log error
			}
		}()
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to complete registration")
		return
	}

	// Log registration event
	go h.logAuditEvent(context.Background(), userID, "user.registered", req.Email, r)

	// Respond
	response := RegistrationResponse{
		UserID:           userID,
		Email:            req.Email,
		VerificationSent: verificationSent,
		Message:          h.getRegistrationMessage(verificationSent),
		CreatedAt:        now,
	}

	respondJSON(w, http.StatusCreated, response)
}

// VerifyEmail handles email verification
func (h *RegistrationHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Token == "" {
		respondError(w, http.StatusBadRequest, "Verification token is required")
		return
	}

	// Hash the token to compare with stored hash
	tokenHash := h.hashToken(req.Token)

	// Find and validate token
	var userID, email string
	var expiresAt time.Time
	var verifiedAt sql.NullTime

	err := h.db.QueryRowContext(ctx, `
		SELECT user_id, email, expires_at, verified_at
		FROM email_verification_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(&userID, &email, &expiresAt, &verifiedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Invalid or expired verification token")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify token")
		return
	}

	// Check if already verified
	if verifiedAt.Valid {
		respondError(w, http.StatusConflict, "Email already verified")
		return
	}

	// Check if token expired
	if time.Now().After(expiresAt) {
		respondError(w, http.StatusGone, "Verification token has expired")
		return
	}

	// Start transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Mark user as verified
	now := time.Now()
	_, err = tx.ExecContext(ctx, `
		UPDATE users 
		SET is_verified = true, updated_at = $1
		WHERE id = $2
	`, now, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify user")
		return
	}

	// Mark token as used
	_, err = tx.ExecContext(ctx, `
		UPDATE email_verification_tokens
		SET verified_at = $1
		WHERE token_hash = $2
	`, now, tokenHash)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update token")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to complete verification")
		return
	}

	// Send welcome email
	var firstName string
	h.db.QueryRowContext(ctx, `SELECT first_name FROM users WHERE id = $1`, userID).Scan(&firstName)
	go h.emailService.SendWelcomeEmail(context.Background(), email, firstName)

	// Log verification event
	go h.logAuditEvent(context.Background(), userID, "user.email_verified", email, r)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":     "Email verified successfully",
		"verified_at": now,
	})
}

// ResendVerification resends verification email
func (h *RegistrationHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Get user
	var userID, firstName string
	var isVerified bool
	err := h.db.QueryRowContext(ctx, `
		SELECT id, first_name, is_verified
		FROM users
		WHERE email = $1 AND is_active = true
	`, strings.ToLower(req.Email)).Scan(&userID, &firstName, &isVerified)

	if err == sql.ErrNoRows {
		// Don't reveal if user exists
		respondJSON(w, http.StatusOK, map[string]string{
			"message": "If the email exists, a verification link has been sent",
		})
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process request")
		return
	}

	if isVerified {
		respondError(w, http.StatusConflict, "Email is already verified")
		return
	}

	// Generate new token
	token, tokenHash, err := h.generateVerificationToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	now := time.Now()
	expiresAt := now.Add(h.config.VerificationTokenTTL)

	// Insert new token
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO email_verification_tokens (user_id, email, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, req.Email, tokenHash, expiresAt, now)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create verification token")
		return
	}

	// Send email
	go h.emailService.SendVerificationEmail(context.Background(), req.Email, token, firstName)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Verification email sent successfully",
	})
}

// Helper functions

func (h *RegistrationHandler) validateRegistrationRequest(req *RegistrationRequest) error {
	// Validate email format
	if !isValidEmail(req.Email) {
		return ErrInvalidEmail
	}

	// Validate password match
	if req.Password != req.ConfirmPassword {
		return ErrPasswordMismatch
	}

	// Validate password strength
	if err := h.validatePasswordStrength(req.Password); err != nil {
		return err
	}

	// Validate terms acceptance
	if !req.AcceptTerms {
		return ErrTermsNotAccepted
	}

	return nil
}

func (h *RegistrationHandler) validatePasswordStrength(password string) error {
	if len(password) < h.config.PasswordMinLength {
		return ErrPasswordTooShort
	}

	if h.config.PasswordRequireUppercase && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return ErrPasswordNoUppercase
	}

	if h.config.PasswordRequireLowercase && !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return ErrPasswordNoLowercase
	}

	if h.config.PasswordRequireNumber && !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return ErrPasswordNoNumber
	}

	if h.config.PasswordRequireSpecial && !regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password) {
		return ErrPasswordNoSpecial
	}

	return nil
}

func (h *RegistrationHandler) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.config.BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (h *RegistrationHandler) userExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := h.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)
	`, strings.ToLower(email)).Scan(&exists)
	return exists, err
}

func (h *RegistrationHandler) generateUsername(email string) string {
	parts := strings.Split(email, "@")
	username := parts[0]
	// Add random suffix to ensure uniqueness
	suffix := uuid.New().String()[:8]
	return username + "_" + suffix
}

func (h *RegistrationHandler) generateVerificationToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.URLEncoding.EncodeToString(b)
	hash = h.hashToken(token)
	return token, hash, nil
}

func (h *RegistrationHandler) hashToken(token string) string {
	// Use bcrypt for token hashing
	hash, _ := bcrypt.GenerateFromPassword([]byte(token), bcrypt.MinCost)
	return string(hash)
}

func (h *RegistrationHandler) getRegistrationMessage(verificationSent bool) string {
	if verificationSent {
		return "Registration successful! Please check your email to verify your account."
	}
	return "Registration successful! You can now log in."
}

func (h *RegistrationHandler) logAuditEvent(ctx context.Context, userID, action, email string, r *http.Request) {
	// Log to audit_logs table
	h.db.ExecContext(ctx, `
		INSERT INTO audit_logs (user_id, actor_type, action, resource_type, resource_id, ip_address, user_agent, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, userID, "user", action, "user", userID, getIPAddress(r), r.UserAgent(), "success", time.Now())
}

// Utility functions

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func getIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

