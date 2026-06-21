package handlers

import (
	"net/http"

	"github.com/dante-gpu/dante-backend/api-gateway/internal/auth"
	"github.com/dante-gpu/dante-backend/api-gateway/internal/config"
	"github.com/dante-gpu/dante-backend/api-gateway/internal/upstream"
	"go.uber.org/zap"
)

// AuthHandler holds dependencies for authentication handlers.
// The gateway no longer stores any credentials; it proxies to the auth-service,
// which owns the PostgreSQL + bcrypt user store and issues the JWT.
type AuthHandler struct {
	Logger *zap.Logger
	Config *config.Config
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(logger *zap.Logger, cfg *config.Config) *AuthHandler {
	return &AuthHandler{Logger: logger, Config: cfg}
}

// Login proxies authentication to the auth-service.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	upstream.Forward(w, r, upstream.BuildURL(h.Config.AuthServiceURL, "/login"), h.Logger)
}

// Register proxies new-user registration to the auth-service, which hashes the
// password and persists the user.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	upstream.Forward(w, r, upstream.BuildURL(h.Config.AuthServiceURL, "/register"), h.Logger)
}

// Profile returns the current user's profile. The Authenticator middleware has
// already validated the JWT and placed the claims in context; the gateway then
// asks the auth-service for the full user record.
func (h *AuthHandler) Profile(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok || claims == nil {
		h.Logger.Error("Claims not found in context for profile request")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	url := upstream.BuildURL(h.Config.AuthServiceURL, "/profile")
	upstream.GetWithHeaders(w, r.Context(), url, map[string]string{
		"X-User-ID":     claims.UserID,
		"Authorization": r.Header.Get("Authorization"),
	}, h.Logger)
}
