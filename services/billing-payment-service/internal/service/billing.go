package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/models"
	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/pricing"
	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/solana"
	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/store"
)

// BillingService handles all billing and payment operations
type BillingService struct {
	store         *store.PostgresStore
	solanaClient  *solana.Client
	pricingEngine *pricing.Engine
	logger        *zap.Logger
	config        *Config
}

// Config represents billing service configuration
type Config struct {
	MinimumBalance         decimal.Decimal `yaml:"minimum_balance"`
	LowBalanceThreshold    decimal.Decimal `yaml:"low_balance_threshold"`
	BillingInterval        time.Duration   `yaml:"billing_interval"`
	InsufficientFundsGrace time.Duration   `yaml:"insufficient_funds_grace_period"`
	MaxTransactionAmount   decimal.Decimal `yaml:"max_transaction_amount"`
	DailyWithdrawalLimit   decimal.Decimal `yaml:"daily_withdrawal_limit"`
	MinimumPayoutAmount    decimal.Decimal `yaml:"minimum_payout_amount"`
	PayoutFeePercent       decimal.Decimal `yaml:"payout_fee_percent"`
}

// NewBillingService creates a new billing service
func NewBillingService(
	store *store.PostgresStore,
	solanaClient *solana.Client,
	pricingEngine *pricing.Engine,
	config *Config,
	logger *zap.Logger,
) *BillingService {
	return &BillingService{
		store:         store,
		solanaClient:  solanaClient,
		pricingEngine: pricingEngine,
		config:        config,
		logger:        logger,
	}
}

// Wallet Management

// CreateWallet creates a new dGPU token wallet for a user or provider
func (s *BillingService) CreateWallet(ctx context.Context, req *models.WalletCreateRequest) (*models.Wallet, error) {
	s.logger.Info("Creating wallet",
		zap.String("user_id", req.UserID),
		zap.String("wallet_type", string(req.WalletType)),
		zap.String("solana_address", req.SolanaAddress),
	)

	// Validate Solana address format
	if !s.isValidSolanaAddress(req.SolanaAddress) {
		return nil, models.NewValidationError("solana_address", "invalid Solana address format")
	}

	// Check if wallet already exists for this user and type
	existingWallet, err := s.store.GetWalletByUserID(ctx, req.UserID, req.WalletType)
	if err != nil && err != models.ErrWalletNotFound {
		return nil, fmt.Errorf("failed to check existing wallet: %w", err)
	}
	if existingWallet != nil {
		return nil, models.NewBillingError(models.ErrCodeWalletExists, "Wallet already exists", models.ErrWalletAlreadyExists)
	}

	// Create associated token account on Solana if needed
	_, err = s.solanaClient.CreateAssociatedTokenAccount(ctx, req.SolanaAddress)
	if err != nil {
		s.logger.Error("Failed to create associated token account", zap.Error(err))
		return nil, models.NewSolanaError("create_ata", err)
	}

	// Create wallet in database
	wallet, err := s.store.CreateWallet(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	s.logger.Info("Wallet created successfully", zap.String("wallet_id", wallet.ID.String()))
	return wallet, nil
}

// GetWalletBalance gets the current balance of a wallet
func (s *BillingService) GetWalletBalance(ctx context.Context, walletID uuid.UUID) (*models.BalanceResponse, error) {
	wallet, err := s.store.GetWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}

	// Get real-time balance from Solana
	solanaBalance, err := s.solanaClient.GetTokenBalance(ctx, wallet.SolanaAddress)
	if err != nil {
		s.logger.Warn("Failed to get Solana balance, using database balance", zap.Error(err))
		solanaBalance = wallet.Balance
	}

	// Update database balance if there's a significant difference
	if solanaBalance.Sub(wallet.Balance).Abs().GreaterThan(decimal.NewFromFloat(0.001)) {
		err = s.store.UpdateWalletBalance(ctx, walletID, solanaBalance, wallet.LockedBalance)
		if err != nil {
			s.logger.Warn("Failed to update wallet balance", zap.Error(err))
		} else {
			wallet.Balance = solanaBalance
		}
	}

	return &models.BalanceResponse{
		WalletID:         wallet.ID,
		Balance:          wallet.Balance,
		LockedBalance:    wallet.LockedBalance,
		PendingBalance:   wallet.PendingBalance,
		AvailableBalance: wallet.AvailableBalance(),
		TotalBalance:     wallet.TotalBalance(),
		LastUpdated:      wallet.UpdatedAt,
	}, nil
}

// Session Management

// StartRentalSession starts a new GPU rental session
func (s *BillingService) StartRentalSession(ctx context.Context, req *models.SessionStartRequest) (*models.SessionResponse, error) {
	s.logger.Info("Starting rental session",
		zap.String("user_id", req.UserID),
		zap.String("provider_id", req.ProviderID.String()),
		zap.String("gpu_model", req.GPUModel),
		zap.Uint64("requested_vram_mb", req.RequestedVRAM),
	)

	// Get user wallet
	userWallet, err := s.store.GetWalletByUserID(ctx, req.UserID, models.WalletTypeUser)
	if err != nil {
		return nil, err
	}

	// Check minimum balance
	if userWallet.AvailableBalance().LessThan(s.config.MinimumBalance) {
		return nil, models.NewInsufficientFundsError(
			s.config.MinimumBalance.String(),
			userWallet.AvailableBalance().String(),
		)
	}

	// Calculate pricing for initial hour
	pricingReq := &pricing.PricingRequest{
		GPUModel:        req.GPUModel,
		RequestedVRAM:   req.RequestedVRAM,
		TotalVRAM:       req.RequestedVRAM, // This should come from provider registry
		EstimatedPowerW: req.EstimatedPowerW,
		DurationHours:   decimal.NewFromInt(1),
		ProviderID:      &req.ProviderID,
		UserID:          &req.UserID,
	}

	pricing, err := s.pricingEngine.CalculatePricing(ctx, pricingReq)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate pricing: %w", err)
	}

	// Check if user can afford at least one hour
	if userWallet.AvailableBalance().LessThan(pricing.TotalHourlyRate) {
		return nil, models.NewInsufficientFundsError(
			pricing.TotalHourlyRate.String(),
			userWallet.AvailableBalance().String(),
		)
	}

	// Lock funds for initial hour
	err = userWallet.LockFunds(pricing.TotalHourlyRate)
	if err != nil {
		return nil, err
	}

	// Update wallet in database
	err = s.store.UpdateWalletBalance(ctx, userWallet.ID, userWallet.Balance, userWallet.LockedBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to lock funds: %w", err)
	}

	// Create rental session
	session := &models.RentalSession{
		ID:               uuid.New(),
		UserID:           req.UserID,
		ProviderID:       req.ProviderID,
		JobID:            req.JobID,
		Status:           models.SessionStatusActive,
		GPUModel:         req.GPUModel,
		AllocatedVRAM:    req.RequestedVRAM,
		TotalVRAM:        req.RequestedVRAM,       // This should come from provider registry
		VRAMPercentage:   decimal.NewFromInt(100), // Assuming full allocation for now
		HourlyRate:       pricing.BaseHourlyRate,
		VRAMRate:         pricing.VRAMHourlyRate,
		PowerRate:        pricing.PowerHourlyRate,
		PlatformFeeRate:  decimal.NewFromFloat(5.0), // From config
		EstimatedPowerW:  req.EstimatedPowerW,
		StartedAt:        time.Now().UTC(),
		LastBilledAt:     time.Now().UTC(),
		TotalCost:        decimal.Zero,
		PlatformFee:      decimal.Zero,
		ProviderEarnings: decimal.Zero,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	// Save session to database
	err = s.store.CreateRentalSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create rental session: %w", err)
	}

	// Create initial transaction record
	txnReq := &models.TransactionCreateRequest{
		FromWalletID: &userWallet.ID,
		Type:         models.TransactionTypeSessionStart,
		Amount:       pricing.TotalHourlyRate,
		Description:  fmt.Sprintf("Session start - locked funds for %s", req.GPUModel),
		SessionID:    &session.ID,
		JobID:        req.JobID,
	}

	_, err = s.store.CreateTransaction(ctx, txnReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Calculate estimated runtime based on available balance
	estimatedRuntime := userWallet.AvailableBalance().Div(pricing.TotalHourlyRate)

	response := &models.SessionResponse{
		Session:             *session,
		CurrentCost:         decimal.Zero,
		EstimatedHourlyCost: pricing.TotalHourlyRate,
		RemainingBalance:    userWallet.AvailableBalance(),
		EstimatedRuntime:    estimatedRuntime,
	}

	s.logger.Info("Rental session started successfully",
		zap.String("session_id", session.ID.String()),
		zap.String("estimated_runtime_hours", estimatedRuntime.String()),
	)

	return response, nil
}

// ProcessUsageUpdate processes real-time usage data from provider daemon
func (s *BillingService) ProcessUsageUpdate(ctx context.Context, req *models.UsageUpdateRequest) error {
	s.logger.Debug("Processing usage update",
		zap.String("session_id", req.SessionID.String()),
		zap.Uint8("gpu_utilization", req.GPUUtilization),
		zap.Uint32("power_draw", req.PowerDraw),
	)

	// Get session
	session, err := s.store.GetRentalSession(ctx, req.SessionID)
	if err != nil {
		return err
	}

	// Calculate period cost based on current session rates
	periodHours := decimal.NewFromInt(1).Div(decimal.NewFromInt(60)) // 1 minute = 1/60 hour

	// Base cost for this period
	baseCost := session.HourlyRate.Mul(periodHours)

	// VRAM cost for this period
	vramGB := decimal.NewFromInt(int64(session.AllocatedVRAM)).Div(decimal.NewFromInt(1024))
	vramCost := session.VRAMRate.Mul(vramGB).Mul(periodHours)

	// Power cost for this period (use actual power if available, otherwise estimated)
	powerW := decimal.NewFromInt(int64(req.PowerDraw))
	powerCost := session.PowerRate.Mul(powerW).Div(decimal.NewFromInt(1000)).Mul(periodHours) // Convert W to kW

	periodCost := baseCost.Add(vramCost).Add(powerCost)

	// Create usage record
	usageRecord := &models.UsageRecord{
		ID:              uuid.New(),
		SessionID:       req.SessionID,
		RecordedAt:      req.Timestamp,
		GPUUtilization:  req.GPUUtilization,
		VRAMUtilization: req.VRAMUtilization,
		PowerDraw:       req.PowerDraw,
		Temperature:     req.Temperature,
		PeriodMinutes:   1, // 1-minute intervals
		PeriodCost:      periodCost,
		CreatedAt:       time.Now().UTC(),
	}

	// Save usage record
	err = s.store.CreateUsageRecord(ctx, usageRecord)
	if err != nil {
		return fmt.Errorf("failed to create usage record: %w", err)
	}

	// Update session with actual power consumption and total cost
	if session.ActualPowerW == nil || *session.ActualPowerW != req.PowerDraw {
		actualPower := req.PowerDraw
		session.ActualPowerW = &actualPower
		session.TotalCost = session.TotalCost.Add(periodCost)
		session.UpdatedAt = time.Now().UTC()

		err = s.store.UpdateRentalSession(ctx, session)
		if err != nil {
			s.logger.Warn("Failed to update session with actual power", zap.Error(err))
		}
	}

	s.logger.Debug("Usage update processed successfully")
	return nil
}

// Helper methods

// EndRentalSession ends a rental session and processes final billing
func (s *BillingService) EndRentalSession(ctx context.Context, req *models.SessionEndRequest) (*models.SessionResponse, error) {
	s.logger.Info("Ending rental session", zap.String("session_id", req.SessionID.String()))

	// Get session
	session, err := s.store.GetRentalSession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	if session.Status != models.SessionStatusActive {
		return nil, models.NewBillingError(models.ErrCodeSessionNotActive, "Session is not active", models.ErrSessionNotActive)
	}

	// Calculate final costs
	now := time.Now().UTC()
	session.EndedAt = &now
	session.Status = models.SessionStatusCompleted

	// Calculate total session cost
	totalCost := session.CalculateCurrentCost()
	platformFee := totalCost.Mul(session.PlatformFeeRate).Div(decimal.NewFromInt(100))
	providerEarnings := totalCost.Sub(platformFee)

	session.TotalCost = totalCost
	session.PlatformFee = platformFee
	session.ProviderEarnings = providerEarnings
	session.UpdatedAt = now

	// Update session in database
	err = s.store.UpdateRentalSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Get user wallet to process final payment
	userWallet, err := s.store.GetWalletByUserID(ctx, session.UserID, models.WalletTypeUser)
	if err != nil {
		return nil, err
	}

	// Unlock any remaining locked funds and deduct actual cost
	userWallet.UnlockFunds(userWallet.LockedBalance)
	err = userWallet.DeductFunds(totalCost)
	if err != nil {
		s.logger.Error("Failed to deduct final session cost", zap.Error(err))
		return nil, err
	}

	// Update wallet balance
	err = s.store.UpdateWalletBalance(ctx, userWallet.ID, userWallet.Balance, userWallet.LockedBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Create final transaction
	txnReq := &models.TransactionCreateRequest{
		FromWalletID: &userWallet.ID,
		Type:         models.TransactionTypeSessionEnd,
		Amount:       totalCost,
		Description:  fmt.Sprintf("Session end - final payment for %s", session.GPUModel),
		SessionID:    &session.ID,
	}

	_, err = s.store.CreateTransaction(ctx, txnReq)
	if err != nil {
		s.logger.Error("Failed to create final transaction", zap.Error(err))
	}

	response := &models.SessionResponse{
		Session:             *session,
		CurrentCost:         totalCost,
		EstimatedHourlyCost: decimal.Zero,
		RemainingBalance:    userWallet.AvailableBalance(),
		EstimatedRuntime:    decimal.Zero,
	}

	s.logger.Info("Rental session ended successfully",
		zap.String("session_id", session.ID.String()),
		zap.String("total_cost", totalCost.String()),
		zap.String("duration", session.Duration().String()),
	)

	return response, nil
}

// GetCurrentUsage gets current usage and cost for an active session
func (s *BillingService) GetCurrentUsage(ctx context.Context, sessionID uuid.UUID) (*models.SessionResponse, error) {
	session, err := s.store.GetRentalSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	currentCost := session.CalculateCurrentCost()

	// Get user wallet for remaining balance
	userWallet, err := s.store.GetWalletByUserID(ctx, session.UserID, models.WalletTypeUser)
	if err != nil {
		return nil, err
	}

	// Calculate estimated runtime based on current hourly rate
	hourlyRate := session.HourlyRate
	if session.VRAMRate.GreaterThan(decimal.Zero) {
		vramGB := decimal.NewFromInt(int64(session.AllocatedVRAM)).Div(decimal.NewFromInt(1024))
		hourlyRate = hourlyRate.Add(session.VRAMRate.Mul(vramGB))
	}
	if session.PowerRate.GreaterThan(decimal.Zero) && session.ActualPowerW != nil {
		powerKW := decimal.NewFromInt(int64(*session.ActualPowerW)).Div(decimal.NewFromInt(1000))
		hourlyRate = hourlyRate.Add(session.PowerRate.Mul(powerKW))
	}

	estimatedRuntime := decimal.Zero
	if hourlyRate.GreaterThan(decimal.Zero) {
		estimatedRuntime = userWallet.AvailableBalance().Div(hourlyRate)
	}

	return &models.SessionResponse{
		Session:             *session,
		CurrentCost:         currentCost,
		EstimatedHourlyCost: hourlyRate,
		RemainingBalance:    userWallet.AvailableBalance(),
		EstimatedRuntime:    estimatedRuntime,
	}, nil
}

// GetBillingHistory retrieves billing history for a user or provider
func (s *BillingService) GetBillingHistory(ctx context.Context, req *models.BillingHistoryRequest) (*models.BillingHistoryResponse, error) {
	return s.store.GetBillingHistory(ctx, req)
}

// GetProviderEarnings retrieves earnings information for a provider
func (s *BillingService) GetProviderEarnings(ctx context.Context, req *models.ProviderEarningsRequest) (*models.ProviderEarningsResponse, error) {
	return s.store.GetProviderEarnings(ctx, req)
}

// ProcessDeposit processes a dGPU token deposit
func (s *BillingService) ProcessDeposit(ctx context.Context, req *models.DepositRequest) (*models.Transaction, error) {
	s.logger.Info("Processing deposit",
		zap.String("wallet_id", req.WalletID.String()),
		zap.String("amount", req.Amount.String()),
		zap.String("signature", req.SolanaSignature),
	)

	// Verify the Solana transaction
	err := s.solanaClient.ConfirmTransaction(ctx, req.SolanaSignature)
	if err != nil {
		return nil, models.NewSolanaError("confirm_deposit", err)
	}

	// Get wallet
	wallet, err := s.store.GetWallet(ctx, req.WalletID)
	if err != nil {
		return nil, err
	}

	// Add funds to wallet
	wallet.AddFunds(req.Amount)

	// Update wallet balance
	err = s.store.UpdateWalletBalance(ctx, wallet.ID, wallet.Balance, wallet.LockedBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Create transaction record
	txnReq := &models.TransactionCreateRequest{
		ToWalletID:  &wallet.ID,
		Type:        models.TransactionTypeDeposit,
		Amount:      req.Amount,
		Description: "dGPU token deposit",
		Metadata: map[string]interface{}{
			"solana_signature": req.SolanaSignature,
		},
	}

	transaction, err := s.store.CreateTransaction(ctx, txnReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update transaction with confirmed status
	err = s.store.UpdateTransactionStatus(ctx, transaction.ID, models.TransactionStatusConfirmed, &req.SolanaSignature)
	if err != nil {
		s.logger.Warn("Failed to update transaction status", zap.Error(err))
	}

	s.logger.Info("Deposit processed successfully",
		zap.String("wallet_id", wallet.ID.String()),
		zap.String("amount", req.Amount.String()),
	)

	return transaction, nil
}

// ProcessWithdrawal processes a dGPU token withdrawal
func (s *BillingService) ProcessWithdrawal(ctx context.Context, req *models.WithdrawalRequest) (*models.Transaction, error) {
	s.logger.Info("Processing withdrawal",
		zap.String("wallet_id", req.WalletID.String()),
		zap.String("amount", req.Amount.String()),
		zap.String("to_address", req.ToAddress),
	)

	// Get wallet
	wallet, err := s.store.GetWallet(ctx, req.WalletID)
	if err != nil {
		return nil, err
	}

	// Check if wallet has sufficient funds
	if !wallet.CanSpend(req.Amount) {
		return nil, models.NewInsufficientFundsError(req.Amount.String(), wallet.AvailableBalance().String())
	}

	// Check daily withdrawal limit
	if req.Amount.GreaterThan(s.config.DailyWithdrawalLimit) {
		return nil, models.NewValidationError("amount", "exceeds daily withdrawal limit")
	}

	// Create transaction record first
	txnReq := &models.TransactionCreateRequest{
		FromWalletID: &wallet.ID,
		Type:         models.TransactionTypeWithdrawal,
		Amount:       req.Amount,
		Description:  fmt.Sprintf("dGPU token withdrawal to %s", req.ToAddress),
		Metadata: map[string]interface{}{
			"to_address": req.ToAddress,
		},
	}

	transaction, err := s.store.CreateTransaction(ctx, txnReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Execute Solana transfer
	signature, err := s.solanaClient.TransferTokens(ctx, wallet.SolanaAddress, req.ToAddress, req.Amount)
	if err != nil {
		// Update transaction as failed
		s.store.UpdateTransactionStatus(ctx, transaction.ID, models.TransactionStatusFailed, nil)
		return nil, models.NewSolanaError("transfer_tokens", err)
	}

	// Deduct funds from wallet
	err = wallet.DeductFunds(req.Amount)
	if err != nil {
		s.logger.Error("Failed to deduct funds after successful transfer", zap.Error(err))
		return nil, err
	}

	// Update wallet balance
	err = s.store.UpdateWalletBalance(ctx, wallet.ID, wallet.Balance, wallet.LockedBalance)
	if err != nil {
		s.logger.Error("Failed to update wallet balance after withdrawal", zap.Error(err))
		return nil, err
	}

	// Update transaction with confirmed status and signature
	err = s.store.UpdateTransactionStatus(ctx, transaction.ID, models.TransactionStatusConfirmed, &signature)
	if err != nil {
		s.logger.Warn("Failed to update transaction status", zap.Error(err))
	}

	s.logger.Info("Withdrawal processed successfully",
		zap.String("wallet_id", wallet.ID.String()),
		zap.String("amount", req.Amount.String()),
		zap.String("signature", signature),
	)

	return transaction, nil
}

// GetTransactionHistory retrieves transaction history for a wallet
func (s *BillingService) GetTransactionHistory(ctx context.Context, req *models.TransactionHistoryRequest) (*models.TransactionHistoryResponse, error) {
	// This would need to be implemented in the store
	// For now, return empty response
	return &models.TransactionHistoryResponse{
		Transactions: []models.Transaction{},
		Total:        0,
		Limit:        req.Limit,
		Offset:       req.Offset,
	}, nil
}

// CalculatePricing calculates pricing for GPU rental requirements
func (s *BillingService) CalculatePricing(ctx context.Context, req interface{}) (*pricing.PricingResponse, error) {
	// Convert the interface{} to proper pricing request
	pricingReq := &pricing.PricingRequest{}

	// This is a simplified conversion - in production, use proper type assertion or JSON marshaling
	if reqMap, ok := req.(map[string]interface{}); ok {
		if gpuModel, ok := reqMap["gpu_model"].(string); ok {
			pricingReq.GPUModel = gpuModel
		}
		if requestedVRAM, ok := reqMap["requested_vram_mb"].(uint64); ok {
			pricingReq.RequestedVRAM = requestedVRAM
		}
		if totalVRAM, ok := reqMap["total_vram_mb"].(uint64); ok {
			pricingReq.TotalVRAM = totalVRAM
		}
		if estimatedPowerW, ok := reqMap["estimated_power_w"].(uint32); ok {
			pricingReq.EstimatedPowerW = estimatedPowerW
		}
		if durationHours, ok := reqMap["duration_hours"].(decimal.Decimal); ok {
			pricingReq.DurationHours = durationHours
		}
	}

	// Set defaults if not provided
	if pricingReq.TotalVRAM == 0 {
		pricingReq.TotalVRAM = pricingReq.RequestedVRAM
	}
	if pricingReq.DurationHours.IsZero() {
		pricingReq.DurationHours = decimal.NewFromInt(1) // Default to 1 hour
	}

	// Validate the request
	if err := s.pricingEngine.ValidatePricingRequest(pricingReq); err != nil {
		return nil, fmt.Errorf("invalid pricing request: %w", err)
	}

	// Calculate pricing
	return s.pricingEngine.CalculatePricing(ctx, pricingReq)
}

// GetPricingRates gets current pricing rates for all supported GPU models
func (s *BillingService) GetPricingRates(ctx context.Context) (map[string]interface{}, error) {
	supportedGPUs := s.pricingEngine.GetSupportedGPUModels()

	response := map[string]interface{}{
		"base_rates":           supportedGPUs,
		"vram_rate_per_gb":     s.pricingEngine.GetVRAMRatePerGB(),
		"power_multiplier":     s.pricingEngine.GetPowerMultiplier(),
		"platform_fee_percent": s.pricingEngine.GetPlatformFeePercent(),
		"currency":             "dGPU",
		"last_updated":         time.Now().UTC(),
	}

	return response, nil
}

// isValidSolanaAddress validates a Solana address format
func (s *BillingService) isValidSolanaAddress(address string) bool {
	// Basic validation - in production, use proper Solana address validation
	return len(address) >= 32 && len(address) <= 44
}
