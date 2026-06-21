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
	"golang.org/x/crypto/bcrypt"
)

func TestLoginHandler_Login(t *testing.T) {
	// Generate password hash for testing
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("SecurePass123!"), bcrypt.DefaultCost)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		mockSetup      func(sqlmock.Sqlmock)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful login",
			requestBody: map[string]interface{}{
				"email":    "test@example.com",
				"password": "SecurePass123!",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Get user
				mock.ExpectQuery("SELECT id, email, password_hash, email_verified, is_active").
					WithArgs("test@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "email_verified", "is_active"}).
						AddRow("user-123", "test@example.com", string(passwordHash), true, true))

				// Check account lockout
				mock.ExpectQuery("SELECT COUNT").
					WithArgs("user-123", sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				// Check 2FA
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("user-123").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

				// Get user roles
				mock.ExpectQuery("SELECT r.name").
					WithArgs("user-123").
					WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("user"))

				// Begin transaction
				mock.ExpectBegin()

				// Create session
				mock.ExpectExec("INSERT INTO active_sessions").
					WithArgs(sqlmock.AnyArg(), "user-123", sqlmock.AnyArg(), sqlmock.AnyArg(),
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Record login attempt
				mock.ExpectExec("INSERT INTO login_attempts").
					WithArgs("user-123", sqlmock.AnyArg(), sqlmock.AnyArg(), true, sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Commit transaction
				mock.ExpectCommit()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid credentials",
			requestBody: map[string]interface{}{
				"email":    "test@example.com",
				"password": "WrongPassword123!",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, email, password_hash, email_verified, is_active").
					WithArgs("test@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "email_verified", "is_active"}).
						AddRow("user-123", "test@example.com", string(passwordHash), true, true))

				// Check account lockout
				mock.ExpectQuery("SELECT COUNT").
					WithArgs("user-123", sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				// Record failed login
				mock.ExpectExec("INSERT INTO login_attempts").
					WithArgs("user-123", sqlmock.AnyArg(), sqlmock.AnyArg(), false, sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid credentials",
		},
		{
			name: "email not verified",
			requestBody: map[string]interface{}{
				"email":    "unverified@example.com",
				"password": "SecurePass123!",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, email, password_hash, email_verified, is_active").
					WithArgs("unverified@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "email_verified", "is_active"}).
						AddRow("user-456", "unverified@example.com", string(passwordHash), false, true))
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Email not verified",
		},
		{
			name: "account locked",
			requestBody: map[string]interface{}{
				"email":    "locked@example.com",
				"password": "SecurePass123!",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, email, password_hash, email_verified, is_active").
					WithArgs("locked@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "email_verified", "is_active"}).
						AddRow("user-789", "locked@example.com", string(passwordHash), true, true))

				// Account is locked (5+ failed attempts)
				mock.ExpectQuery("SELECT COUNT").
					WithArgs("user-789", sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
			},
			expectedStatus: http.StatusLocked,
			expectedError:  "Account locked",
		},
		{
			name: "user not found",
			requestBody: map[string]interface{}{
				"email":    "nonexistent@example.com",
				"password": "SecurePass123!",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, email, password_hash, email_verified, is_active").
					WithArgs("nonexistent@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			// Create mock JWT manager
			jwtManager := &mockJWTManager{}

			// Create mock session store
			sessionStore := &mockSessionStore{}

			handler := NewLoginHandler(db, jwtManager, sessionStore, &mockEmailService{})

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.Login(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedError != "" {
				var response map[string]string
				json.Unmarshal(rr.Body.Bytes(), &response)
				assert.Contains(t, response["error"], tt.expectedError)
			}

			// For successful login, verify tokens are returned
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NotEmpty(t, response["access_token"])
				assert.NotEmpty(t, response["refresh_token"])
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLoginHandler_RefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		refreshToken   string
		mockSetup      func(*mockJWTManager, sqlmock.Sqlmock)
		expectedStatus int
		expectedError  string
	}{
		{
			name:         "successful token refresh",
			refreshToken: "valid-refresh-token",
			mockSetup: func(jwtMgr *mockJWTManager, mock sqlmock.Sqlmock) {
				jwtMgr.validateRefreshFunc = func(token string) (string, []string, error) {
					return "user-123", []string{"user"}, nil
				}
				jwtMgr.generateTokenPairFunc = func(userID, email string, roles []string) (string, string, error) {
					return "new-access-token", "new-refresh-token", nil
				}

				// Get user email
				mock.ExpectQuery("SELECT email").
					WithArgs("user-123").
					WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("test@example.com"))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "invalid refresh token",
			refreshToken: "invalid-token",
			mockSetup: func(jwtMgr *mockJWTManager, mock sqlmock.Sqlmock) {
				jwtMgr.validateRefreshFunc = func(token string) (string, []string, error) {
					return "", nil, ErrInvalidToken
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			jwtManager := &mockJWTManager{}
			sessionStore := &mockSessionStore{}

			tt.mockSetup(jwtManager, mock)

			handler := NewLoginHandler(db, jwtManager, sessionStore, &mockEmailService{})

			body, _ := json.Marshal(map[string]string{"refresh_token": tt.refreshToken})
			req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.RefreshToken(rr, req)

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

// Mock JWT Manager
type mockJWTManager struct {
	generateTokenPairFunc func(userID, email string, roles []string) (string, string, error)
	validateRefreshFunc   func(token string) (string, []string, error)
}

func (m *mockJWTManager) GenerateTokenPair(userID, email string, roles []string) (string, string, error) {
	if m.generateTokenPairFunc != nil {
		return m.generateTokenPairFunc(userID, email, roles)
	}
	return "access-token", "refresh-token", nil
}

func (m *mockJWTManager) ValidateAccessToken(token string) (string, []string, error) {
	return "user-123", []string{"user"}, nil
}

func (m *mockJWTManager) ValidateRefreshToken(token string) (string, []string, error) {
	if m.validateRefreshFunc != nil {
		return m.validateRefreshFunc(token)
	}
	return "user-123", []string{"user"}, nil
}

// Mock Session Store
type mockSessionStore struct{}

func (m *mockSessionStore) Create(ctx context.Context, userID, sessionID string, expiresAt time.Time) error {
	return nil
}

func (m *mockSessionStore) Get(ctx context.Context, sessionID string) (string, error) {
	return "user-123", nil
}

func (m *mockSessionStore) Delete(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockSessionStore) DeleteAllUserSessions(ctx context.Context, userID string) error {
	return nil
}

