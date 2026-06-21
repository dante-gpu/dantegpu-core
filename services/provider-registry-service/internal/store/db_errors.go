package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

// Common database error codes
const (
	PgUniqueViolation     = "23505" // unique_violation
	PgForeignKeyViolation = "23503" // foreign_key_violation
	PgConnectionException = "08000" // connection_exception
	PgConnectionFailure   = "08006" // connection_failure
)

// Custom error types
var (
	ErrDBConnection = errors.New("database connection error")
	ErrDBConstraint = errors.New("database constraint violation")
	ErrDBTimeout    = errors.New("database operation timeout")
	ErrDBCanceled   = errors.New("database operation canceled")
)

// IsTransientError determines if an error is likely transient and can be retried
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context cancellation or timeout
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false // These aren't transient in the sense that retrying with the same context won't help
	}

	// Check for pgx-specific error types
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Check error codes that indicate transient errors
		switch pgErr.Code {
		case PgConnectionException, PgConnectionFailure:
			return true
		}
		// Constraint violations are not transient
		return false
	}

	// Check for connectivity errors
	if errors.Is(err, pgx.ErrTxClosed) {
		return true
	}

	// Other potentially transient errors
	return false
}

// WithRetry executes a database operation with retries for transient errors
func WithRetry(ctx context.Context, logger *zap.Logger, operation string, retryCount int, retryDelay time.Duration, fn func() error) error {
	var err error
	var attempt int

	for attempt = 1; attempt <= retryCount; attempt++ {
		operationErr := fn()
		if operationErr == nil {
			return nil // Success
		}

		err = operationErr // Store the last error

		// If it's not a transient error, don't retry
		if !IsTransientError(err) {
			logger.Error("Non-transient database error",
				zap.String("operation", operation),
				zap.Error(err),
				zap.Int("attempt", attempt),
			)
			return fmt.Errorf("%w: %v", mapToCustomError(err), err)
		}

		// Check if context is still valid
		if ctx.Err() != nil {
			logger.Warn("Context canceled during database retry",
				zap.String("operation", operation),
				zap.Error(ctx.Err()),
				zap.Int("attempt", attempt),
			)
			return ctx.Err()
		}

		// Log the retry attempt
		logger.Warn("Retrying transient database error",
			zap.String("operation", operation),
			zap.Error(err),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", retryCount),
			zap.Duration("delay_before_retry", retryDelay),
		)

		// Wait before retrying, using a simple exponential backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay * time.Duration(attempt)):
			// Continue to next attempt
		}
	}

	// If we get here, all retries failed
	logger.Error("Database operation failed after all retry attempts",
		zap.String("operation", operation),
		zap.Error(err),
		zap.Int("attempts", attempt-1),
	)

	return fmt.Errorf("after %d attempts: %w", attempt-1, mapToCustomError(err))
}

// mapToCustomError maps database errors to our custom error types
func mapToCustomError(err error) error {
	if err == nil {
		return nil
	}

	// Check for context errors
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", ErrDBTimeout, err)
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%w: %v", ErrDBCanceled, err)
	}

	// Check for pgx-specific error types
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case PgUniqueViolation, PgForeignKeyViolation:
			return fmt.Errorf("%w: %v", ErrDBConstraint, err)
		case PgConnectionException, PgConnectionFailure:
			return fmt.Errorf("%w: %v", ErrDBConnection, err)
		}
	}

	// Connection-related errors
	if errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("%w: %v", ErrDBConnection, err)
	}

	// Default case: just return the original error
	return err
}
