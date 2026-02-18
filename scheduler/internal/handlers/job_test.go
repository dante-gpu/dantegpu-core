package handlers

import (
	"bytes"
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

func TestJobHandler_SubmitJob(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		mockSetup      func(sqlmock.Sqlmock)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful job submission",
			requestBody: map[string]interface{}{
				"user_id":       "user-123",
				"gpu_id":        "gpu-456",
				"docker_image":  "tensorflow/tensorflow:latest-gpu",
				"command":       "python train.py",
				"gpu_count":     1,
				"estimated_duration_hours": 2,
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Check GPU availability
				mock.ExpectQuery("SELECT is_available, status FROM gpu_capabilities").
					WithArgs("gpu-456").
					WillReturnRows(sqlmock.NewRows([]string{"is_available", "status"}).
						AddRow(true, "online"))

				// Insert job
				mock.ExpectExec("INSERT INTO jobs").
					WithArgs(sqlmock.AnyArg(), "user-123", "gpu-456",
						"tensorflow/tensorflow:latest-gpu", "python train.py",
						1, 2, "pending", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Publish to NATS queue (mocked)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "GPU not available",
			requestBody: map[string]interface{}{
				"user_id":      "user-123",
				"gpu_id":       "gpu-789",
				"docker_image": "pytorch/pytorch:latest",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT is_available, status FROM gpu_capabilities").
					WithArgs("gpu-789").
					WillReturnRows(sqlmock.NewRows([]string{"is_available", "status"}).
						AddRow(false, "offline"))
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "GPU not available",
		},
		{
			name: "missing required fields",
			requestBody: map[string]interface{}{
				"user_id": "user-123",
			},
			mockSetup:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			handler := NewJobHandler(db, &mockNATSClient{})

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.SubmitJob(rr, req)

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

func TestJobHandler_GetJob(t *testing.T) {
	jobID := "job-123"

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM jobs").
		WithArgs(jobID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "gpu_id", "docker_image", "command",
			"gpu_count", "status", "created_at", "started_at", "completed_at",
		}).AddRow(
			jobID, "user-123", "gpu-456", "tensorflow/tensorflow:latest-gpu",
			"python train.py", 1, "running", time.Now(), time.Now(), nil,
		))

	handler := NewJobHandler(db, &mockNATSClient{})

	req := httptest.NewRequest(http.MethodGet, "/jobs/"+jobID, nil)
	rr := httptest.NewRecorder()

	handler.GetJob(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var job map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &job)

	assert.Equal(t, jobID, job["id"])
	assert.Equal(t, "running", job["status"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestJobHandler_ListJobs(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(sqlmock.Sqlmock)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:        "list all jobs for user",
			queryParams: "?user_id=user-123",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "gpu_id", "docker_image", "status", "created_at",
				}).
					AddRow("job-1", "user-123", "gpu-1", "tensorflow/tensorflow:latest-gpu", "completed", time.Now()).
					AddRow("job-2", "user-123", "gpu-2", "pytorch/pytorch:latest", "running", time.Now())

				mock.ExpectQuery("SELECT (.+) FROM jobs").
					WithArgs("user-123").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:        "filter by status",
			queryParams: "?user_id=user-123&status=running",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "gpu_id", "docker_image", "status", "created_at",
				}).
					AddRow("job-2", "user-123", "gpu-2", "pytorch/pytorch:latest", "running", time.Now())

				mock.ExpectQuery("SELECT (.+) FROM jobs").
					WithArgs("user-123", "running").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			handler := NewJobHandler(db, &mockNATSClient{})

			req := httptest.NewRequest(http.MethodGet, "/jobs"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.ListJobs(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &response)
				jobs := response["jobs"].([]interface{})
				assert.Equal(t, tt.expectedCount, len(jobs))
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestJobHandler_CancelJob(t *testing.T) {
	tests := []struct {
		name           string
		jobID          string
		mockSetup      func(sqlmock.Sqlmock)
		expectedStatus int
	}{
		{
			name:  "successful job cancellation",
			jobID: "job-123",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Get job status
				mock.ExpectQuery("SELECT status FROM jobs").
					WithArgs("job-123").
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("running"))

				// Update job status
				mock.ExpectExec("UPDATE jobs").
					WithArgs("cancelled", sqlmock.AnyArg(), "job-123").
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Publish cancellation event to NATS
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "job already completed",
			jobID: "job-456",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT status FROM jobs").
					WithArgs("job-456").
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("completed"))
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:  "job not found",
			jobID: "job-999",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT status FROM jobs").
					WithArgs("job-999").
					WillReturnError(sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			handler := NewJobHandler(db, &mockNATSClient{})

			req := httptest.NewRequest(http.MethodPost, "/jobs/"+tt.jobID+"/cancel", nil)
			rr := httptest.NewRecorder()

			handler.CancelJob(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestJobHandler_GetJobLogs(t *testing.T) {
	jobID := "job-123"

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM job_logs").
		WithArgs(jobID).
		WillReturnRows(sqlmock.NewRows([]string{"timestamp", "level", "message"}).
			AddRow(time.Now(), "INFO", "Starting job execution").
			AddRow(time.Now(), "INFO", "Loading model").
			AddRow(time.Now(), "INFO", "Training started"))

	handler := NewJobHandler(db, &mockNATSClient{})

	req := httptest.NewRequest(http.MethodGet, "/jobs/"+jobID+"/logs", nil)
	rr := httptest.NewRecorder()

	handler.GetJobLogs(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)

	logs := response["logs"].([]interface{})
	assert.Equal(t, 3, len(logs))

	assert.NoError(t, mock.ExpectationsWereMet())
}

// Mock NATS client for testing
type mockNATSClient struct{}

func (m *mockNATSClient) Publish(subject string, data []byte) error {
	return nil
}

func (m *mockNATSClient) Subscribe(subject string, handler func([]byte)) error {
	return nil
}

func (m *mockNATSClient) Close() error {
	return nil
}

