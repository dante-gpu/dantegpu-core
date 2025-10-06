package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// RentalFlowTestSuite tests the complete rental flow with real Solana devnet
type RentalFlowTestSuite struct {
	suite.Suite
	db             *sql.DB
	authDB         *sql.DB
	billingDB      *sql.DB
	registryDB     *sql.DB
	accessToken    string
	userID         string
	walletAddress  string
	gpuID          string
	rentalID       string
	sessionID      string
}

func (suite *RentalFlowTestSuite) SetupSuite() {
	t := suite.T()

	// Connect to databases
	authDB, err := sql.Open("postgres", "postgres://dante_user:dante_password@localhost:5432/dante_auth_test?sslmode=disable")
	require.NoError(t, err)
	suite.authDB = authDB

	billingDB, err := sql.Open("postgres", "postgres://dante_user:dante_password@localhost:5432/dante_billing_test?sslmode=disable")
	require.NoError(t, err)
	suite.billingDB = billingDB

	registryDB, err := sql.Open("postgres", "postgres://dante_user:dante_password@localhost:5432/dante_registry_test?sslmode=disable")
	require.NoError(t, err)
	suite.registryDB = registryDB

	// Create and login test user
	suite.setupTestUser()

	// Create test GPU
	suite.setupTestGPU()
}

func (suite *RentalFlowTestSuite) TearDownSuite() {
	suite.cleanupTestData()
	suite.authDB.Close()
	suite.billingDB.Close()
	suite.registryDB.Close()
}

func (suite *RentalFlowTestSuite) setupTestUser() {
	t := suite.T()

	// Register user
	email := fmt.Sprintf("rental_test_%d@example.com", time.Now().Unix())
	reqBody := map[string]interface{}{
		"email":      email,
		"password":   "RentalTest123!",
		"first_name": "Rental",
		"last_name":  "Tester",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "http://localhost:8001/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	// Get user ID and verify email directly in database
	err = suite.authDB.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&suite.userID)
	require.NoError(t, err)

	_, err = suite.authDB.Exec("UPDATE users SET email_verified = true WHERE id = $1", suite.userID)
	require.NoError(t, err)

	// Login
	loginBody := map[string]string{
		"email":    email,
		"password": "RentalTest123!",
	}

	body, _ = json.Marshal(loginBody)
	req, _ = http.NewRequest("POST", "http://localhost:8001/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var loginResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&loginResp)
	suite.accessToken = loginResp["access_token"].(string)

	t.Logf("✅ Test user created and logged in: %s", suite.userID)
}

func (suite *RentalFlowTestSuite) setupTestGPU() {
	t := suite.T()

	// Insert test GPU directly into database
	err := suite.registryDB.QueryRow(`
		INSERT INTO gpu_capabilities (id, provider_id, gpu_model, gpu_memory_gb, 
			compute_capability, price_per_minute, is_available, status)
		VALUES (gen_random_uuid(), gen_random_uuid(), 'NVIDIA RTX 4090', 24, 
			'8.9', 0.05, true, 'online')
		RETURNING id
	`).Scan(&suite.gpuID)
	require.NoError(t, err)

	t.Logf("✅ Test GPU created: %s", suite.gpuID)
}

func (suite *RentalFlowTestSuite) cleanupTestData() {
	ctx := context.Background()

	// Cleanup in reverse order of dependencies
	if suite.sessionID != "" {
		suite.billingDB.ExecContext(ctx, "DELETE FROM rental_sessions WHERE id = $1", suite.sessionID)
	}
	if suite.walletAddress != "" {
		suite.billingDB.ExecContext(ctx, "DELETE FROM wallets WHERE user_id = $1", suite.userID)
	}
	if suite.gpuID != "" {
		suite.registryDB.ExecContext(ctx, "DELETE FROM gpu_capabilities WHERE id = $1", suite.gpuID)
	}
	if suite.userID != "" {
		suite.authDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", suite.userID)
	}
}

// Test 1: Create Wallet
func (suite *RentalFlowTestSuite) TestStep1_CreateWallet() {
	t := suite.T()

	req, _ := http.NewRequest("POST", "http://localhost:8002/api/v1/wallet/create", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	assert.Contains(t, response, "wallet_address")
	suite.walletAddress = response["wallet_address"].(string)
	assert.NotEmpty(t, suite.walletAddress)

	// Verify wallet in database
	var walletID string
	err = suite.billingDB.QueryRow("SELECT id FROM wallets WHERE user_id = $1", suite.userID).
		Scan(&walletID)
	require.NoError(t, err)

	t.Logf("✅ Wallet created: %s", suite.walletAddress)
}

// Test 2: Get Wallet Balance
func (suite *RentalFlowTestSuite) TestStep2_GetWalletBalance() {
	t := suite.T()

	req, _ := http.NewRequest("GET", "http://localhost:8002/api/v1/wallet", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var wallet map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&wallet)

	assert.Equal(t, suite.walletAddress, wallet["wallet_address"])
	assert.Contains(t, wallet, "balance")
	assert.Contains(t, wallet, "available_balance")

	// Balance should be 0 for new wallet
	balance := wallet["balance"].(float64)
	assert.GreaterOrEqual(t, balance, float64(0))

	t.Logf("✅ Wallet balance retrieved: %.6f dGPU", balance)
}

// Test 3: Browse GPUs
func (suite *RentalFlowTestSuite) TestStep3_BrowseGPUs() {
	t := suite.T()

	req, _ := http.NewRequest("GET", "http://localhost:8003/api/v1/gpus?available=true", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	gpus := response["gpus"].([]interface{})
	assert.Greater(t, len(gpus), 0, "Should have at least one GPU")

	// Find our test GPU
	var foundTestGPU bool
	for _, gpu := range gpus {
		gpuMap := gpu.(map[string]interface{})
		if gpuMap["id"].(string) == suite.gpuID {
			foundTestGPU = true
			assert.Equal(t, "NVIDIA RTX 4090", gpuMap["gpu_model"])
			assert.Equal(t, float64(24), gpuMap["gpu_memory_gb"])
			assert.True(t, gpuMap["is_available"].(bool))
			break
		}
	}

	assert.True(t, foundTestGPU, "Test GPU should be in the list")

	t.Logf("✅ GPUs browsed successfully, found %d available GPUs", len(gpus))
}

// Test 4: Start Rental (with simulated deposit)
func (suite *RentalFlowTestSuite) TestStep4_StartRental() {
	t := suite.T()

	// Simulate deposit by updating wallet balance directly
	// In production, this would be done via real Solana transaction
	_, err := suite.billingDB.Exec(`
		UPDATE wallets 
		SET balance = 100.0, available_balance = 100.0 
		WHERE user_id = $1
	`, suite.userID)
	require.NoError(t, err)

	// Start rental
	reqBody := map[string]interface{}{
		"gpu_id":          suite.gpuID,
		"estimated_hours": 2,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "http://localhost:8002/api/v1/billing/start-rental", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	assert.Contains(t, response, "session_id")
	suite.sessionID = response["session_id"].(string)
	assert.NotEmpty(t, suite.sessionID)

	// Verify rental session in database
	var status string
	var escrowAmount float64
	err = suite.billingDB.QueryRow(`
		SELECT status, escrow_amount 
		FROM rental_sessions 
		WHERE id = $1
	`, suite.sessionID).Scan(&status, &escrowAmount)
	require.NoError(t, err)

	assert.Equal(t, "active", status)
	assert.Greater(t, escrowAmount, float64(0))

	// Verify GPU is no longer available
	var isAvailable bool
	err = suite.registryDB.QueryRow("SELECT is_available FROM gpu_capabilities WHERE id = $1", suite.gpuID).
		Scan(&isAvailable)
	require.NoError(t, err)
	assert.False(t, isAvailable, "GPU should not be available during rental")

	t.Logf("✅ Rental started: session %s, escrow %.6f dGPU", suite.sessionID, escrowAmount)
}

// Test 5: Billing Processing (simulate minute-based billing)
func (suite *RentalFlowTestSuite) TestStep5_BillingProcessing() {
	t := suite.T()

	// Wait a bit to simulate usage
	time.Sleep(2 * time.Second)

	// Trigger billing processing (normally done by cron job)
	req, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:8002/api/v1/billing/process/%s", suite.sessionID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify usage records were created
	var usageCount int
	err = suite.billingDB.QueryRow("SELECT COUNT(*) FROM usage_records WHERE session_id = $1", suite.sessionID).
		Scan(&usageCount)
	require.NoError(t, err)
	assert.Greater(t, usageCount, 0, "Usage records should be created")

	t.Logf("✅ Billing processed, %d usage records created", usageCount)
}

// Test 6: End Rental and Payout
func (suite *RentalFlowTestSuite) TestStep6_EndRental() {
	t := suite.T()

	// End rental
	reqBody := map[string]string{
		"reason": "completed",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:8002/api/v1/billing/end-rental/%s", suite.sessionID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify rental session is completed
	var status string
	var endedAt *time.Time
	err = suite.billingDB.QueryRow("SELECT status, ended_at FROM rental_sessions WHERE id = $1", suite.sessionID).
		Scan(&status, &endedAt)
	require.NoError(t, err)

	assert.Equal(t, "completed", status)
	assert.NotNil(t, endedAt)

	// Verify provider payout was created
	var payoutCount int
	err = suite.billingDB.QueryRow("SELECT COUNT(*) FROM provider_payouts WHERE session_id = $1", suite.sessionID).
		Scan(&payoutCount)
	require.NoError(t, err)
	assert.Greater(t, payoutCount, 0, "Provider payout should be created")

	// Verify platform fee was collected
	var feeCount int
	err = suite.billingDB.QueryRow("SELECT COUNT(*) FROM platform_fees WHERE session_id = $1", suite.sessionID).
		Scan(&feeCount)
	require.NoError(t, err)
	assert.Greater(t, feeCount, 0, "Platform fee should be collected")

	// Verify GPU is available again
	var isAvailable bool
	err = suite.registryDB.QueryRow("SELECT is_available FROM gpu_capabilities WHERE id = $1", suite.gpuID).
		Scan(&isAvailable)
	require.NoError(t, err)
	assert.True(t, isAvailable, "GPU should be available after rental ends")

	t.Logf("✅ Rental ended successfully, provider payout and platform fee processed")
}

// Run the test suite
func TestRentalFlowTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite.Run(t, new(RentalFlowTestSuite))
}

