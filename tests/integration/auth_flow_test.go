package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AuthFlowTestSuite tests the complete authentication flow
type AuthFlowTestSuite struct {
	suite.Suite
	db          *sql.DB
	authServer  *httptest.Server
	testEmail   string
	testUser    TestUser
}

type TestUser struct {
	ID           string
	Email        string
	Password     string
	FirstName    string
	LastName     string
	AccessToken  string
	RefreshToken string
	VerifyToken  string
}

func (suite *AuthFlowTestSuite) SetupSuite() {
	// Connect to test database
	connStr := "postgres://dante_user:dante_password@localhost:5432/dante_auth_test?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	require.NoError(suite.T(), err)

	suite.db = db

	// Clean up test data
	suite.cleanupTestData()

	// Setup test user
	suite.testEmail = fmt.Sprintf("test_%d@example.com", time.Now().Unix())
	suite.testUser = TestUser{
		Email:     suite.testEmail,
		Password:  "SecureTestPass123!",
		FirstName: "Integration",
		LastName:  "Test",
	}
}

func (suite *AuthFlowTestSuite) TearDownSuite() {
	suite.cleanupTestData()
	suite.db.Close()
}

func (suite *AuthFlowTestSuite) cleanupTestData() {
	ctx := context.Background()

	// Delete test users
	_, err := suite.db.ExecContext(ctx, "DELETE FROM users WHERE email LIKE 'test_%@example.com'")
	if err != nil {
		suite.T().Logf("Cleanup warning: %v", err)
	}
}

// Test 1: User Registration
func (suite *AuthFlowTestSuite) TestStep1_UserRegistration() {
	t := suite.T()

	// Prepare registration request
	reqBody := map[string]interface{}{
		"email":      suite.testUser.Email,
		"password":   suite.testUser.Password,
		"first_name": suite.testUser.FirstName,
		"last_name":  suite.testUser.LastName,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "http://localhost:8001/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert successful registration
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	assert.Contains(t, response, "message")
	assert.Equal(t, "Registration successful. Please check your email to verify your account.", response["message"])

	// Verify user was created in database
	var userID string
	var emailVerified bool
	err = suite.db.QueryRow("SELECT id, email_verified FROM users WHERE email = $1", suite.testUser.Email).
		Scan(&userID, &emailVerified)
	require.NoError(t, err)

	suite.testUser.ID = userID
	assert.False(t, emailVerified, "Email should not be verified yet")

	// Get verification token from database
	var verifyToken string
	err = suite.db.QueryRow("SELECT token FROM email_verification_tokens WHERE user_id = $1 AND used_at IS NULL",
		userID).Scan(&verifyToken)
	require.NoError(t, err)

	suite.testUser.VerifyToken = verifyToken
	t.Logf("✅ User registered successfully with ID: %s", userID)
}

// Test 2: Email Verification
func (suite *AuthFlowTestSuite) TestStep2_EmailVerification() {
	t := suite.T()

	// Verify email with token
	reqBody := map[string]string{
		"token": suite.testUser.VerifyToken,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "http://localhost:8001/api/v1/auth/verify-email", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert successful verification
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify email_verified flag in database
	var emailVerified bool
	err = suite.db.QueryRow("SELECT email_verified FROM users WHERE id = $1", suite.testUser.ID).
		Scan(&emailVerified)
	require.NoError(t, err)

	assert.True(t, emailVerified, "Email should be verified")

	// Verify token was marked as used
	var usedAt *time.Time
	err = suite.db.QueryRow("SELECT used_at FROM email_verification_tokens WHERE token = $1",
		suite.testUser.VerifyToken).Scan(&usedAt)
	require.NoError(t, err)
	assert.NotNil(t, usedAt, "Token should be marked as used")

	t.Logf("✅ Email verified successfully")
}

// Test 3: Login
func (suite *AuthFlowTestSuite) TestStep3_Login() {
	t := suite.T()

	// Login with credentials
	reqBody := map[string]string{
		"email":    suite.testUser.Email,
		"password": suite.testUser.Password,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "http://localhost:8001/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert successful login
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	// Extract tokens
	assert.Contains(t, response, "access_token")
	assert.Contains(t, response, "refresh_token")

	suite.testUser.AccessToken = response["access_token"].(string)
	suite.testUser.RefreshToken = response["refresh_token"].(string)

	assert.NotEmpty(t, suite.testUser.AccessToken)
	assert.NotEmpty(t, suite.testUser.RefreshToken)

	// Verify session was created in database
	var sessionCount int
	err = suite.db.QueryRow("SELECT COUNT(*) FROM active_sessions WHERE user_id = $1", suite.testUser.ID).
		Scan(&sessionCount)
	require.NoError(t, err)
	assert.Greater(t, sessionCount, 0, "Session should be created")

	// Verify login attempt was recorded
	var attemptCount int
	err = suite.db.QueryRow("SELECT COUNT(*) FROM login_attempts WHERE user_id = $1 AND success = true",
		suite.testUser.ID).Scan(&attemptCount)
	require.NoError(t, err)
	assert.Greater(t, attemptCount, 0, "Login attempt should be recorded")

	t.Logf("✅ Login successful, tokens received")
}

// Test 4: Access Protected Endpoint
func (suite *AuthFlowTestSuite) TestStep4_AccessProtectedEndpoint() {
	t := suite.T()

	// Access protected endpoint with access token
	req, _ := http.NewRequest("GET", "http://localhost:8001/api/v1/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+suite.testUser.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert successful access
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var profile map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&profile)

	assert.Equal(t, suite.testUser.Email, profile["email"])
	assert.Equal(t, suite.testUser.FirstName, profile["first_name"])
	assert.Equal(t, suite.testUser.LastName, profile["last_name"])

	t.Logf("✅ Protected endpoint accessed successfully")
}

// Test 5: Token Refresh
func (suite *AuthFlowTestSuite) TestStep5_TokenRefresh() {
	t := suite.T()

	// Refresh tokens
	reqBody := map[string]string{
		"refresh_token": suite.testUser.RefreshToken,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "http://localhost:8001/api/v1/auth/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert successful refresh
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	// Verify new tokens
	assert.Contains(t, response, "access_token")
	assert.Contains(t, response, "refresh_token")

	newAccessToken := response["access_token"].(string)
	newRefreshToken := response["refresh_token"].(string)

	assert.NotEmpty(t, newAccessToken)
	assert.NotEmpty(t, newRefreshToken)
	assert.NotEqual(t, suite.testUser.AccessToken, newAccessToken, "New access token should be different")

	// Update tokens
	suite.testUser.AccessToken = newAccessToken
	suite.testUser.RefreshToken = newRefreshToken

	t.Logf("✅ Tokens refreshed successfully")
}

// Test 6: Logout
func (suite *AuthFlowTestSuite) TestStep6_Logout() {
	t := suite.T()

	// Logout
	req, _ := http.NewRequest("POST", "http://localhost:8001/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+suite.testUser.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert successful logout
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify session was deleted
	var sessionCount int
	err = suite.db.QueryRow("SELECT COUNT(*) FROM active_sessions WHERE user_id = $1", suite.testUser.ID).
		Scan(&sessionCount)
	require.NoError(t, err)
	assert.Equal(t, 0, sessionCount, "All sessions should be deleted")

	// Verify token was blacklisted
	var blacklistedCount int
	err = suite.db.QueryRow("SELECT COUNT(*) FROM revoked_tokens WHERE user_id = $1", suite.testUser.ID).
		Scan(&blacklistedCount)
	require.NoError(t, err)
	assert.Greater(t, blacklistedCount, 0, "Token should be blacklisted")

	t.Logf("✅ Logout successful, session terminated")
}

// Run the test suite
func TestAuthFlowTestSuite(t *testing.T) {
	suite.Run(t, new(AuthFlowTestSuite))
}

