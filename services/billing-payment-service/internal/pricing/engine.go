package pricing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Engine handles dynamic pricing calculations for GPU rentals
type Engine struct {
	logger    *zap.Logger
	config    *Config
	baseRates map[string]decimal.Decimal
}

// Config represents pricing engine configuration
type Config struct {
	// Base rates by GPU model (dGPU tokens per hour)
	BaseRates map[string]float64 `yaml:"base_rates"`

	// VRAM pricing (dGPU tokens per GB per hour)
	VRAMRatePerGB decimal.Decimal `yaml:"vram_rate_per_gb"`

	// Power consumption multiplier (additional cost per watt per hour)
	PowerMultiplier decimal.Decimal `yaml:"power_multiplier"`

	// Platform fee percentage
	PlatformFeePercent decimal.Decimal `yaml:"platform_fee_percent"`

	// Session constraints
	MinimumSessionMinutes int `yaml:"minimum_session_minutes"`
	MaximumSessionHours   int `yaml:"maximum_session_hours"`

	// Dynamic pricing factors
	DemandMultiplierMax decimal.Decimal `yaml:"demand_multiplier_max"`
	SupplyBonusMax      decimal.Decimal `yaml:"supply_bonus_max"`
}

// NewEngine creates a new pricing engine
func NewEngine(config *Config, logger *zap.Logger) *Engine {
	baseRates := make(map[string]decimal.Decimal)
	for model, rate := range config.BaseRates {
		baseRates[strings.ToLower(model)] = decimal.NewFromFloat(rate)
	}

	return &Engine{
		logger:    logger,
		config:    config,
		baseRates: baseRates,
	}
}

// PricingRequest represents a request for pricing calculation
type PricingRequest struct {
	GPUModel        string          `json:"gpu_model"`
	RequestedVRAM   uint64          `json:"requested_vram_mb"`
	TotalVRAM       uint64          `json:"total_vram_mb"`
	EstimatedPowerW uint32          `json:"estimated_power_w"`
	DurationHours   decimal.Decimal `json:"duration_hours"`
	ProviderID      *uuid.UUID      `json:"provider_id,omitempty"`
	UserID          *string         `json:"user_id,omitempty"`
}

// PricingResponse represents the calculated pricing
type PricingResponse struct {
	// Base pricing components
	BaseHourlyRate  decimal.Decimal `json:"base_hourly_rate"`
	VRAMHourlyRate  decimal.Decimal `json:"vram_hourly_rate"`
	PowerHourlyRate decimal.Decimal `json:"power_hourly_rate"`
	TotalHourlyRate decimal.Decimal `json:"total_hourly_rate"`

	// Session pricing
	BaseCost         decimal.Decimal `json:"base_cost"`
	VRAMCost         decimal.Decimal `json:"vram_cost"`
	PowerCost        decimal.Decimal `json:"power_cost"`
	SubtotalCost     decimal.Decimal `json:"subtotal_cost"`
	PlatformFee      decimal.Decimal `json:"platform_fee"`
	TotalCost        decimal.Decimal `json:"total_cost"`
	ProviderEarnings decimal.Decimal `json:"provider_earnings"`

	// Dynamic pricing factors
	DemandMultiplier decimal.Decimal `json:"demand_multiplier"`
	SupplyBonus      decimal.Decimal `json:"supply_bonus"`

	// VRAM allocation details
	VRAMPercentage  decimal.Decimal `json:"vram_percentage"`
	AllocatedVRAMGB decimal.Decimal `json:"allocated_vram_gb"`

	// Metadata
	CalculatedAt time.Time `json:"calculated_at"`
	ValidUntil   time.Time `json:"valid_until"`
}

// CalculatePricing calculates the pricing for a GPU rental request
func (e *Engine) CalculatePricing(ctx context.Context, req *PricingRequest) (*PricingResponse, error) {
	e.logger.Debug("Calculating pricing",
		zap.String("gpu_model", req.GPUModel),
		zap.Uint64("requested_vram_mb", req.RequestedVRAM),
		zap.Uint32("estimated_power_w", req.EstimatedPowerW),
		zap.String("duration_hours", req.DurationHours.String()),
	)

	// Get base rate for GPU model
	baseRate, err := e.getBaseRate(req.GPUModel)
	if err != nil {
		return nil, fmt.Errorf("failed to get base rate: %w", err)
	}

	// Calculate VRAM allocation
	vramPercentage := decimal.NewFromInt(int64(req.RequestedVRAM)).Div(decimal.NewFromInt(int64(req.TotalVRAM)))
	allocatedVRAMGB := decimal.NewFromInt(int64(req.RequestedVRAM)).Div(decimal.NewFromInt(1024))

	// Calculate VRAM hourly rate
	vramHourlyRate := e.config.VRAMRatePerGB.Mul(allocatedVRAMGB)

	// Calculate power hourly rate
	powerKW := decimal.NewFromInt(int64(req.EstimatedPowerW)).Div(decimal.NewFromInt(1000))
	powerHourlyRate := e.config.PowerMultiplier.Mul(powerKW)

	// Get dynamic pricing factors
	demandMultiplier, supplyBonus, err := e.getDynamicPricingFactors(ctx, req)
	if err != nil {
		e.logger.Warn("Failed to get dynamic pricing factors, using defaults", zap.Error(err))
		demandMultiplier = decimal.NewFromInt(1)
		supplyBonus = decimal.Zero
	}

	// Apply dynamic pricing to base rate
	adjustedBaseRate := baseRate.Mul(demandMultiplier).Sub(baseRate.Mul(supplyBonus))
	if adjustedBaseRate.LessThan(baseRate.Mul(decimal.NewFromFloat(0.5))) {
		// Don't let price go below 50% of base rate
		adjustedBaseRate = baseRate.Mul(decimal.NewFromFloat(0.5))
	}

	// Calculate total hourly rate
	totalHourlyRate := adjustedBaseRate.Add(vramHourlyRate).Add(powerHourlyRate)

	// Calculate session costs
	baseCost := adjustedBaseRate.Mul(req.DurationHours)
	vramCost := vramHourlyRate.Mul(req.DurationHours)
	powerCost := powerHourlyRate.Mul(req.DurationHours)
	subtotalCost := baseCost.Add(vramCost).Add(powerCost)

	// Calculate platform fee
	platformFee := subtotalCost.Mul(e.config.PlatformFeePercent).Div(decimal.NewFromInt(100))
	totalCost := subtotalCost.Add(platformFee)
	providerEarnings := subtotalCost.Sub(platformFee)

	now := time.Now().UTC()
	response := &PricingResponse{
		BaseHourlyRate:   adjustedBaseRate,
		VRAMHourlyRate:   vramHourlyRate,
		PowerHourlyRate:  powerHourlyRate,
		TotalHourlyRate:  totalHourlyRate,
		BaseCost:         baseCost,
		VRAMCost:         vramCost,
		PowerCost:        powerCost,
		SubtotalCost:     subtotalCost,
		PlatformFee:      platformFee,
		TotalCost:        totalCost,
		ProviderEarnings: providerEarnings,
		DemandMultiplier: demandMultiplier,
		SupplyBonus:      supplyBonus,
		VRAMPercentage:   vramPercentage,
		AllocatedVRAMGB:  allocatedVRAMGB,
		CalculatedAt:     now,
		ValidUntil:       now.Add(5 * time.Minute), // Pricing valid for 5 minutes
	}

	e.logger.Debug("Pricing calculated",
		zap.String("total_hourly_rate", totalHourlyRate.String()),
		zap.String("total_cost", totalCost.String()),
		zap.String("provider_earnings", providerEarnings.String()),
	)

	return response, nil
}

// getBaseRate gets the base hourly rate for a GPU model
func (e *Engine) getBaseRate(gpuModel string) (decimal.Decimal, error) {
	normalizedModel := strings.ToLower(strings.TrimSpace(gpuModel))

	// Try exact match first
	if rate, exists := e.baseRates[normalizedModel]; exists {
		return rate, nil
	}

	// Try partial matches for common GPU naming variations
	for model, rate := range e.baseRates {
		if strings.Contains(normalizedModel, model) || strings.Contains(model, normalizedModel) {
			return rate, nil
		}
	}

	// Check for GPU family matches
	if rate := e.getGPUFamilyRate(normalizedModel); !rate.IsZero() {
		return rate, nil
	}

	// Use default rate if no match found
	if defaultRate, exists := e.baseRates["default"]; exists {
		e.logger.Warn("Using default rate for unknown GPU model", zap.String("gpu_model", gpuModel))
		return defaultRate, nil
	}

	return decimal.Zero, fmt.Errorf("no pricing available for GPU model: %s", gpuModel)
}

// getGPUFamilyRate attempts to match GPU families for pricing
func (e *Engine) getGPUFamilyRate(gpuModel string) decimal.Decimal {
	// NVIDIA RTX 40 series
	if strings.Contains(gpuModel, "rtx-40") || strings.Contains(gpuModel, "rtx 40") {
		if strings.Contains(gpuModel, "4090") {
			return e.baseRates["nvidia-geforce-rtx-4090"]
		} else if strings.Contains(gpuModel, "4080") {
			return e.baseRates["nvidia-geforce-rtx-4080"]
		} else if strings.Contains(gpuModel, "4070") {
			return e.baseRates["nvidia-geforce-rtx-4070"]
		}
	}

	// NVIDIA RTX 30 series
	if strings.Contains(gpuModel, "rtx-30") || strings.Contains(gpuModel, "rtx 30") {
		if strings.Contains(gpuModel, "3090") {
			return e.baseRates["nvidia-geforce-rtx-3090"]
		} else if strings.Contains(gpuModel, "3080") {
			return e.baseRates["nvidia-geforce-rtx-3080"]
		}
	}

	// NVIDIA Tesla/Data Center
	if strings.Contains(gpuModel, "tesla") || strings.Contains(gpuModel, "a100") || strings.Contains(gpuModel, "h100") {
		if strings.Contains(gpuModel, "h100") {
			return e.baseRates["nvidia-tesla-h100"]
		} else if strings.Contains(gpuModel, "a100") {
			return e.baseRates["nvidia-tesla-a100"]
		} else if strings.Contains(gpuModel, "v100") {
			return e.baseRates["nvidia-tesla-v100"]
		}
	}

	// Apple Silicon
	if strings.Contains(gpuModel, "apple") || strings.Contains(gpuModel, "m1") || strings.Contains(gpuModel, "m2") || strings.Contains(gpuModel, "m3") {
		if strings.Contains(gpuModel, "m3") {
			if strings.Contains(gpuModel, "ultra") {
				return e.baseRates["apple-m3-ultra"]
			} else if strings.Contains(gpuModel, "max") {
				return e.baseRates["apple-m3-max"]
			}
		} else if strings.Contains(gpuModel, "m2") {
			if strings.Contains(gpuModel, "ultra") {
				return e.baseRates["apple-m2-ultra"]
			} else if strings.Contains(gpuModel, "max") {
				return e.baseRates["apple-m2-max"]
			}
		} else if strings.Contains(gpuModel, "m1") {
			if strings.Contains(gpuModel, "ultra") {
				return e.baseRates["apple-m1-ultra"]
			} else if strings.Contains(gpuModel, "max") {
				return e.baseRates["apple-m1-max"]
			}
		}
	}

	// AMD Radeon
	if strings.Contains(gpuModel, "amd") || strings.Contains(gpuModel, "radeon") {
		if strings.Contains(gpuModel, "7900") {
			return e.baseRates["amd-radeon-rx-7900-xtx"]
		} else if strings.Contains(gpuModel, "6900") {
			return e.baseRates["amd-radeon-rx-6900-xt"]
		}
	}

	return decimal.Zero
}

// getDynamicPricingFactors calculates demand and supply factors for dynamic pricing
func (e *Engine) getDynamicPricingFactors(ctx context.Context, req *PricingRequest) (demandMultiplier, supplyBonus decimal.Decimal, err error) {
	// This would typically query the provider registry and current usage statistics
	// For now, return default values
	// TODO: Implement actual demand/supply analysis based on:
	// - Current GPU utilization across the platform
	// - Number of available providers with similar specs
	// - Historical demand patterns
	// - Time of day/week factors
	// - Geographic demand distribution

	demandMultiplier = decimal.NewFromInt(1) // No demand adjustment
	supplyBonus = decimal.Zero               // No supply bonus

	return demandMultiplier, supplyBonus, nil
}

// ValidatePricingRequest validates a pricing request
func (e *Engine) ValidatePricingRequest(req *PricingRequest) error {
	if req.GPUModel == "" {
		return fmt.Errorf("GPU model is required")
	}

	if req.RequestedVRAM == 0 {
		return fmt.Errorf("requested VRAM must be greater than 0")
	}

	if req.TotalVRAM == 0 {
		return fmt.Errorf("total VRAM must be greater than 0")
	}

	if req.RequestedVRAM > req.TotalVRAM {
		return fmt.Errorf("requested VRAM cannot exceed total VRAM")
	}

	if req.EstimatedPowerW == 0 {
		return fmt.Errorf("estimated power consumption must be greater than 0")
	}

	if req.DurationHours.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("duration must be greater than 0")
	}

	minDuration := decimal.NewFromInt(int64(e.config.MinimumSessionMinutes)).Div(decimal.NewFromInt(60))
	if req.DurationHours.LessThan(minDuration) {
		return fmt.Errorf("duration must be at least %d minutes", e.config.MinimumSessionMinutes)
	}

	maxDuration := decimal.NewFromInt(int64(e.config.MaximumSessionHours))
	if req.DurationHours.GreaterThan(maxDuration) {
		return fmt.Errorf("duration cannot exceed %d hours", e.config.MaximumSessionHours)
	}

	return nil
}

// GetSupportedGPUModels returns a list of supported GPU models and their base rates
func (e *Engine) GetSupportedGPUModels() map[string]decimal.Decimal {
	result := make(map[string]decimal.Decimal)
	for model, rate := range e.baseRates {
		if model != "default" {
			result[model] = rate
		}
	}
	return result
}

// GetVRAMRatePerGB returns the VRAM rate per GB per hour
func (e *Engine) GetVRAMRatePerGB() decimal.Decimal {
	return e.config.VRAMRatePerGB
}

// GetPowerMultiplier returns the power consumption multiplier
func (e *Engine) GetPowerMultiplier() decimal.Decimal {
	return e.config.PowerMultiplier
}

// GetPlatformFeePercent returns the platform fee percentage
func (e *Engine) GetPlatformFeePercent() decimal.Decimal {
	return e.config.PlatformFeePercent
}
