package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/dante-gpu/dante-backend/api-gateway/internal/billing"
)

// BillingHandler handles billing-related HTTP requests
type BillingHandler struct {
	billingClient *billing.Client
	logger        *zap.Logger
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(billingClient *billing.Client, logger *zap.Logger) *BillingHandler {
	return &BillingHandler{
		billingClient: billingClient,
		logger:        logger,
	}
}

// CreateWallet handles wallet creation requests
func (h *BillingHandler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	var req billing.WalletCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode wallet creation request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	wallet, err := h.billingClient.CreateWallet(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create wallet", zap.Error(err))
		http.Error(w, "Failed to create wallet", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wallet)
}

// GetWalletBalance handles wallet balance requests
func (h *BillingHandler) GetWalletBalance(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletID")
	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		h.logger.Error("Invalid wallet ID", zap.String("wallet_id", walletIDStr), zap.Error(err))
		http.Error(w, "Invalid wallet ID", http.StatusBadRequest)
		return
	}

	balance, err := h.billingClient.GetWalletBalance(r.Context(), walletID)
	if err != nil {
		h.logger.Error("Failed to get wallet balance", zap.String("wallet_id", walletIDStr), zap.Error(err))
		http.Error(w, "Failed to get wallet balance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

// DepositTokens handles token deposit requests
func (h *BillingHandler) DepositTokens(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletID")
	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		h.logger.Error("Invalid wallet ID", zap.String("wallet_id", walletIDStr), zap.Error(err))
		http.Error(w, "Invalid wallet ID", http.StatusBadRequest)
		return
	}

	var req billing.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode deposit request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	transaction, err := h.billingClient.DepositTokens(r.Context(), walletID, &req)
	if err != nil {
		h.logger.Error("Failed to process deposit", zap.Error(err))
		http.Error(w, "Failed to process deposit", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transaction)
}

// WithdrawTokens handles token withdrawal requests
func (h *BillingHandler) WithdrawTokens(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletID")
	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		h.logger.Error("Invalid wallet ID", zap.String("wallet_id", walletIDStr), zap.Error(err))
		http.Error(w, "Invalid wallet ID", http.StatusBadRequest)
		return
	}

	var req billing.WithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode withdrawal request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	transaction, err := h.billingClient.WithdrawTokens(r.Context(), walletID, &req)
	if err != nil {
		h.logger.Error("Failed to process withdrawal", zap.Error(err))
		http.Error(w, "Failed to process withdrawal", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transaction)
}

// GetPricingRates handles pricing rates requests
func (h *BillingHandler) GetPricingRates(w http.ResponseWriter, r *http.Request) {
	rates, err := h.billingClient.GetPricingRates(r.Context())
	if err != nil {
		h.logger.Error("Failed to get pricing rates", zap.Error(err))
		http.Error(w, "Failed to get pricing rates", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rates)
}

// CalculatePricing handles pricing calculation requests
func (h *BillingHandler) CalculatePricing(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode pricing request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	pricing, err := h.billingClient.CalculatePricing(r.Context(), req)
	if err != nil {
		h.logger.Error("Failed to calculate pricing", zap.Error(err))
		http.Error(w, "Failed to calculate pricing", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pricing)
}

// GetUserWallet gets a user's wallet by user ID
func (h *BillingHandler) GetUserWallet(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// This would need to be implemented in the billing service
	// For now, return a placeholder response
	response := map[string]interface{}{
		"message": "Get user wallet endpoint not yet implemented",
		"user_id": userID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUserBalance gets a user's dGPU token balance
func (h *BillingHandler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// This would need to be implemented in the billing service
	// For now, return a placeholder response
	response := map[string]interface{}{
		"user_id":           userID,
		"balance":           "100.0",
		"locked_balance":    "0.0",
		"available_balance": "100.0",
		"currency":          "dGPU",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetGPUMarketplace gets available GPUs with pricing
func (h *BillingHandler) GetGPUMarketplace(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	gpuType := r.URL.Query().Get("gpu_type")
	minVRAM := r.URL.Query().Get("min_vram")
	maxPrice := r.URL.Query().Get("max_price")

	h.logger.Info("Getting GPU marketplace",
		zap.String("gpu_type", gpuType),
		zap.String("min_vram", minVRAM),
		zap.String("max_price", maxPrice),
	)

	// This would integrate with provider registry and pricing service
	// For now, return a placeholder response
	response := map[string]interface{}{
		"gpus": []map[string]interface{}{
			{
				"provider_id":      "provider-1",
				"gpu_model":        "NVIDIA RTX 4090",
				"vram_total":       24576,
				"vram_available":   24576,
				"hourly_rate":      "0.50",
				"vram_rate_per_gb": "0.02",
				"power_rate":       "0.001",
				"location":         "US-East",
				"availability":     "available",
			},
			{
				"provider_id":      "provider-2",
				"gpu_model":        "NVIDIA A100",
				"vram_total":       81920,
				"vram_available":   81920,
				"hourly_rate":      "2.00",
				"vram_rate_per_gb": "0.025",
				"power_rate":       "0.001",
				"location":         "US-West",
				"availability":     "available",
			},
		},
		"filters": map[string]interface{}{
			"gpu_type":  gpuType,
			"min_vram":  minVRAM,
			"max_price": maxPrice,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// EstimateJobCost estimates the cost of a GPU rental job
func (h *BillingHandler) EstimateJobCost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GPUModel        string  `json:"gpu_model"`
		VRAMRequired    uint64  `json:"vram_required_mb"`
		EstimatedHours  float64 `json:"estimated_hours"`
		EstimatedPowerW uint32  `json:"estimated_power_w"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode cost estimation request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Calculate estimated cost using pricing service
	pricingReq := map[string]interface{}{
		"gpu_model":         req.GPUModel,
		"requested_vram":    req.VRAMRequired,
		"duration_hours":    req.EstimatedHours,
		"estimated_power_w": req.EstimatedPowerW,
	}

	pricing, err := h.billingClient.CalculatePricing(r.Context(), pricingReq)
	if err != nil {
		h.logger.Error("Failed to calculate pricing", zap.Error(err))
		http.Error(w, "Failed to calculate pricing", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"estimated_cost": pricing,
		"currency":       "dGPU",
		"breakdown": map[string]interface{}{
			"base_cost":    "calculated from pricing service",
			"vram_cost":    "calculated from pricing service",
			"power_cost":   "calculated from pricing service",
			"platform_fee": "5% of total",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTransactionHistory gets transaction history for a wallet
func (h *BillingHandler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletID")
	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		h.logger.Error("Invalid wallet ID", zap.String("wallet_id", walletIDStr), zap.Error(err))
		http.Error(w, "Invalid wallet ID", http.StatusBadRequest)
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // Default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0 // Default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	h.logger.Info("Getting transaction history",
		zap.String("wallet_id", walletID.String()),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	// This would need to be implemented in the billing service
	// For now, return a placeholder response
	response := map[string]interface{}{
		"transactions": []map[string]interface{}{},
		"total":        0,
		"limit":        limit,
		"offset":       offset,
		"wallet_id":    walletID.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
