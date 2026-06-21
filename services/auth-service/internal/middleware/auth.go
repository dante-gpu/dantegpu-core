package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/dantegpu/auth-service/pkg/jwt"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	jwtManager *jwt.Manager
	db         *sql.DB
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtManager *jwt.Manager, db *sql.DB) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
		db:         db,
	}
}

// Authenticate validates JWT token and adds user info to context
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := m.jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			if err == jwt.ErrExpiredToken {
				respondError(w, http.StatusUnauthorized, "Token has expired")
			} else {
				respondError(w, http.StatusUnauthorized, "Invalid token")
			}
			return
		}

		// Check if token is revoked
		revoked, err := m.isTokenRevoked(r.Context(), claims.TokenID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to validate token")
			return
		}
		if revoked {
			respondError(w, http.StatusUnauthorized, "Token has been revoked")
			return
		}

		// Check if user is still active
		active, err := m.isUserActive(r.Context(), claims.UserID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to validate user")
			return
		}
		if !active {
			respondError(w, http.StatusForbidden, "User account is inactive")
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "email", claims.Email)
		ctx = context.WithValue(ctx, "roles", claims.Roles)
		ctx = context.WithValue(ctx, "token_id", claims.TokenID)

		// Update session activity
		go m.updateSessionActivity(context.Background(), claims.UserID, tokenString)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole checks if user has a specific role
func (m *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, ok := r.Context().Value("roles").([]string)
			if !ok {
				respondError(w, http.StatusForbidden, "Forbidden")
				return
			}

			hasRole := false
			for _, r := range roles {
				if r == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				respondError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole checks if user has any of the specified roles
func (m *AuthMiddleware) RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles, ok := r.Context().Value("roles").([]string)
			if !ok {
				respondError(w, http.StatusForbidden, "Forbidden")
				return
			}

			hasRole := false
			for _, role := range roles {
				for _, userRole := range userRoles {
					if userRole == role {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				respondError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission checks if user has a specific permission
func (m *AuthMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value("user_id").(string)
			if !ok {
				respondError(w, http.StatusForbidden, "Forbidden")
				return
			}

			hasPermission, err := m.userHasPermission(r.Context(), userID, permission)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "Failed to check permissions")
				return
			}

			if !hasPermission {
				respondError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth validates token if present but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			next.ServeHTTP(w, r)
			return
		}

		tokenString := parts[1]
		claims, err := m.jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "email", claims.Email)
		ctx = context.WithValue(ctx, "roles", claims.Roles)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper functions

func (m *AuthMiddleware) isTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	// Check if token is in revoked tokens table
	var exists bool
	err := m.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM revoked_tokens
			WHERE token_id = $1 AND expires_at > NOW()
		)
	`, tokenID).Scan(&exists)
	
	return exists, err
}

func (m *AuthMiddleware) isUserActive(ctx context.Context, userID string) (bool, error) {
	var isActive bool
	err := m.db.QueryRowContext(ctx, `
		SELECT is_active FROM users WHERE id = $1
	`, userID).Scan(&isActive)
	
	return isActive, err
}

func (m *AuthMiddleware) userHasPermission(ctx context.Context, userID, permission string) (bool, error) {
	var hasPermission bool
	err := m.db.QueryRowContext(ctx, `
		SELECT user_has_permission($1, $2)
	`, userID, permission).Scan(&hasPermission)
	
	return hasPermission, err
}

func (m *AuthMiddleware) updateSessionActivity(ctx context.Context, userID, token string) {
	m.db.ExecContext(ctx, `
		UPDATE active_sessions
		SET last_activity = $1
		WHERE user_id = $2 AND revoked_at IS NULL
	`, time.Now(), userID)
}

// RateLimitMiddleware implements rate limiting
type RateLimitMiddleware struct {
	db *sql.DB
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(db *sql.DB) *RateLimitMiddleware {
	return &RateLimitMiddleware{db: db}
}

// RateLimit applies rate limiting based on IP address
func (m *RateLimitMiddleware) RateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ipAddress := getIPAddress(r)
			
			// Check rate limit
			var count int
			err := m.db.QueryRowContext(r.Context(), `
				SELECT COUNT(*)
				FROM api_requests
				WHERE ip_address = $1
				AND created_at > NOW() - INTERVAL '1 minute'
			`, ipAddress).Scan(&count)
			
			if err == nil && count >= requestsPerMinute {
				respondError(w, http.StatusTooManyRequests, "Rate limit exceeded")
				return
			}
			
			// Record request
			go m.recordRequest(context.Background(), ipAddress, r.URL.Path)
			
			next.ServeHTTP(w, r)
		})
	}
}

func (m *RateLimitMiddleware) recordRequest(ctx context.Context, ipAddress, path string) {
	m.db.ExecContext(ctx, `
		INSERT INTO api_requests (ip_address, path, created_at)
		VALUES ($1, $2, $3)
	`, ipAddress, path, time.Now())
}

// Utility functions

func getIPAddress(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + message + `"}`))
}

