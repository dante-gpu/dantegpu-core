package retryer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

// DatabaseRetryConfig holds configuration for database operation retries
type DatabaseRetryConfig struct {
	MaxAttempts      int           // Maximum number of retry attempts
	InitialDelay     time.Duration // Initial delay between retries
	MaxDelay         time.Duration // Maximum delay between retries
	BackoffFactor    float64       // Multiplicative factor for backoff
	JitterPercentage float64       // Random jitter percentage to add (0-1)
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() DatabaseRetryConfig {
	return DatabaseRetryConfig{
		MaxAttempts:      3,
		InitialDelay:     100 * time.Millisecond,
		MaxDelay:         2 * time.Second,
		BackoffFactor:    2.0,
		JitterPercentage: 0.2,
	}
}

// IsTransientError determines if an error is a transient database error
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common database transient errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Check PostgreSQL error classes
		// Connection exception error class
		if pgErr.Code[:2] == "08" {
			return true
		}

		// Operational error class
		if pgErr.Code[:2] == "57" {
			return true
		}

		// System errors
		if pgErr.Code[:2] == "53" {
			return true
		}

		// Deadlock detected
		if pgErr.Code == "40P01" {
			return true
		}

		// Serialization failure
		if pgErr.Code == "40001" {
			return true
		}
	}

	// Check for connection issues
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "connection") &&
		(strings.Contains(errMsg, "reset") ||
			strings.Contains(errMsg, "closed") ||
			strings.Contains(errMsg, "refused") ||
			strings.Contains(errMsg, "timeout"))
}

// WithRetry executes a database operation with configurable retry policy
func WithRetry(ctx context.Context, logger *zap.Logger, config DatabaseRetryConfig, operation string, fn func() error) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute the operation
		err := fn()
		if err == nil {
			return nil // Success, no need to retry
		}

		lastErr = err

		// If it's not a transient error or we've reached max attempts, return immediately
		if !IsTransientError(err) || attempt == config.MaxAttempts {
			if attempt > 1 {
				logger.Warn("Operation failed after retries",
					zap.String("operation", operation),
					zap.Int("attempts", attempt),
					zap.Error(err))
			}
			return fmt.Errorf("%s: %w", operation, err)
		}

		// Calculate jitter
		jitter := time.Duration(float64(delay) * config.JitterPercentage * (0.5 + (float64(attempt) / float64(config.MaxAttempts))))

		// Apply backoff with jitter
		sleepTime := delay + jitter

		logger.Warn("Retrying database operation due to transient error",
			zap.String("operation", operation),
			zap.Int("attempt", attempt),
			zap.Duration("retry_delay", sleepTime),
			zap.Error(err))

		// Check if context has been cancelled before sleeping
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s cancelled: %w", operation, ctx.Err())
		case <-time.After(sleepTime):
			// Continue with next attempt
		}

		// Increase delay for next attempt (with max limit)
		delay = time.Duration(float64(delay) * config.BackoffFactor)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	// This should never be reached due to the return in the loop, but just in case
	return fmt.Errorf("%s: %w", operation, lastErr)
}
