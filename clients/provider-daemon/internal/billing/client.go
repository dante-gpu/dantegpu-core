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
	// For CliFinancialSummary type if needed, or define local types
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

// UsageUpdateRequest represents a usage update request
type UsageUpdateRequest struct {
	SessionID       uuid.UUID `json:"session_id"`
	GPUUtilization  uint8     `json:"gpu_utilization_percent"`
	VRAMUtilization uint8     `json:"vram_utilization_percent"`
	PowerDraw       uint32    `json:"power_draw_w"`
	Temperature     uint8     `json:"temperature_c"`
	Timestamp       time.Time `json:"timestamp"`
}

// SessionResponse represents a session response from billing service
type SessionResponse struct {
	Session struct {
		ID               uuid.UUID       `json:"id"`
		UserID           string          `json:"user_id"`
		ProviderID       uuid.UUID       `json:"provider_id"`
		JobID            *string         `json:"job_id,omitempty"`
		Status           string          `json:"status"`
		GPUModel         string          `json:"gpu_model"`
		AllocatedVRAM    uint64          `json:"allocated_vram_mb"`
		TotalVRAM        uint64          `json:"total_vram_mb"`
		VRAMPercentage   decimal.Decimal `json:"vram_percentage"`
		HourlyRate       decimal.Decimal `json:"hourly_rate"`
		VRAMRate         decimal.Decimal `json:"vram_rate"`
		PowerRate        decimal.Decimal `json:"power_rate"`
		PlatformFeeRate  decimal.Decimal `json:"platform_fee_rate"`
		EstimatedPowerW  uint32          `json:"estimated_power_w"`
		ActualPowerW     *uint32         `json:"actual_power_w,omitempty"`
		StartedAt        time.Time       `json:"started_at"`
		EndedAt          *time.Time      `json:"ended_at,omitempty"`
		LastBilledAt     time.Time       `json:"last_billed_at"`
		TotalCost        decimal.Decimal `json:"total_cost"`
		PlatformFee      decimal.Decimal `json:"platform_fee"`
		ProviderEarnings decimal.Decimal `json:"provider_earnings"`
		CreatedAt        time.Time       `json:"created_at"`
		UpdatedAt        time.Time       `json:"updated_at"`
	} `json:"session"`
	CurrentCost         decimal.Decimal `json:"current_cost"`
	EstimatedHourlyCost decimal.Decimal `json:"estimated_hourly_cost"`
	RemainingBalance    decimal.Decimal `json:"remaining_balance"`
	EstimatedRuntime    decimal.Decimal `json:"estimated_runtime_hours"`
}

// SendUsageUpdate sends real-time usage data to the billing service
func (c *Client) SendUsageUpdate(ctx context.Context, req *UsageUpdateRequest) error {
	c.logger.Debug("Sending usage update",
		zap.String("session_id", req.SessionID.String()),
		zap.Uint8("gpu_utilization", req.GPUUtilization),
		zap.Uint32("power_draw", req.PowerDraw),
	)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal usage update: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/billing/usage-update", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send usage update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("billing service returned status %d", resp.StatusCode)
	}

	c.logger.Debug("Usage update sent successfully")
	return nil
}

// GetCurrentUsage gets current usage information for a session
func (c *Client) GetCurrentUsage(ctx context.Context, sessionID uuid.UUID) (*SessionResponse, error) {
	url := fmt.Sprintf("%s/api/v1/billing/current-usage/%s", c.baseURL, sessionID.String())

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage: %w", err)
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

// Monitor starts monitoring a session and sends periodic usage updates
func (c *Client) Monitor(ctx context.Context, sessionID uuid.UUID, gpuID string, interval time.Duration) error {
	c.logger.Info("Starting billing monitor",
		zap.String("session_id", sessionID.String()),
		zap.String("gpu_id", gpuID),
		zap.Duration("interval", interval),
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Billing monitor stopped", zap.String("session_id", sessionID.String()))
			return ctx.Err()
		case <-ticker.C:
			// Get current GPU metrics
			metrics, err := c.getGPUMetrics(gpuID)
			if err != nil {
				c.logger.Error("Failed to get GPU metrics", zap.Error(err))
				continue
			}

			// Send usage update
			req := &UsageUpdateRequest{
				SessionID:       sessionID,
				GPUUtilization:  metrics.Utilization,
				VRAMUtilization: metrics.VRAMUtilization,
				PowerDraw:       metrics.PowerDraw,
				Temperature:     metrics.Temperature,
				Timestamp:       time.Now().UTC(),
			}

			if err := c.SendUsageUpdate(ctx, req); err != nil {
				c.logger.Error("Failed to send usage update", zap.Error(err))
				// Continue monitoring even if one update fails
			}
		}
	}
}

// GPUMetrics represents GPU metrics for billing
type GPUMetrics struct {
	Utilization     uint8  `json:"utilization_percent"`
	VRAMUtilization uint8  `json:"vram_utilization_percent"`
	PowerDraw       uint32 `json:"power_draw_w"`
	Temperature     uint8  `json:"temperature_c"`
}

// getGPUMetrics gets current GPU metrics
func (c *Client) getGPUMetrics(gpuID string) (*GPUMetrics, error) {
	// This would integrate with the GPU detector
	// For now, return mock data
	return &GPUMetrics{
		Utilization:     75,  // mock data
		VRAMUtilization: 50,  // mock data
		PowerDraw:       150, // mock data
		Temperature:     65,  // mock data
	}, nil
}

// StartBilling informs the billing service that a task's billable period has begun.
func (c *Client) StartBilling(ctx context.Context, jobID string, userID string, gpuInstanceID string, pricePerHour float64) error {
	c.logger.Info("StartBilling called",
		zap.String("jobID", jobID),
		zap.String("userID", userID),
		zap.String("gpuInstanceID", gpuInstanceID),
		zap.Float64("pricePerHour", pricePerHour),
	)
	// TODO: This method needs to be implemented to correctly interact with the billing service API.
	// It might involve creating or activating a session, or directly logging the start of a billable event.
	// The current parameters (jobID, userID, gpuInstanceID, pricePerHour) should be used.
	// For now, this is a stub.
	c.logger.Warn("StartBilling is currently a stub and does not actually initiate a billing event with the service.")
	return nil // Placeholder
}

// StopBilling informs the billing service that a task's billable period has ended.
func (c *Client) StopBilling(ctx context.Context, jobID string, userID string, durationHours float64) error {
	c.logger.Info("StopBilling called",
		zap.String("jobID", jobID),
		zap.String("userID", userID),
		zap.Float64("durationHours", durationHours),
	)
	// TODO: This method needs to be implemented to correctly interact with the billing service API.
	// It might involve ending a session, or logging the end of a billable event with its duration.
	// The current parameters (jobID, userID, durationHours) should be used.
	// For now, this is a stub.
	c.logger.Warn("StopBilling is currently a stub and does not actually terminate a billing event with the service correctly.")
	return nil // Placeholder
}

// CheckSessionStatus checks if a session is still considered active by the billing service.
// This might be useful for the daemon to periodically verify if it should continue processing a task.
// Returns true if active, false if not active or error.
func (c *Client) CheckSessionStatus(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	usage, err := c.GetCurrentUsage(ctx, sessionID)
	if err != nil {
		return false, err
	}

	// Check if session is still active and has remaining balance
	isActive := usage.Session.Status == "active" && usage.RemainingBalance.GreaterThan(decimal.Zero)

	if !isActive {
		c.logger.Warn("Session is no longer active or funded",
			zap.String("session_id", sessionID.String()),
			zap.String("status", usage.Session.Status),
			zap.String("remaining_balance", usage.RemainingBalance.String()),
		)
	}

	return isActive, nil
}

// FinancialSummaryDetails represents the detailed financial data expected from the billing service.
// This is a local representation, mapping to what CliFinancialSummary might need.
type FinancialSummaryDetails struct {
	TotalEarnedDGPU    float32
	PendingPayoutDGPU  float32
	CurrentBalanceDGPU float32 // This might be the primary balance query
	LastPayoutDGPU     float32
	LastPayoutAt       *time.Time
	// Potentially other fields like NextEstimatedPayout etc.
}

// GetBalance (hypothetical, from previous linter error context)
// This is a placeholder, assuming the billing service would have an endpoint for this.
func (c *Client) GetBalance(ctx context.Context, providerID string) (float64, error) {
	c.logger.Warn("billing.Client.GetBalance is a STUB and not implemented. Returning mock data.", zap.String("providerID", providerID))
	// Mock implementation: call an endpoint like /api/v1/wallet/{providerID}/balance
	// url := fmt.Sprintf("%s/api/v1/wallet/%s/balance", c.baseURL, providerID)
	// ... http GET request ...
	// ... unmarshal response ...
	return 123.45, nil // Mock balance
}

// GetFinancialSummary retrieves a summary of financial data for the provider.
// This is a STUB method and needs a corresponding endpoint in the billing service.
func (c *Client) GetFinancialSummary(ctx context.Context, providerID string) (*FinancialSummaryDetails, error) {
	c.logger.Warn("billing.Client.GetFinancialSummary is a STUB and not implemented. Returning mock data.", zap.String("providerID", providerID))

	// In a real implementation, this would call an endpoint on the billing service:
	// e.g., GET /api/v1/providers/{providerID}/financial-summary
	// For now, return mock data based on CliFinancialSummary structure.

	// Attempt to get at least the current balance using the stubbed GetBalance
	balance, _ := c.GetBalance(ctx, providerID) // Ignore error for stub

	mockTime := time.Now().Add(-24 * time.Hour)
	return &FinancialSummaryDetails{
		TotalEarnedDGPU:    float32(balance + 1000.0), // Mock
		PendingPayoutDGPU:  float32(balance),          // Mock based on GetBalance stub
		CurrentBalanceDGPU: float32(balance),          // Mock
		LastPayoutDGPU:     500.0,                     // Mock
		LastPayoutAt:       &mockTime,                 // Mock
	}, nil
}
