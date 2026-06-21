package logging

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ContextKey is a type for context keys to avoid collisions
type ContextKey string

const (
	// CorrelationIDKey is the key used to store and retrieve correlation IDs from context
	CorrelationIDKey ContextKey = "correlation_id"

	// RequestIDKey is the key used to store and retrieve request IDs from context
	RequestIDKey ContextKey = "request_id"
)

// WithCorrelationID returns a new context with the correlation ID set
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// WithRequestID returns a new context with the request ID set
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// NewCorrelationID generates a new correlation ID if one doesn't exist
// and returns a context with the correlation ID set
func NewCorrelationID(ctx context.Context) (context.Context, string) {
	if id := GetCorrelationID(ctx); id != "" {
		return ctx, id
	}
	id := uuid.New().String()
	return WithCorrelationID(ctx, id), id
}

// GetCorrelationID retrieves the correlation ID from the context
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// EnrichLoggerWithContext creates a new logger with correlation ID and request ID fields
// added from the context
func EnrichLoggerWithContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	contextFields := []zapcore.Field{}

	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		contextFields = append(contextFields, zap.String("correlation_id", correlationID))
	}

	if requestID := GetRequestID(ctx); requestID != "" {
		contextFields = append(contextFields, zap.String("request_id", requestID))
	}

	if len(contextFields) > 0 {
		return logger.With(contextFields...)
	}

	return logger
}

// ContextLogger provides a logger that includes context information
type ContextLogger struct {
	base *zap.Logger
}

// NewContextLogger creates a new context-aware logger
func NewContextLogger(base *zap.Logger) *ContextLogger {
	return &ContextLogger{base: base}
}

// With returns a ContextLogger with the specified fields added
func (cl *ContextLogger) With(fields ...zapcore.Field) *ContextLogger {
	return &ContextLogger{base: cl.base.With(fields...)}
}

// FromContext creates a logger with context fields added
func (cl *ContextLogger) FromContext(ctx context.Context) *zap.Logger {
	return EnrichLoggerWithContext(ctx, cl.base)
}

// DBOperation logs a database operation with details
func (cl *ContextLogger) DBOperation(ctx context.Context, operation string, query string, args ...interface{}) *zap.Logger {
	logger := cl.FromContext(ctx).With(
		zap.String("db_operation", operation),
	)

	// Only log query in debug mode to avoid sensitive data in logs
	if ce := logger.Check(zapcore.DebugLevel, fmt.Sprintf("DB %s", operation)); ce != nil {
		// Avoid logging query and args at info level or higher
		return logger.With(
			zap.String("query", query),
			zap.Any("args", args),
		)
	}

	return logger
}
