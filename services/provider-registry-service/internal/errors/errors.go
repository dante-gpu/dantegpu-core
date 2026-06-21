package errors

import (
	"errors"
	"fmt"
)

// Standard error types that can be used for error checking
var (
	// ErrNotFound is returned when a requested resource doesn't exist
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists is returned when trying to create a resource that already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput is returned when the input data is invalid
	ErrInvalidInput = errors.New("invalid input data")

	// ErrDatabase is returned when a database operation fails
	ErrDatabase = errors.New("database operation failed")

	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("operation timed out")

	// ErrPermissionDenied is returned when the user doesn't have permission to perform an action
	ErrPermissionDenied = errors.New("permission denied")

	// ErrUnavailable is returned when a service is unavailable
	ErrUnavailable = errors.New("service unavailable")

	// ErrInternal is returned for unexpected internal errors
	ErrInternal = errors.New("internal error")
)

// ProviderError represents an error related to provider operations
type ProviderError struct {
	Op      string // Operation that failed (e.g., "GetProvider", "AddProvider")
	ID      string // ID of the provider (if applicable)
	Message string // Human-readable error message
	Err     error  // Underlying error
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s: provider ID %s: %s: %v", e.Op, e.ID, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
}

// Unwrap implements the errors.Unwrap interface
func (e *ProviderError) Unwrap() error {
	return e.Err
}

// Is implements the errors.Is interface
func (e *ProviderError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// NewProviderError creates a new ProviderError
func NewProviderError(op, id, message string, err error) *ProviderError {
	return &ProviderError{
		Op:      op,
		ID:      id,
		Message: message,
		Err:     err,
	}
}

// DatabaseError represents an error related to database operations
type DatabaseError struct {
	Op        string // Operation that failed (e.g., "Query", "Exec")
	Table     string // Table involved in the operation
	Query     string // SQL query (if applicable, may be redacted for security)
	Message   string // Human-readable error message
	Err       error  // Underlying error
	Transient bool   // Whether the error is transient and can be retried
}

// Error implements the error interface
func (e *DatabaseError) Error() string {
	if e.Query != "" {
		return fmt.Sprintf("%s on %s: %s: query: %s: %v", e.Op, e.Table, e.Message, e.Query, e.Err)
	}
	return fmt.Sprintf("%s on %s: %s: %v", e.Op, e.Table, e.Message, e.Err)
}

// Unwrap implements the errors.Unwrap interface
func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// Is implements the errors.Is interface
func (e *DatabaseError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// NewDatabaseError creates a new DatabaseError
func NewDatabaseError(op, table, query, message string, err error, transient bool) *DatabaseError {
	return &DatabaseError{
		Op:        op,
		Table:     table,
		Query:     query,
		Message:   message,
		Err:       err,
		Transient: transient,
	}
}

// IsNotFound returns true if the error or its cause is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsAlreadyExists returns true if the error or its cause is an already exists error
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// IsInvalidInput returns true if the error or its cause is an invalid input error
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsDatabase returns true if the error or its cause is a database error
func IsDatabase(err error) bool {
	return errors.Is(err, ErrDatabase)
}

// IsTimeout returns true if the error or its cause is a timeout error
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsPermissionDenied returns true if the error or its cause is a permission denied error
func IsPermissionDenied(err error) bool {
	return errors.Is(err, ErrPermissionDenied)
}

// IsUnavailable returns true if the error or its cause is a service unavailable error
func IsUnavailable(err error) bool {
	return errors.Is(err, ErrUnavailable)
}

// IsInternal returns true if the error or its cause is an internal error
func IsInternal(err error) bool {
	return errors.Is(err, ErrInternal)
}

// IsTransient returns true if the error is a transient error that can be retried
func IsTransient(err error) bool {
	var dbErr *DatabaseError
	if errors.As(err, &dbErr) {
		return dbErr.Transient
	}
	return false
}
