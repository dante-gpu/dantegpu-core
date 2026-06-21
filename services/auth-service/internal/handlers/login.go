package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dantegpu/auth-service/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest represents the login payload
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Remember bool   `json:"remember,omitempty"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserInfo  `json:"user"`
}

// UserInfo represents user information in response
type UserInfo struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	Username         string    `json:"username"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	Roles            []string  `json:"roles"`
	IsVerified       bool      `json:"is_verified"`
	SubscriptionTier string    `json:"subscription_tier"`
	CreatedAt        time.Time `json:"created_at"`
}

// RefreshTokenRequest represents the refresh token payload
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest represents the logout payload
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

// LoginHandler handles user authentication
type LoginHandler struct {
	db           *sql.DB
	jwtManager   *jwt.Manager
	emailService EmailService
	config       *LoginConfig
}

// LoginConfig holds login configuration
type LoginConfig struct {
	MaxLoginAttempts      int
	LockoutDuration       time.Duration
	SessionTimeout        time.Duration
	RememberMeDuration    time.Duration
	RequireEmailVerified  bool
	EnableLoginAlerts     bool
	Enable2FA             bool
}

// NewLoginHandler creates a new login handler
func NewLoginHandler(db *sql.DB, jwtManager *jwt.Manager, emailService EmailService, config *LoginConfig) *LoginHandler {
	return &LoginHandler{
		db:           db,
		jwtManager:   jwtManager,
		emailService: emailService,
		config:       config,
	}
}

// Login handles user login
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate email format
	if !isValidEmail(req.Email) {
		respondError(w, http.StatusBadRequest, "Invalid email format")
		return
	}

	email := strings.ToLower(req.Email)

	// Check if account is locked
	locked, err := h.isAccountLocked(ctx, email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check account status")
		return
	}
	if locked {
		respondError(w, http.StatusForbidden, "Account is locked due to too many failed login attempts")
		return
	}

	// Get user from database
	var user struct {
		ID           string
		Email        string
		Username     string
		FirstName    string
		LastName     string
		PasswordHash string
		IsActive     bool
		IsVerified   bool
		Subscription string
		CreatedAt    time.Time
	}

	err = h.db.QueryRowContext(ctx, `
		SELECT id, email, username, first_name, last_name, password_hash, 
		       is_active, is_verified, subscription_tier, created_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID, &user.Email, &user.Username, &user.FirstName, &user.LastName,
		&user.PasswordHash, &user.IsActive, &user.IsVerified, &user.Subscription, &user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		// Record failed attempt (don't reveal if user exists)
		h.recordLoginAttempt(ctx, email, false, getIPAddress(r), r.UserAgent())
		respondError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to authenticate")
		return
	}

	// Check if user is active
	if !user.IsActive {
		respondError(w, http.StatusForbidden, "Account is inactive")
		return
	}

	// Check if email is verified (if required)
	if h.config.RequireEmailVerified && !user.IsVerified {
		respondError(w, http.StatusForbidden, "Email not verified. Please check your email for verification link.")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// Record failed attempt
		h.recordLoginAttempt(ctx, email, false, getIPAddress(r), r.UserAgent())
		respondError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Check if 2FA is enabled
	has2FA, err := h.has2FAEnabled(ctx, user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check 2FA status")
		return
	}

	if has2FA && h.config.Enable2FA {
		// Generate 2FA session token and return it
		// Client will need to verify 2FA code before getting actual tokens
		sessionToken, err := h.create2FASession(ctx, user.ID, getIPAddress(r), r.UserAgent())
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to create 2FA session")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"requires_2fa":    true,
			"session_token":   sessionToken,
			"message":         "Please enter your 2FA code",
		})
		return
	}

	// Get user roles
	roles, err := h.getUserRoles(ctx, user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user roles")
		return
	}

	// Generate JWT tokens
	tokenTTL := h.config.SessionTimeout
	if req.Remember {
		tokenTTL = h.config.RememberMeDuration
	}

	// Temporarily set the JWT manager's access TTL
	tokenPair, err := h.jwtManager.GenerateTokenPair(user.ID, user.Email, roles)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	// Create session in database
	sessionID, err := h.createSession(ctx, user.ID, tokenPair.RefreshToken, getIPAddress(r), r.UserAgent(), tokenTTL)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Record successful login
	h.recordLoginAttempt(ctx, email, true, getIPAddress(r), r.UserAgent())

	// Send login alert email (async)
	if h.config.EnableLoginAlerts {
		go h.sendLoginAlert(context.Background(), user.Email, user.FirstName, getIPAddress(r))
	}

	// Log audit event
	go h.logAuditEvent(context.Background(), user.ID, "user.login", email, r)

	// Respond with tokens and user info
	response := LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    int(tokenTTL.Seconds()),
		ExpiresAt:    tokenPair.ExpiresAt,
		User: UserInfo{
			ID:               user.ID,
			Email:            user.Email,
			Username:         user.Username,
			FirstName:        user.FirstName,
			LastName:         user.LastName,
			Roles:            roles,
			IsVerified:       user.IsVerified,
			SubscriptionTier: user.Subscription,
			CreatedAt:        user.CreatedAt,
		},
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenTTL.Seconds()),
	})

	respondJSON(w, http.StatusOK, response)
}

// RefreshToken handles token refresh
func (h *LoginHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate refresh token
	claims, err := h.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		if err == jwt.ErrExpiredToken {
			respondError(w, http.StatusUnauthorized, "Refresh token has expired")
		} else {
			respondError(w, http.StatusUnauthorized, "Invalid refresh token")
		}
		return
	}

	// Check if session exists and is valid
	valid, err := h.isSessionValid(ctx, claims.UserID, req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to validate session")
		return
	}
	if !valid {
		respondError(w, http.StatusUnauthorized, "Session has been revoked")
		return
	}

	// Check if user is still active
	var isActive bool
	err = h.db.QueryRowContext(ctx, `SELECT is_active FROM users WHERE id = $1`, claims.UserID).Scan(&isActive)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check user status")
		return
	}
	if !isActive {
		respondError(w, http.StatusForbidden, "Account is inactive")
		return
	}

	// Generate new token pair
	tokenPair, err := h.jwtManager.RefreshTokens(req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to refresh tokens")
		return
	}

	// Update session with new refresh token
	err = h.updateSession(ctx, claims.UserID, req.RefreshToken, tokenPair.RefreshToken)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update session")
		return
	}

	// Log audit event
	go h.logAuditEvent(context.Background(), claims.UserID, "user.token_refreshed", claims.Email, r)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"token_type":    tokenPair.TokenType,
		"expires_in":    int(h.config.SessionTimeout.Seconds()),
		"expires_at":    tokenPair.ExpiresAt,
	})
}

// Logout handles user logout
func (h *LoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req LogoutRequest
	json.NewDecoder(r.Body).Decode(&req)

	// Get user ID from context (set by auth middleware)
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Revoke session
	if req.RefreshToken != "" {
		err := h.revokeSession(ctx, userID, req.RefreshToken)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to logout")
			return
		}
	} else {
		// Revoke all sessions for user
		err := h.revokeAllSessions(ctx, userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to logout")
			return
		}
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	// Log audit event
	go h.logAuditEvent(context.Background(), userID, "user.logout", "", r)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// Helper functions continue in next part...

