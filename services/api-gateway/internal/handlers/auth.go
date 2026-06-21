package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dante-gpu/dante-backend/api-gateway/internal/auth"
	"github.com/dante-gpu/dante-backend/api-gateway/internal/config"
	"go.uber.org/zap"
)

// AuthHandler holds dependencies for authentication handlers.
// I need the logger and config (for JWT secret/expiration).
type AuthHandler struct {
	Logger *zap.Logger
	Config *config.Config
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(logger *zap.Logger, cfg *config.Config) *AuthHandler {
	return &AuthHandler{Logger: logger, Config: cfg}
}

// LoginRequest defines the structure for the login request body.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse defines the structure for the login response body.
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
}

// Login handles user login requests.
// It validates credentials and issues a JWT upon success.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	// I need to decode the request body.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("Failed to decode login request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// I should validate the input (basic validation here).
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// I need to find the user (using the mock function for now).
	user, found := auth.FindUserByUsername(req.Username)
	if !found {
		h.Logger.Warn("Login attempt for non-existent user", zap.String("username", req.Username))
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// I must compare the password (plain text comparison - insecure!).
	// In a real app, I'd use bcrypt.CompareHashAndPassword.
	if user.Password != req.Password {
		h.Logger.Warn("Incorrect password attempt", zap.String("username", req.Username))
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// If credentials are valid, I should generate a JWT.
	tokenString, expiresAt, err := auth.GenerateJWT(user, h.Config.JwtSecret, h.Config.JwtExpiration)
	if err != nil {
		h.Logger.Error("Failed to generate JWT", zap.Error(err))
		http.Error(w, "Failed to process login", http.StatusInternalServerError)
		return
	}

	// I need to build the response.
	resp := LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
	}

	// I should send the response back as JSON.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.Logger.Error("Failed to encode login response", zap.Error(err))
		// Header already sent, so can't send http.Error
	}
}

// RegisterRequest defines the structure for the registration request body.
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"` // Allowing role specification for simplicity, maybe restrict this.
}

// RegisterResponse defines the structure for the registration response body.
type RegisterResponse struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

// Register handles new user registration requests.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	// I need to decode the request body.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("Failed to decode register request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// I should perform basic validation.
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}
	// Maybe validate the role?
	if req.Role == "" {
		req.Role = "user" // Default role
	}

	// Create a new user object (in reality, hash the password first!)
	newUser := &auth.User{
		Username: req.Username,
		Password: req.Password, // Store plain text temporarily (INSECURE)
		Role:     req.Role,
	}

	// Attempt to add the user (using mock function)
	err := auth.AddUser(newUser)
	if err != nil {
		h.Logger.Warn("Failed to register user", zap.String("username", req.Username), zap.Error(err))
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict) // 409 Conflict
		} else {
			http.Error(w, "Failed to register user", http.StatusInternalServerError)
		}
		return
	}

	// Respond with success message.
	resp := RegisterResponse{
		Message: "User registered successfully",
		UserID:  newUser.ID, // ID generated by AddUser
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.Logger.Error("Failed to encode register response", zap.Error(err))
	}
}

// ProfileResponse defines the structure for the profile response body.
type ProfileResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Profile handles requests to get the current user's profile.
// This requires an authentication middleware to run first.
func (h *AuthHandler) Profile(w http.ResponseWriter, r *http.Request) {
	// I expect the authentication middleware to have placed the claims in the context.
	claims, ok := r.Context().Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok || claims == nil {
		h.Logger.Error("Claims not found in context for profile request")
		// This should ideally not happen if the middleware is correctly applied.
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// I can build the response directly from the claims.
	resp := ProfileResponse{
		ID:       claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.Logger.Error("Failed to encode profile response", zap.Error(err))
	}
}
