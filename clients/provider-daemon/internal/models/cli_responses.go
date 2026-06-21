package models

import "time"

// CliGpuInfo mirrors the GpuInfo struct in provider-gui/src-tauri/src/main.rs
type CliGpuInfo struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Model                 string   `json:"model"`
	VRAMTotalMB           uint32   `json:"vram_total_mb"`
	VRAMFreeMB            uint32   `json:"vram_free_mb"`
	UtilizationGPUPercent *uint32  `json:"utilization_gpu_percent,omitempty"`
	TemperatureC          *uint32  `json:"temperature_c,omitempty"`
	PowerDrawW            *uint32  `json:"power_draw_w,omitempty"`
	IsAvailableForRent    bool     `json:"is_available_for_rent"`
	CurrentHourlyRateDGPU *float32 `json:"current_hourly_rate_dgpu,omitempty"`
}

// CliProviderSettings mirrors the ProviderSettings struct in provider-gui
type CliProviderSettings struct {
	DefaultHourlyRateDGPU float32 `json:"default_hourly_rate_dgpu"`
	PreferredCurrency     string  `json:"preferred_currency"`
	MinJobDurationMinutes uint32  `json:"min_job_duration_minutes"`
	MaxConcurrentJobs     uint32  `json:"max_concurrent_jobs"`
}

// CliLocalJob mirrors the LocalJob struct in provider-gui
type CliLocalJob struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Status            string   `json:"status"`
	ProgressPercent   float32  `json:"progress_percent"`
	SubmittedAt       string   `json:"submitted_at"`
	StartedAt         *string  `json:"started_at,omitempty"`
	CompletedAt       *string  `json:"completed_at,omitempty"`
	EstimatedCostDGPU *float32 `json:"estimated_cost_dgpu,omitempty"`
}

// CliSystemOverview provides a snapshot of system-level metrics.
// TODO: Populate with actual system metrics (CPU, RAM, Disk, Network I/O).
type CliSystemOverview struct {
	TotalDiskSpaceGB uint64  `json:"total_disk_space_gb"`
	FreeDiskSpaceGB  uint64  `json:"free_disk_space_gb"`
	CpuUsagePercent  float32 `json:"cpu_usage_percent"`
	RamUsagePercent  float32 `json:"ram_usage_percent"`
	UptimeSeconds    uint64  `json:"uptime_seconds"`
}

// CliNetworkStatus provides information about the daemon's network connectivity.
type CliNetworkStatus struct {
	NatsConnected       bool   `json:"nats_connected"`
	NatsServerURL       string `json:"nats_server_url"`
	LastNatsError       string `json:"last_nats_error,omitempty"`
	ActiveSubscriptions int    `json:"active_subscriptions,omitempty"` // Example: Number of active NATS subscriptions
}

// CliFinancialSummary mirrors the FinancialSummary struct in provider-gui
type CliFinancialSummary struct {
	CurrentBalanceDGPU float32 `json:"current_balance_dgpu"`
	TotalEarnedDGPU    float32 `json:"total_earned_dgpu"`
	PendingPayoutDGPU  float32 `json:"pending_payout_dgpu"`
	LastPayoutAt       *string `json:"last_payout_at,omitempty"`
}

// CliFinancialOverview provides a summary of financial data.
// TODO: Populate with actual financial data (e.g., earnings, balance).

// ptrFloat32 returns a pointer to a float32 value.
func ptrFloat32(f float32) *float32 {
	return &f
}

// ptrUint32 returns a pointer to a uint32 value.
func ptrUint32(u uint32) *uint32 {
	return &u
}

// ptrString returns a pointer to a string value, or nil if the string is empty.
func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ptrTime returns a pointer to a Time value, or nil if the time is zero.
func ptrTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// PtrStringToTime converts a string pointer in RFC3339 format to a time.Time pointer.
func PtrStringToTime(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		// Consider logging the error or handling it more gracefully
		return nil
	}
	return &t
}
