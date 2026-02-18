package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderHandler_RegisterProvider(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		mockSetup      func(sqlmock.Sqlmock)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful provider registration",
			requestBody: map[string]interface{}{
				"user_id":      "user-123",
				"company_name": "GPU Cloud Inc",
				"contact_email": "contact@gpucloud.com",
				"description":  "Professional GPU provider",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Check if provider already exists
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("user-123").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

				// Insert provider
				mock.ExpectExec("INSERT INTO providers").
					WithArgs(sqlmock.AnyArg(), "user-123", "GPU Cloud Inc", 
						"contact@gpucloud.com", "Professional GPU provider", 
						"pending", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "duplicate provider registration",
			requestBody: map[string]interface{}{
				"user_id":      "user-456",
				"company_name": "Existing Provider",
				"contact_email": "existing@provider.com",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("user-456").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "Provider already registered",
		},
		{
			name: "missing required fields",
			requestBody: map[string]interface{}{
				"user_id": "user-789",
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

			handler := NewProviderHandler(db)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/providers/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.RegisterProvider(rr, req)

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

func TestProviderHandler_AddGPUCapability(t *testing.T) {
	tests := []struct {
		name           string
		providerID     string
		requestBody    map[string]interface{}
		mockSetup      func(sqlmock.Sqlmock)
		expectedStatus int
	}{
		{
			name:       "successful GPU addition",
			providerID: "provider-123",
			requestBody: map[string]interface{}{
				"gpu_model":          "NVIDIA RTX 4090",
				"gpu_memory_gb":      24,
				"compute_capability": "8.9",
				"price_per_minute":   0.05,
				"cuda_cores":         16384,
				"tensor_cores":       512,
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Verify provider exists and is approved
				mock.ExpectQuery("SELECT status FROM providers").
					WithArgs("provider-123").
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("approved"))

				// Insert GPU capability
				mock.ExpectExec("INSERT INTO gpu_capabilities").
					WithArgs(sqlmock.AnyArg(), "provider-123", "NVIDIA RTX 4090",
						24, "8.9", 0.05, 16384, 512, true, "online",
						sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:       "provider not approved",
			providerID: "provider-456",
			requestBody: map[string]interface{}{
				"gpu_model":      "NVIDIA RTX 3090",
				"gpu_memory_gb":  24,
				"price_per_minute": 0.04,
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT status FROM providers").
					WithArgs("provider-456").
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("pending"))
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			handler := NewProviderHandler(db)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/providers/"+tt.providerID+"/gpus", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.AddGPUCapability(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProviderHandler_ListGPUs(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(sqlmock.Sqlmock)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:        "list all available GPUs",
			queryParams: "?available=true",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "provider_id", "gpu_model", "gpu_memory_gb",
					"compute_capability", "price_per_minute", "is_available", "status",
				}).
					AddRow("gpu-1", "provider-1", "NVIDIA RTX 4090", 24, "8.9", 0.05, true, "online").
					AddRow("gpu-2", "provider-2", "NVIDIA RTX 3090", 24, "8.6", 0.04, true, "online")

				mock.ExpectQuery("SELECT (.+) FROM gpu_capabilities").
					WithArgs(true).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:        "filter by GPU model",
			queryParams: "?gpu_model=RTX 4090",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "provider_id", "gpu_model", "gpu_memory_gb",
					"compute_capability", "price_per_minute", "is_available", "status",
				}).
					AddRow("gpu-1", "provider-1", "NVIDIA RTX 4090", 24, "8.9", 0.05, true, "online")

				mock.ExpectQuery("SELECT (.+) FROM gpu_capabilities").
					WithArgs("RTX 4090").
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

			handler := NewProviderHandler(db)

			req := httptest.NewRequest(http.MethodGet, "/gpus"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			handler.ListGPUs(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &response)
				gpus := response["gpus"].([]interface{})
				assert.Equal(t, tt.expectedCount, len(gpus))
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProviderHandler_UpdateGPUAvailability(t *testing.T) {
	tests := []struct {
		name           string
		gpuID          string
		available      bool
		mockSetup      func(sqlmock.Sqlmock)
		expectedStatus int
	}{
		{
			name:      "mark GPU as unavailable",
			gpuID:     "gpu-123",
			available: false,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE gpu_capabilities").
					WithArgs(false, sqlmock.AnyArg(), "gpu-123").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "mark GPU as available",
			gpuID:     "gpu-456",
			available: true,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE gpu_capabilities").
					WithArgs(true, sqlmock.AnyArg(), "gpu-456").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "GPU not found",
			gpuID:     "gpu-999",
			available: true,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE gpu_capabilities").
					WithArgs(true, sqlmock.AnyArg(), "gpu-999").
					WillReturnResult(sqlmock.NewResult(0, 0))
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

			handler := NewProviderHandler(db)

			body, _ := json.Marshal(map[string]bool{"available": tt.available})
			req := httptest.NewRequest(http.MethodPatch, "/gpus/"+tt.gpuID+"/availability", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.UpdateGPUAvailability(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProviderHandler_GetProviderStatistics(t *testing.T) {
	providerID := "provider-123"

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock statistics query
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(providerID).
		WillReturnRows(sqlmock.NewRows([]string{"total_gpus"}).AddRow(5))

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(providerID, true).
		WillReturnRows(sqlmock.NewRows([]string{"available_gpus"}).AddRow(3))

	mock.ExpectQuery("SELECT COALESCE").
		WithArgs(providerID).
		WillReturnRows(sqlmock.NewRows([]string{"total_earnings"}).AddRow(1250.50))

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(providerID).
		WillReturnRows(sqlmock.NewRows([]string{"total_rentals"}).AddRow(42))

	handler := NewProviderHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/providers/"+providerID+"/statistics", nil)
	rr := httptest.NewRecorder()

	handler.GetProviderStatistics(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var stats map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &stats)

	assert.Equal(t, float64(5), stats["total_gpus"])
	assert.Equal(t, float64(3), stats["available_gpus"])
	assert.Equal(t, 1250.50, stats["total_earnings"])
	assert.Equal(t, float64(42), stats["total_rentals"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

