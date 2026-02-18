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

// JobExecutionTestSuite tests complete job execution flow
type JobExecutionTestSuite struct {
	suite.Suite
	schedulerDB *sql.DB
	registryDB  *sql.DB
	authDB      *sql.DB
	accessToken string
	userID      string
	gpuID       string
	jobID       string
}

func (suite *JobExecutionTestSuite) SetupSuite() {
	t := suite.T()

	// Connect to databases
	schedulerDB, err := sql.Open("postgres", "postgres://dante_user:dante_password@localhost:5432/dante_scheduler_test?sslmode=disable")
	require.NoError(t, err)
	suite.schedulerDB = schedulerDB

	registryDB, err := sql.Open("postgres", "postgres://dante_user:dante_password@localhost:5432/dante_registry_test?sslmode=disable")
	require.NoError(t, err)
	suite.registryDB = registryDB

	authDB, err := sql.Open("postgres", "postgres://dante_user:dante_password@localhost:5432/dante_auth_test?sslmode=disable")
	require.NoError(t, err)
	suite.authDB = authDB

	// Setup test user and GPU
	suite.setupTestUser()
	suite.setupTestGPU()
}

func (suite *JobExecutionTestSuite) TearDownSuite() {
	suite.cleanupTestData()
	suite.schedulerDB.Close()
	suite.registryDB.Close()
	suite.authDB.Close()
}

func (suite *JobExecutionTestSuite) setupTestUser() {
	t := suite.T()

	email := fmt.Sprintf("job_test_%d@example.com", time.Now().Unix())
	
	// Register and login user (simplified)
	reqBody := map[string]interface{}{
		"email":      email,
		"password":   "JobTest123!",
		"first_name": "Job",
		"last_name":  "Tester",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "http://localhost:8001/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	// Get user ID and verify
	err = suite.authDB.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&suite.userID)
	require.NoError(t, err)

	_, err = suite.authDB.Exec("UPDATE users SET email_verified = true WHERE id = $1", suite.userID)
	require.NoError(t, err)

	// Login
	loginBody := map[string]string{"email": email, "password": "JobTest123!"}
	body, _ = json.Marshal(loginBody)
	req, _ = http.NewRequest("POST", "http://localhost:8001/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var loginResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&loginResp)
	suite.accessToken = loginResp["access_token"].(string)

	t.Logf("✅ Test user created: %s", suite.userID)
}

func (suite *JobExecutionTestSuite) setupTestGPU() {
	t := suite.T()

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

func (suite *JobExecutionTestSuite) cleanupTestData() {
	ctx := context.Background()

	if suite.jobID != "" {
		suite.schedulerDB.ExecContext(ctx, "DELETE FROM jobs WHERE id = $1", suite.jobID)
	}
	if suite.gpuID != "" {
		suite.registryDB.ExecContext(ctx, "DELETE FROM gpu_capabilities WHERE id = $1", suite.gpuID)
	}
	if suite.userID != "" {
		suite.authDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", suite.userID)
	}
}

// Test 1: Submit Job
func (suite *JobExecutionTestSuite) TestStep1_SubmitJob() {
	t := suite.T()

	reqBody := map[string]interface{}{
		"gpu_id":                   suite.gpuID,
		"docker_image":             "tensorflow/tensorflow:latest-gpu",
		"command":                  "python -c 'print(\"Hello from GPU!\")'",
		"gpu_count":                1,
		"estimated_duration_hours": 1,
		"environment_variables": map[string]string{
			"CUDA_VISIBLE_DEVICES": "0",
		},
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "http://localhost:8004/api/v1/jobs", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	assert.Contains(t, response, "job_id")
	suite.jobID = response["job_id"].(string)
	assert.NotEmpty(t, suite.jobID)

	// Verify job in database
	var status string
	err = suite.schedulerDB.QueryRow("SELECT status FROM jobs WHERE id = $1", suite.jobID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "pending", status)

	t.Logf("✅ Job submitted: %s", suite.jobID)
}

// Test 2: Job Scheduling
func (suite *JobExecutionTestSuite) TestStep2_JobScheduling() {
	t := suite.T()

	// Wait for job to be scheduled (in real system, scheduler picks it up)
	time.Sleep(2 * time.Second)

	// Check job status
	req, _ := http.NewRequest("GET", "http://localhost:8004/api/v1/jobs/"+suite.jobID, nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var job map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&job)

	status := job["status"].(string)
	assert.Contains(t, []string{"pending", "scheduled", "running"}, status)

	t.Logf("✅ Job status: %s", status)
}

// Test 3: Job Execution (simulated)
func (suite *JobExecutionTestSuite) TestStep3_JobExecution() {
	t := suite.T()

	// Simulate job execution by updating status
	_, err := suite.schedulerDB.Exec(`
		UPDATE jobs 
		SET status = 'running', started_at = NOW() 
		WHERE id = $1
	`, suite.jobID)
	require.NoError(t, err)

	// Simulate log generation
	_, err = suite.schedulerDB.Exec(`
		INSERT INTO job_logs (id, job_id, timestamp, level, message)
		VALUES 
			(gen_random_uuid(), $1, NOW(), 'INFO', 'Container started'),
			(gen_random_uuid(), $1, NOW(), 'INFO', 'Loading TensorFlow'),
			(gen_random_uuid(), $1, NOW(), 'INFO', 'Hello from GPU!'),
			(gen_random_uuid(), $1, NOW(), 'INFO', 'Job completed successfully')
	`, suite.jobID)
	require.NoError(t, err)

	t.Logf("✅ Job execution simulated")
}

// Test 4: Get Job Logs
func (suite *JobExecutionTestSuite) TestStep4_GetJobLogs() {
	t := suite.T()

	req, _ := http.NewRequest("GET", "http://localhost:8004/api/v1/jobs/"+suite.jobID+"/logs", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	logs := response["logs"].([]interface{})
	assert.Greater(t, len(logs), 0, "Should have job logs")

	// Verify log content
	foundHelloMessage := false
	for _, log := range logs {
		logMap := log.(map[string]interface{})
		if logMap["message"] == "Hello from GPU!" {
			foundHelloMessage = true
			break
		}
	}
	assert.True(t, foundHelloMessage, "Should find 'Hello from GPU!' message")

	t.Logf("✅ Job logs retrieved: %d log entries", len(logs))
}

// Test 5: Job Completion
func (suite *JobExecutionTestSuite) TestStep5_JobCompletion() {
	t := suite.T()

	// Mark job as completed
	_, err := suite.schedulerDB.Exec(`
		UPDATE jobs 
		SET status = 'completed', completed_at = NOW() 
		WHERE id = $1
	`, suite.jobID)
	require.NoError(t, err)

	// Verify job status
	req, _ := http.NewRequest("GET", "http://localhost:8004/api/v1/jobs/"+suite.jobID, nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var job map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&job)

	assert.Equal(t, "completed", job["status"])
	assert.NotNil(t, job["completed_at"])

	// Verify GPU is available again
	var isAvailable bool
	err = suite.registryDB.QueryRow("SELECT is_available FROM gpu_capabilities WHERE id = $1", suite.gpuID).
		Scan(&isAvailable)
	require.NoError(t, err)
	assert.True(t, isAvailable, "GPU should be available after job completion")

	t.Logf("✅ Job completed successfully")
}

// Test 6: Job Cancellation
func (suite *JobExecutionTestSuite) TestStep6_JobCancellation() {
	t := suite.T()

	// Submit another job for cancellation test
	reqBody := map[string]interface{}{
		"gpu_id":       suite.gpuID,
		"docker_image": "pytorch/pytorch:latest",
		"command":      "python train.py",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "http://localhost:8004/api/v1/jobs", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var submitResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&submitResp)
	cancelJobID := submitResp["job_id"].(string)

	// Cancel the job
	req, _ = http.NewRequest("POST", "http://localhost:8004/api/v1/jobs/"+cancelJobID+"/cancel", nil)
	req.Header.Set("Authorization", "Bearer "+suite.accessToken)

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify job is cancelled
	var status string
	err = suite.schedulerDB.QueryRow("SELECT status FROM jobs WHERE id = $1", cancelJobID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", status)

	// Cleanup
	suite.schedulerDB.Exec("DELETE FROM jobs WHERE id = $1", cancelJobID)

	t.Logf("✅ Job cancellation successful")
}

// Run the test suite
func TestJobExecutionTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite.Run(t, new(JobExecutionTestSuite))
}

