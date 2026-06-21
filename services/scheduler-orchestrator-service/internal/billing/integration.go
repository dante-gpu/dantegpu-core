package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Client represents a client for the billing service
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// Config represents billing client configuration
type Config struct {
	BaseURL string        `yaml:"base_url"`
	Timeout time.Duration `yaml:"timeout"`
}

// NewClient creates a new billing service client
func NewClient(config *Config, logger *zap.Logger) *Client {
	return &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
	}
}

// SessionStartRequest represents a request to start a rental session
type SessionStartRequest struct {
	UserID           string          `json:"user_id"`
	ProviderID       uuid.UUID       `json:"provider_id"`
	JobID            *string         `json:"job_id,omitempty"`
	GPUModel         string          `json:"gpu_model"`
	RequestedVRAM    uint64          `json:"requested_vram_mb"`
	EstimatedPowerW  uint32          `json:"estimated_power_w"`
	MaxHourlyRate    *decimal.Decimal `json:"max_hourly_rate,omitempty"`
	MaxDurationHours *int            `json:"max_duration_hours,omitempty"`
}

// SessionEndRequest represents a request to end a rental session
type SessionEndRequest struct {
	SessionID uuid.UUID `json:"session_id"`
	Reason    string    `json:"reason,omitempty"`
}

// SessionResponse represents a session response from billing service
type SessionResponse struct {
	Session struct {
		ID                uuid.UUID       `json:"id"`
		UserID            string          `json:"user_id"`
		ProviderID        uuid.UUID       `json:"provider_id"`
		JobID             *string         `json:"job_id,omitempty"`
		Status            string          `json:"status"`
		GPUModel          string          `json:"gpu_model"`
		AllocatedVRAM     uint64          `json:"allocated_vram_mb"`
		TotalVRAM         uint64          `json:"total_vram_mb"`
		VRAMPercentage    decimal.Decimal `json:"vram_percentage"`
		HourlyRate        decimal.Decimal `json:"hourly_rate"`
		VRAMRate          decimal.Decimal `json:"vram_rate"`
		PowerRate         decimal.Decimal `json:"power_rate"`
		PlatformFeeRate   decimal.Decimal `json:"platform_fee_rate"`
		EstimatedPowerW   uint32          `json:"estimated_power_w"`
		ActualPowerW      *uint32         `json:"actual_power_w,omitempty"`
		StartedAt         time.Time       `json:"started_at"`
		EndedAt           *time.Time      `json:"ended_at,omitempty"`
		LastBilledAt      time.Time       `json:"last_billed_at"`
		TotalCost         decimal.Decimal `json:"total_cost"`
		PlatformFee       decimal.Decimal `json:"platform_fee"`
		ProviderEarnings  decimal.Decimal `json:"provider_earnings"`
		CreatedAt         time.Time       `json:"created_at"`
		UpdatedAt         time.Time       `json:"updated_at"`
	} `json:"session"`
	CurrentCost         decimal.Decimal `json:"current_cost"`
	EstimatedHourlyCost decimal.Decimal `json:"estimated_hourly_cost"`
	RemainingBalance    decimal.Decimal `json:"remaining_balance"`
	EstimatedRuntime    decimal.Decimal `json:"estimated_runtime_hours"`
}

// StartSession starts a new rental session
func (c *Client) StartSession(ctx context.Context, req *SessionStartRequest) (*SessionResponse, error) {
	c.logger.Info("Starting rental session",
		zap.String("user_id", req.UserID),
		zap.String("provider_id", req.ProviderID.String()),
		zap.String("gpu_model", req.GPUModel),
		zap.Uint64("requested_vram_mb", req.RequestedVRAM),
	)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session start request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/billing/start-session", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("billing service returned status %d", resp.StatusCode)
	}

	var sessionResp SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("Rental session started successfully",
		zap.String("session_id", sessionResp.Session.ID.String()),
		zap.String("estimated_hourly_cost", sessionResp.EstimatedHourlyCost.String()),
	)

	return &sessionResp, nil
}

// EndSession ends a rental session
func (c *Client) EndSession(ctx context.Context, req *SessionEndRequest) (*SessionResponse, error) {
	c.logger.Info("Ending rental session",
		zap.String("session_id", req.SessionID.String()),
		zap.String("reason", req.Reason),
	)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session end request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/billing/end-session", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to end session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("billing service returned status %d", resp.StatusCode)
	}

	var sessionResp SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("Rental session ended successfully",
		zap.String("session_id", sessionResp.Session.ID.String()),
		zap.String("total_cost", sessionResp.CurrentCost.String()),
	)

	return &sessionResp, nil
}

// CheckUserBalance checks if a user has sufficient balance for a session
func (c *Client) CheckUserBalance(ctx context.Context, userID string, estimatedCost decimal.Decimal) (bool, decimal.Decimal, error) {
	// This would need to be implemented to check user wallet balance
	// For now, return a placeholder
	return true, decimal.NewFromInt(100), nil
}

// GetSessionStatus gets the current status of a session
func (c *Client) GetSessionStatus(ctx context.Context, sessionID uuid.UUID) (*SessionResponse, error) {
	url := fmt.Sprintf("%s/api/v1/billing/current-usage/%s", c.baseURL, sessionID.String())
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get session status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("billing service returned status %d", resp.StatusCode)
	}

	var sessionResp SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &sessionResp, nil
}

// ValidateJobRequirements validates that a job can be billed properly
func (c *Client) ValidateJobRequirements(ctx context.Context, userID string, gpuModel string, vramMB uint64, estimatedPowerW uint32) error {
	c.logger.Debug("Validating job requirements for billing",
		zap.String("user_id", userID),
		zap.String("gpu_model", gpuModel),
		zap.Uint64("vram_mb", vramMB),
		zap.Uint32("estimated_power_w", estimatedPowerW),
	)

	// Check if user has sufficient balance
	// This is a simplified check - in production, you'd calculate actual estimated cost
	estimatedHourlyCost := decimal.NewFromFloat(1.0) // Placeholder
	hasBalance, balance, err := c.CheckUserBalance(ctx, userID, estimatedHourlyCost)
	if err != nil {
		return fmt.Errorf("failed to check user balance: %w", err)
	}

	if !hasBalance {
		return fmt.Errorf("insufficient balance: required %s, available %s", estimatedHourlyCost.String(), balance.String())
	}

	c.logger.Debug("Job requirements validated successfully")
	return nil
}

// EstimateJobCost estimates the cost of a job based on requirements
func (c *Client) EstimateJobCost(ctx context.Context, gpuModel string, vramMB uint64, estimatedPowerW uint32, durationHours decimal.Decimal) (decimal.Decimal, error) {
	// This would call the pricing calculation endpoint
	// For now, return a simple calculation
	baseRate := decimal.NewFromFloat(0.5) // Base rate per hour
	vramRate := decimal.NewFromFloat(0.02) // Per GB per hour
	powerRate := decimal.NewFromFloat(0.001) // Per watt per hour

	vramGB := decimal.NewFromInt(int64(vramMB)).Div(decimal.NewFromInt(1024))
	powerKW := decimal.NewFromInt(int64(estimatedPowerW)).Div(decimal.NewFromInt(1000))

	hourlyCost := baseRate.Add(vramRate.Mul(vramGB)).Add(powerRate.Mul(powerKW))
	totalCost := hourlyCost.Mul(durationHours)

	c.logger.Debug("Estimated job cost",
		zap.String("hourly_cost", hourlyCost.String()),
		zap.String("total_cost", totalCost.String()),
		zap.String("duration_hours", durationHours.String()),
	)

	return totalCost, nil
}
