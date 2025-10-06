package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistrationHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		mockSetup      func(sqlmock.Sqlmock)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful registration",
			requestBody: map[string]interface{}{
				"email":      "test@example.com",
				"password":   "SecurePass123!",
				"first_name": "John",
				"last_name":  "Doe",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Check if user exists
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("test@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

				// Begin transaction
				mock.ExpectBegin()

				// Insert user
				mock.ExpectExec("INSERT INTO users").
					WithArgs(sqlmock.AnyArg(), "test@example.com", sqlmock.AnyArg(),
						"John", "Doe", false, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Insert verification token
				mock.ExpectExec("INSERT INTO email_verification_tokens").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Commit transaction
				mock.ExpectCommit()
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "duplicate email",
			requestBody: map[string]interface{}{
				"email":      "existing@example.com",
				"password":   "SecurePass123!",
				"first_name": "Jane",
				"last_name":  "Doe",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("existing@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "Email already registered",
		},
		{
			name: "weak password",
			requestBody: map[string]interface{}{
				"email":      "test@example.com",
				"password":   "weak",
				"first_name": "John",
				"last_name":  "Doe",
			},
			mockSetup:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Password must be at least 8 characters",
		},
		{
			name: "invalid email format",
			requestBody: map[string]interface{}{
				"email":      "invalid-email",
				"password":   "SecurePass123!",
				"first_name": "John",
				"last_name":  "Doe",
			},
			mockSetup:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid email format",
		},
		{
			name: "missing required fields",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			mockSetup:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			// Setup mock expectations
			tt.mockSetup(mock)

			// Create handler
			handler := NewRegistrationHandler(db, &mockEmailService{})

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handler.Register(rr, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Assert error message if expected
			if tt.expectedError != "" {
				var response map[string]string
				json.Unmarshal(rr.Body.Bytes(), &response)
				assert.Contains(t, response["error"], tt.expectedError)
			}

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRegistrationHandler_VerifyEmail(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		mockSetup      func(sqlmock.Sqlmock)
		expectedStatus int
		expectedError  string
	}{
		{
			name:  "successful verification",
			token: "valid-token-123",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Get token
				mock.ExpectQuery("SELECT user_id, expires_at, used_at").
					WithArgs("valid-token-123").
					WillReturnRows(sqlmock.NewRows([]string{"user_id", "expires_at", "used_at"}).
						AddRow("user-123", time.Now().Add(24*time.Hour), nil))

				// Begin transaction
				mock.ExpectBegin()

				// Mark token as used
				mock.ExpectExec("UPDATE email_verification_tokens").
					WithArgs(sqlmock.AnyArg(), "valid-token-123").
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Update user
				mock.ExpectExec("UPDATE users").
					WithArgs(sqlmock.AnyArg(), "user-123").
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Commit transaction
				mock.ExpectCommit()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "expired token",
			token: "expired-token",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT user_id, expires_at, used_at").
					WithArgs("expired-token").
					WillReturnRows(sqlmock.NewRows([]string{"user_id", "expires_at", "used_at"}).
						AddRow("user-123", time.Now().Add(-24*time.Hour), nil))
			},
			expectedStatus: http.StatusGone,
			expectedError:  "Verification token has expired",
		},
		{
			name:  "already used token",
			token: "used-token",
			mockSetup: func(mock sqlmock.Sqlmock) {
				usedAt := time.Now().Add(-1 * time.Hour)
				mock.ExpectQuery("SELECT user_id, expires_at, used_at").
					WithArgs("used-token").
					WillReturnRows(sqlmock.NewRows([]string{"user_id", "expires_at", "used_at"}).
						AddRow("user-123", time.Now().Add(24*time.Hour), &usedAt))
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "Verification token already used",
		},
		{
			name:  "invalid token",
			token: "invalid-token",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT user_id, expires_at, used_at").
					WithArgs("invalid-token").
					WillReturnError(sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Invalid verification token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			handler := NewRegistrationHandler(db, &mockEmailService{})

			body, _ := json.Marshal(map[string]string{"token": tt.token})
			req := httptest.NewRequest(http.MethodPost, "/verify-email", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.VerifyEmail(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedError != "" {
				var response map[string]string
				json.Unmarshal(rr.Body.Bytes(), &response)
				assert.Contains(t, response["error"], tt.expectedError)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPasswordStrength(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
	}{
		{"SecurePass123!", true},
		{"Weak1!", false},                    // Too short
		{"nouppercase123!", false},           // No uppercase
		{"NOLOWERCASE123!", false},           // No lowercase
		{"NoNumbers!", false},                // No numbers
		{"NoSpecialChar123", false},          // No special char
		{"ValidPassword1@", true},
		{"AnotherGood1#", true},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			err := validatePasswordStrength(tt.password)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// Mock email service for testing
type mockEmailService struct{}

func (m *mockEmailService) SendVerificationEmail(ctx context.Context, email, token, firstName string) error {
	return nil
}

func (m *mockEmailService) SendWelcomeEmail(ctx context.Context, email, firstName string) error {
	return nil
}

func (m *mockEmailService) SendPasswordResetEmail(ctx context.Context, email, token, firstName string) error {
	return nil
}

func (m *mockEmailService) SendPasswordChangedEmail(ctx context.Context, email, firstName string) error {
	return nil
}

func (m *mockEmailService) Send2FACodeEmail(ctx context.Context, email, code, firstName string) error {
	return nil
}

func (m *mockEmailService) SendLoginAlertEmail(ctx context.Context, email, firstName, ipAddress, location string) error {
	return nil
}

// Helper function to validate password strength
func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		case char == '!' || char == '@' || char == '#' || char == '$' || char == '%' || char == '^' || char == '&' || char == '*':
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return ErrPasswordTooWeak
	}

	return nil
}

