package models

import (
	"errors"
	"fmt"
)

// Common billing and payment errors
var (
	// Wallet errors
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrWalletAlreadyExists = errors.New("wallet already exists")
	ErrWalletInactive      = errors.New("wallet is inactive")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrInvalidWalletType   = errors.New("invalid wallet type")
	ErrInvalidSolanaAddress = errors.New("invalid Solana address")

	// Transaction errors
	ErrTransactionNotFound    = errors.New("transaction not found")
	ErrTransactionFailed      = errors.New("transaction failed")
	ErrTransactionCancelled   = errors.New("transaction cancelled")
	ErrInvalidTransactionType = errors.New("invalid transaction type")
	ErrInvalidAmount          = errors.New("invalid amount")
	ErrTransactionTimeout     = errors.New("transaction timeout")

	// Session errors
	ErrSessionNotFound        = errors.New("session not found")
	ErrSessionAlreadyActive   = errors.New("session already active")
	ErrSessionNotActive       = errors.New("session not active")
	ErrInvalidSessionStatus   = errors.New("invalid session status")
	ErrSessionExpired         = errors.New("session expired")
	ErrMaxSessionDuration     = errors.New("maximum session duration exceeded")

	// Provider errors
	ErrProviderNotFound       = errors.New("provider not found")
	ErrProviderNotAvailable   = errors.New("provider not available")
	ErrInsufficientVRAM       = errors.New("insufficient VRAM available")
	ErrInvalidGPUModel        = errors.New("invalid GPU model")

	// Pricing errors
	ErrInvalidPricing         = errors.New("invalid pricing configuration")
	ErrRateNotFound           = errors.New("rate not found for GPU model")
	ErrPricingCalculation     = errors.New("pricing calculation error")

	// Solana blockchain errors
	ErrSolanaConnection       = errors.New("Solana connection error")
	ErrSolanaTransaction      = errors.New("Solana transaction error")
	ErrInvalidSignature       = errors.New("invalid transaction signature")
	ErrTokenTransferFailed    = errors.New("token transfer failed")

	// Billing errors
	ErrBillingCalculation     = errors.New("billing calculation error")
	ErrBillingRecordNotFound  = errors.New("billing record not found")
	ErrUsageRecordNotFound    = errors.New("usage record not found")

	// Payout errors
	ErrPayoutNotFound         = errors.New("payout not found")
	ErrPayoutFailed           = errors.New("payout failed")
	ErrMinimumPayoutAmount    = errors.New("amount below minimum payout threshold")
	ErrPayoutAlreadyProcessed = errors.New("payout already processed")

	// Validation errors
	ErrValidationFailed       = errors.New("validation failed")
	ErrMissingRequiredField   = errors.New("missing required field")
	ErrInvalidFieldValue      = errors.New("invalid field value")

	// Database errors
	ErrDatabaseConnection     = errors.New("database connection error")
	ErrDatabaseQuery          = errors.New("database query error")
	ErrDatabaseTransaction    = errors.New("database transaction error")

	// Configuration errors
	ErrInvalidConfiguration   = errors.New("invalid configuration")
	ErrMissingConfiguration   = errors.New("missing configuration")

	// Rate limiting errors
	ErrRateLimitExceeded      = errors.New("rate limit exceeded")
	ErrTooManyRequests        = errors.New("too many requests")

	// Security errors
	ErrUnauthorized           = errors.New("unauthorized")
	ErrForbidden              = errors.New("forbidden")
	ErrInvalidAPIKey          = errors.New("invalid API key")
	ErrSecurityViolation      = errors.New("security violation")
)

// BillingError represents a structured error with additional context
type BillingError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Cause   error                  `json:"-"`
}

// Error implements the error interface
func (e *BillingError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *BillingError) Unwrap() error {
	return e.Cause
}

// NewBillingError creates a new BillingError
func NewBillingError(code, message string, cause error) *BillingError {
	return &BillingError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Details: make(map[string]interface{}),
	}
}

// WithDetail adds a detail to the error
func (e *BillingError) WithDetail(key string, value interface{}) *BillingError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// Error codes for structured error handling
const (
	// Wallet error codes
	ErrCodeWalletNotFound      = "WALLET_NOT_FOUND"
	ErrCodeWalletExists        = "WALLET_ALREADY_EXISTS"
	ErrCodeWalletInactive      = "WALLET_INACTIVE"
	ErrCodeInsufficientFunds   = "INSUFFICIENT_FUNDS"
	ErrCodeInvalidWalletType   = "INVALID_WALLET_TYPE"
	ErrCodeInvalidSolanaAddr   = "INVALID_SOLANA_ADDRESS"

	// Transaction error codes
	ErrCodeTransactionNotFound = "TRANSACTION_NOT_FOUND"
	ErrCodeTransactionFailed   = "TRANSACTION_FAILED"
	ErrCodeTransactionCancelled = "TRANSACTION_CANCELLED"
	ErrCodeInvalidTxnType      = "INVALID_TRANSACTION_TYPE"
	ErrCodeInvalidAmount       = "INVALID_AMOUNT"
	ErrCodeTransactionTimeout  = "TRANSACTION_TIMEOUT"

	// Session error codes
	ErrCodeSessionNotFound     = "SESSION_NOT_FOUND"
	ErrCodeSessionActive       = "SESSION_ALREADY_ACTIVE"
	ErrCodeSessionNotActive    = "SESSION_NOT_ACTIVE"
	ErrCodeInvalidSessionStatus = "INVALID_SESSION_STATUS"
	ErrCodeSessionExpired      = "SESSION_EXPIRED"
	ErrCodeMaxSessionDuration  = "MAX_SESSION_DURATION"

	// Provider error codes
	ErrCodeProviderNotFound    = "PROVIDER_NOT_FOUND"
	ErrCodeProviderUnavailable = "PROVIDER_NOT_AVAILABLE"
	ErrCodeInsufficientVRAM    = "INSUFFICIENT_VRAM"
	ErrCodeInvalidGPUModel     = "INVALID_GPU_MODEL"

	// Pricing error codes
	ErrCodeInvalidPricing      = "INVALID_PRICING"
	ErrCodeRateNotFound        = "RATE_NOT_FOUND"
	ErrCodePricingCalculation  = "PRICING_CALCULATION_ERROR"

	// Solana error codes
	ErrCodeSolanaConnection    = "SOLANA_CONNECTION_ERROR"
	ErrCodeSolanaTransaction   = "SOLANA_TRANSACTION_ERROR"
	ErrCodeInvalidSignature    = "INVALID_SIGNATURE"
	ErrCodeTokenTransferFailed = "TOKEN_TRANSFER_FAILED"

	// Billing error codes
	ErrCodeBillingCalculation  = "BILLING_CALCULATION_ERROR"
	ErrCodeBillingNotFound     = "BILLING_RECORD_NOT_FOUND"
	ErrCodeUsageNotFound       = "USAGE_RECORD_NOT_FOUND"

	// Payout error codes
	ErrCodePayoutNotFound      = "PAYOUT_NOT_FOUND"
	ErrCodePayoutFailed        = "PAYOUT_FAILED"
	ErrCodeMinimumPayout       = "MINIMUM_PAYOUT_AMOUNT"
	ErrCodePayoutProcessed     = "PAYOUT_ALREADY_PROCESSED"

	// Validation error codes
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
	ErrCodeMissingField        = "MISSING_REQUIRED_FIELD"
	ErrCodeInvalidField        = "INVALID_FIELD_VALUE"

	// System error codes
	ErrCodeDatabaseError       = "DATABASE_ERROR"
	ErrCodeConfigError         = "CONFIGURATION_ERROR"
	ErrCodeRateLimited         = "RATE_LIMITED"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeSecurityViolation   = "SECURITY_VIOLATION"
)

// Common error constructors
func NewWalletNotFoundError(walletID string) *BillingError {
	return NewBillingError(ErrCodeWalletNotFound, "Wallet not found", ErrWalletNotFound).
		WithDetail("wallet_id", walletID)
}

func NewInsufficientFundsError(required, available string) *BillingError {
	return NewBillingError(ErrCodeInsufficientFunds, "Insufficient funds", ErrInsufficientFunds).
		WithDetail("required", required).
		WithDetail("available", available)
}

func NewSessionNotFoundError(sessionID string) *BillingError {
	return NewBillingError(ErrCodeSessionNotFound, "Session not found", ErrSessionNotFound).
		WithDetail("session_id", sessionID)
}

func NewProviderNotAvailableError(providerID string) *BillingError {
	return NewBillingError(ErrCodeProviderUnavailable, "Provider not available", ErrProviderNotAvailable).
		WithDetail("provider_id", providerID)
}

func NewTransactionFailedError(txnID string, cause error) *BillingError {
	return NewBillingError(ErrCodeTransactionFailed, "Transaction failed", cause).
		WithDetail("transaction_id", txnID)
}

func NewValidationError(field, message string) *BillingError {
	return NewBillingError(ErrCodeValidationFailed, "Validation failed", ErrValidationFailed).
		WithDetail("field", field).
		WithDetail("message", message)
}

func NewSolanaError(operation string, cause error) *BillingError {
	return NewBillingError(ErrCodeSolanaTransaction, "Solana operation failed", cause).
		WithDetail("operation", operation)
}

func NewDatabaseError(operation string, cause error) *BillingError {
	return NewBillingError(ErrCodeDatabaseError, "Database operation failed", cause).
		WithDetail("operation", operation)
}
