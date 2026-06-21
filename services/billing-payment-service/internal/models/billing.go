package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SessionStatus represents the status of a rental session
type SessionStatus string

const (
	SessionStatusActive     SessionStatus = "active"
	SessionStatusCompleted  SessionStatus = "completed"
	SessionStatusCancelled  SessionStatus = "cancelled"
	SessionStatusSuspended  SessionStatus = "suspended"
	SessionStatusTerminated SessionStatus = "terminated"
)

// RentalSession represents a GPU rental session
type RentalSession struct {
	ID                uuid.UUID       `json:"id" db:"id"`
	UserID            string          `json:"user_id" db:"user_id"`
	ProviderID        uuid.UUID       `json:"provider_id" db:"provider_id"`
	JobID             *string         `json:"job_id,omitempty" db:"job_id"`
	Status            SessionStatus   `json:"status" db:"status"`
	
	// GPU allocation details
	GPUModel          string          `json:"gpu_model" db:"gpu_model"`
	AllocatedVRAM     uint64          `json:"allocated_vram_mb" db:"allocated_vram_mb"` // VRAM in MB
	TotalVRAM         uint64          `json:"total_vram_mb" db:"total_vram_mb"`         // Total GPU VRAM
	VRAMPercentage    decimal.Decimal `json:"vram_percentage" db:"vram_percentage"`     // Percentage of VRAM allocated
	
	// Pricing information
	HourlyRate        decimal.Decimal `json:"hourly_rate" db:"hourly_rate"`             // dGPU tokens per hour
	VRAMRate          decimal.Decimal `json:"vram_rate" db:"vram_rate"`                 // dGPU tokens per GB per hour
	PowerRate         decimal.Decimal `json:"power_rate" db:"power_rate"`               // dGPU tokens per watt per hour
	PlatformFeeRate   decimal.Decimal `json:"platform_fee_rate" db:"platform_fee_rate"` // Platform fee percentage
	
	// Power consumption
	EstimatedPowerW   uint32          `json:"estimated_power_w" db:"estimated_power_w"`
	ActualPowerW      *uint32         `json:"actual_power_w,omitempty" db:"actual_power_w"`
	
	// Session timing
	StartedAt         time.Time       `json:"started_at" db:"started_at"`
	EndedAt           *time.Time      `json:"ended_at,omitempty" db:"ended_at"`
	LastBilledAt      time.Time       `json:"last_billed_at" db:"last_billed_at"`
	
	// Financial tracking
	TotalCost         decimal.Decimal `json:"total_cost" db:"total_cost"`               // Total cost in dGPU tokens
	PlatformFee       decimal.Decimal `json:"platform_fee" db:"platform_fee"`          // Platform fee amount
	ProviderEarnings  decimal.Decimal `json:"provider_earnings" db:"provider_earnings"` // Provider earnings
	
	// Metadata
	Metadata          map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
}

// Duration returns the duration of the session
func (rs *RentalSession) Duration() time.Duration {
	if rs.EndedAt != nil {
		return rs.EndedAt.Sub(rs.StartedAt)
	}
	return time.Since(rs.StartedAt)
}

// DurationHours returns the duration in hours
func (rs *RentalSession) DurationHours() decimal.Decimal {
	duration := rs.Duration()
	hours := decimal.NewFromFloat(duration.Hours())
	return hours
}

// CalculateCurrentCost calculates the current cost of the session
func (rs *RentalSession) CalculateCurrentCost() decimal.Decimal {
	hours := rs.DurationHours()
	
	// Base cost from hourly rate
	baseCost := rs.HourlyRate.Mul(hours)
	
	// VRAM cost
	vramGB := decimal.NewFromInt(int64(rs.AllocatedVRAM)).Div(decimal.NewFromInt(1024))
	vramCost := rs.VRAMRate.Mul(vramGB).Mul(hours)
	
	// Power cost
	powerCost := decimal.Zero
	if rs.ActualPowerW != nil {
		powerKW := decimal.NewFromInt(int64(*rs.ActualPowerW)).Div(decimal.NewFromInt(1000))
		powerCost = rs.PowerRate.Mul(powerKW).Mul(hours)
	} else {
		powerKW := decimal.NewFromInt(int64(rs.EstimatedPowerW)).Div(decimal.NewFromInt(1000))
		powerCost = rs.PowerRate.Mul(powerKW).Mul(hours)
	}
	
	totalCost := baseCost.Add(vramCost).Add(powerCost)
	return totalCost
}

// UsageRecord represents detailed usage tracking for billing
type UsageRecord struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	SessionID        uuid.UUID       `json:"session_id" db:"session_id"`
	RecordedAt       time.Time       `json:"recorded_at" db:"recorded_at"`
	
	// GPU utilization metrics
	GPUUtilization   uint8           `json:"gpu_utilization_percent" db:"gpu_utilization_percent"`
	VRAMUtilization  uint8           `json:"vram_utilization_percent" db:"vram_utilization_percent"`
	PowerDraw        uint32          `json:"power_draw_w" db:"power_draw_w"`
	Temperature      uint8           `json:"temperature_c" db:"temperature_c"`
	
	// Billing calculations for this period
	PeriodMinutes    int             `json:"period_minutes" db:"period_minutes"`
	PeriodCost       decimal.Decimal `json:"period_cost" db:"period_cost"`
	
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

// BillingRecord represents aggregated billing information
type BillingRecord struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	UserID           string          `json:"user_id" db:"user_id"`
	ProviderID       uuid.UUID       `json:"provider_id" db:"provider_id"`
	SessionID        uuid.UUID       `json:"session_id" db:"session_id"`
	
	// Billing period
	BillingPeriodStart time.Time     `json:"billing_period_start" db:"billing_period_start"`
	BillingPeriodEnd   time.Time     `json:"billing_period_end" db:"billing_period_end"`
	
	// Usage summary
	TotalMinutes     int             `json:"total_minutes" db:"total_minutes"`
	AvgGPUUtil       decimal.Decimal `json:"avg_gpu_utilization" db:"avg_gpu_utilization"`
	AvgVRAMUtil      decimal.Decimal `json:"avg_vram_utilization" db:"avg_vram_utilization"`
	AvgPowerDraw     decimal.Decimal `json:"avg_power_draw" db:"avg_power_draw"`
	
	// Cost breakdown
	BaseCost         decimal.Decimal `json:"base_cost" db:"base_cost"`
	VRAMCost         decimal.Decimal `json:"vram_cost" db:"vram_cost"`
	PowerCost        decimal.Decimal `json:"power_cost" db:"power_cost"`
	TotalCost        decimal.Decimal `json:"total_cost" db:"total_cost"`
	PlatformFee      decimal.Decimal `json:"platform_fee" db:"platform_fee"`
	ProviderEarnings decimal.Decimal `json:"provider_earnings" db:"provider_earnings"`
	
	// Transaction reference
	TransactionID    uuid.UUID       `json:"transaction_id" db:"transaction_id"`
	
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

// SessionStartRequest represents a request to start a rental session
type SessionStartRequest struct {
	UserID           string          `json:"user_id" validate:"required"`
	ProviderID       uuid.UUID       `json:"provider_id" validate:"required"`
	JobID            *string         `json:"job_id,omitempty"`
	GPUModel         string          `json:"gpu_model" validate:"required"`
	RequestedVRAM    uint64          `json:"requested_vram_mb" validate:"required,gt=0"`
	EstimatedPowerW  uint32          `json:"estimated_power_w" validate:"required,gt=0"`
	MaxHourlyRate    *decimal.Decimal `json:"max_hourly_rate,omitempty"`
	MaxDurationHours *int            `json:"max_duration_hours,omitempty"`
}

// SessionEndRequest represents a request to end a rental session
type SessionEndRequest struct {
	SessionID uuid.UUID `json:"session_id" validate:"required"`
	Reason    string    `json:"reason,omitempty"`
}

// UsageUpdateRequest represents real-time usage data from provider daemon
type UsageUpdateRequest struct {
	SessionID        uuid.UUID `json:"session_id" validate:"required"`
	GPUUtilization   uint8     `json:"gpu_utilization_percent" validate:"max=100"`
	VRAMUtilization  uint8     `json:"vram_utilization_percent" validate:"max=100"`
	PowerDraw        uint32    `json:"power_draw_w" validate:"required"`
	Temperature      uint8     `json:"temperature_c" validate:"required"`
	Timestamp        time.Time `json:"timestamp" validate:"required"`
}

// SessionResponse represents a session response
type SessionResponse struct {
	Session          RentalSession   `json:"session"`
	CurrentCost      decimal.Decimal `json:"current_cost"`
	EstimatedHourlyCost decimal.Decimal `json:"estimated_hourly_cost"`
	RemainingBalance decimal.Decimal `json:"remaining_balance"`
	EstimatedRuntime decimal.Decimal `json:"estimated_runtime_hours"`
}

// BillingHistoryRequest represents a request for billing history
type BillingHistoryRequest struct {
	UserID     *string    `json:"user_id,omitempty"`
	ProviderID *uuid.UUID `json:"provider_id,omitempty"`
	StartDate  *time.Time `json:"start_date,omitempty"`
	EndDate    *time.Time `json:"end_date,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// BillingHistoryResponse represents a billing history response
type BillingHistoryResponse struct {
	Records []BillingRecord `json:"records"`
	Total   int             `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
}

// ProviderEarningsRequest represents a request for provider earnings
type ProviderEarningsRequest struct {
	ProviderID uuid.UUID  `json:"provider_id" validate:"required"`
	StartDate  *time.Time `json:"start_date,omitempty"`
	EndDate    *time.Time `json:"end_date,omitempty"`
}

// ProviderEarningsResponse represents provider earnings response
type ProviderEarningsResponse struct {
	ProviderID       uuid.UUID       `json:"provider_id"`
	TotalEarnings    decimal.Decimal `json:"total_earnings"`
	PendingEarnings  decimal.Decimal `json:"pending_earnings"`
	PaidEarnings     decimal.Decimal `json:"paid_earnings"`
	TotalSessions    int             `json:"total_sessions"`
	TotalHours       decimal.Decimal `json:"total_hours"`
	AvgHourlyRate    decimal.Decimal `json:"avg_hourly_rate"`
	Period           string          `json:"period"`
}
