package handlers

import "errors"

// Authentication errors
var (
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrUserNotFound         = errors.New("user not found")
	ErrUserInactive         = errors.New("user account is inactive")
	ErrUserNotVerified      = errors.New("email not verified")
	ErrAccountLocked        = errors.New("account is locked due to too many failed login attempts")
	ErrInvalidToken         = errors.New("invalid or expired token")
	ErrTokenExpired         = errors.New("token has expired")
	ErrSessionExpired       = errors.New("session has expired")
	ErrUnauthorized         = errors.New("unauthorized access")
	ErrForbidden            = errors.New("forbidden - insufficient permissions")
)

// Registration errors
var (
	ErrInvalidEmail         = errors.New("invalid email format")
	ErrEmailAlreadyExists   = errors.New("email already registered")
	ErrPasswordMismatch     = errors.New("passwords do not match")
	ErrPasswordTooShort     = errors.New("password must be at least 8 characters")
	ErrPasswordNoUppercase  = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLowercase  = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoNumber     = errors.New("password must contain at least one number")
	ErrPasswordNoSpecial    = errors.New("password must contain at least one special character")
	ErrTermsNotAccepted     = errors.New("you must accept the terms and conditions")
	ErrRegistrationDisabled = errors.New("registration is currently disabled")
)

// Verification errors
var (
	ErrVerificationTokenInvalid = errors.New("invalid verification token")
	ErrVerificationTokenExpired = errors.New("verification token has expired")
	ErrAlreadyVerified          = errors.New("email is already verified")
)

// Password reset errors
var (
	ErrResetTokenInvalid = errors.New("invalid password reset token")
	ErrResetTokenExpired = errors.New("password reset token has expired")
	ErrResetTokenUsed    = errors.New("password reset token has already been used")
)

// Two-factor authentication errors
var (
	ErrInvalid2FACode     = errors.New("invalid 2FA code")
	Err2FANotEnabled      = errors.New("2FA is not enabled for this account")
	Err2FAAlreadyEnabled  = errors.New("2FA is already enabled for this account")
	ErrInvalidBackupCode  = errors.New("invalid backup code")
)

// API key errors
var (
	ErrInvalidAPIKey     = errors.New("invalid API key")
	ErrAPIKeyExpired     = errors.New("API key has expired")
	ErrAPIKeyRevoked     = errors.New("API key has been revoked")
	ErrAPIKeyRateLimit   = errors.New("API key rate limit exceeded")
)

// OAuth errors
var (
	ErrOAuthProviderNotSupported = errors.New("OAuth provider not supported")
	ErrOAuthStateMismatch        = errors.New("OAuth state mismatch")
	ErrOAuthCodeInvalid          = errors.New("invalid OAuth authorization code")
	ErrOAuthTokenExchange        = errors.New("failed to exchange OAuth code for token")
)

// Validation errors
var (
	ErrInvalidInput       = errors.New("invalid input data")
	ErrMissingField       = errors.New("required field is missing")
	ErrInvalidFieldFormat = errors.New("invalid field format")
)

// Rate limiting errors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrTooManyRequests   = errors.New("too many requests")
)

// Database errors
var (
	ErrDatabaseConnection = errors.New("database connection error")
	ErrDatabaseQuery      = errors.New("database query error")
	ErrTransactionFailed  = errors.New("transaction failed")
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// Error codes for client-side handling
const (
	CodeInvalidCredentials   = "INVALID_CREDENTIALS"
	CodeUserNotFound         = "USER_NOT_FOUND"
	CodeUserInactive         = "USER_INACTIVE"
	CodeUserNotVerified      = "USER_NOT_VERIFIED"
	CodeAccountLocked        = "ACCOUNT_LOCKED"
	CodeInvalidToken         = "INVALID_TOKEN"
	CodeTokenExpired         = "TOKEN_EXPIRED"
	CodeSessionExpired       = "SESSION_EXPIRED"
	CodeUnauthorized         = "UNAUTHORIZED"
	CodeForbidden            = "FORBIDDEN"
	CodeInvalidEmail         = "INVALID_EMAIL"
	CodeEmailAlreadyExists   = "EMAIL_ALREADY_EXISTS"
	CodePasswordMismatch     = "PASSWORD_MISMATCH"
	CodePasswordTooWeak      = "PASSWORD_TOO_WEAK"
	CodeTermsNotAccepted     = "TERMS_NOT_ACCEPTED"
	CodeRegistrationDisabled = "REGISTRATION_DISABLED"
	CodeVerificationInvalid  = "VERIFICATION_INVALID"
	CodeVerificationExpired  = "VERIFICATION_EXPIRED"
	CodeAlreadyVerified      = "ALREADY_VERIFIED"
	CodeResetTokenInvalid    = "RESET_TOKEN_INVALID"
	CodeResetTokenExpired    = "RESET_TOKEN_EXPIRED"
	CodeInvalid2FACode       = "INVALID_2FA_CODE"
	Code2FANotEnabled        = "2FA_NOT_ENABLED"
	Code2FAAlreadyEnabled    = "2FA_ALREADY_ENABLED"
	CodeInvalidAPIKey        = "INVALID_API_KEY"
	CodeAPIKeyExpired        = "API_KEY_EXPIRED"
	CodeAPIKeyRevoked        = "API_KEY_REVOKED"
	CodeRateLimitExceeded    = "RATE_LIMIT_EXCEEDED"
	CodeInvalidInput         = "INVALID_INPUT"
	CodeInternalError        = "INTERNAL_ERROR"
)

// NewErrorResponse creates a new error response
func NewErrorResponse(err error, code string) *ErrorResponse {
	return &ErrorResponse{
		Error:   err.Error(),
		Code:    code,
		Message: err.Error(),
	}
}

// WithDetails adds details to the error response
func (e *ErrorResponse) WithDetails(details map[string]string) *ErrorResponse {
	e.Details = details
	return e
}

