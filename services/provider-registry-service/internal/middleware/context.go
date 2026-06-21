package middleware

import (
	"net/http"

	"github.com/dante-gpu/dante-backend/provider-registry-service/internal/logging"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// CorrelationID is middleware that injects a correlation ID into the context
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if we already have a correlation ID from a header
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			// Generate a new correlation ID if not present
			correlationID = uuid.New().String()
		}

		// Add the correlation ID to response headers
		w.Header().Set("X-Correlation-ID", correlationID)

		// Store the correlation ID in the request context
		ctx := logging.WithCorrelationID(r.Context(), correlationID)

		// Get request ID from chi middleware if it exists
		if requestID := middleware.GetReqID(r.Context()); requestID != "" {
			ctx = logging.WithRequestID(ctx, requestID)
		}

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
