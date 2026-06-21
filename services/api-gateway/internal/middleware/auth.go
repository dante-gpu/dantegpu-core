package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/dante-gpu/dante-backend/api-gateway/internal/auth"
	"go.uber.org/zap"
)

// Authenticator provides a middleware for JWT authentication.
// It needs the logger and JWT secret key.
func Authenticator(logger *zap.Logger, jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// I need to get the Authorization header.
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.Warn("Missing Authorization header")
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// The header should be in the format "Bearer <token>".
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				logger.Warn("Invalid Authorization header format", zap.String("header", authHeader))
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// I need to validate the token.
			claims, err := auth.ValidateJWT(tokenString, jwtSecret)
			if err != nil {
				logger.Warn("Invalid JWT token", zap.Error(err))
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// If the token is valid, I should add the claims to the request context.
			ctx := context.WithValue(r.Context(), auth.ContextKeyClaims, claims)

			// I'll pass the request with the new context to the next handler.
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
