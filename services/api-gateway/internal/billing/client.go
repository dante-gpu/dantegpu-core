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

// WalletCreateRequest represents a request to create a new wallet
type WalletCreateRequest struct {
	UserID        string `json:"user_id"`
	WalletType    string `json:"wallet_type"`
	SolanaAddress string `json:"solana_address"`
}

// WalletResponse represents a wallet response
type WalletResponse struct {
	ID              uuid.UUID       `json:"id"`
	UserID          string          `json:"user_id"`
	WalletType      string          `json:"wallet_type"`
	SolanaAddress   string          `json:"solana_address"`
	Balance         decimal.Decimal `json:"balance"`
	LockedBalance   decimal.Decimal `json:"locked_balance"`
	PendingBalance  decimal.Decimal `json:"pending_balance"`
	IsActive        bool            `json:"is_active"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	LastActivityAt  *time.Time      `json:"last_activity_at,omitempty"`
}

// BalanceResponse represents a wallet balance response
type BalanceResponse struct {
	WalletID         uuid.UUID       `json:"wallet_id"`
	Balance          decimal.Decimal `json:"balance"`
	LockedBalance    decimal.Decimal `json:"locked_balance"`
	PendingBalance   decimal.Decimal `json:"pending_balance"`
	AvailableBalance decimal.Decimal `json:"available_balance"`
	TotalBalance     decimal.Decimal `json:"total_balance"`
	LastUpdated      time.Time       `json:"last_updated"`
}

// DepositRequest represents a request to deposit dGPU tokens
type DepositRequest struct {
	WalletID        uuid.UUID       `json:"wallet_id"`
	Amount          decimal.Decimal `json:"amount"`
	SolanaSignature string          `json:"solana_signature"`
}

// WithdrawalRequest represents a request to withdraw dGPU tokens
type WithdrawalRequest struct {
	WalletID  uuid.UUID       `json:"wallet_id"`
	Amount    decimal.Decimal `json:"amount"`
	ToAddress string          `json:"to_address"`
}

// TransactionResponse represents a transaction response
type TransactionResponse struct {
	ID              uuid.UUID       `json:"id"`
	FromWalletID    *uuid.UUID      `json:"from_wallet_id,omitempty"`
	ToWalletID      *uuid.UUID      `json:"to_wallet_id,omitempty"`
	Type            string          `json:"type"`
	Status          string          `json:"status"`
	Amount          decimal.Decimal `json:"amount"`
	Fee             decimal.Decimal `json:"fee"`
	Description     string          `json:"description"`
	SolanaSignature *string         `json:"solana_signature,omitempty"`
	SessionID       *uuid.UUID      `json:"session_id,omitempty"`
	JobID           *string         `json:"job_id,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	ConfirmedAt     *time.Time      `json:"confirmed_at,omitempty"`
}

// CreateWallet creates a new dGPU token wallet
func (c *Client) CreateWallet(ctx context.Context, req *WalletCreateRequest) (*WalletResponse, error) {
	c.logger.Info("Creating wallet",
		zap.String("user_id", req.UserID),
		zap.String("wallet_type", req.WalletType),
	)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal wallet create request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/wallet", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("billing service returned status %d", resp.StatusCode)
	}

	var walletResp WalletResponse
	if err := json.NewDecoder(resp.Body).Decode(&walletResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("Wallet created successfully", zap.String("wallet_id", walletResp.ID.String()))
	return &walletResp, nil
}

// GetWalletBalance gets the current balance of a wallet
func (c *Client) GetWalletBalance(ctx context.Context, walletID uuid.UUID) (*BalanceResponse, error) {
	url := fmt.Sprintf("%s/api/v1/wallet/%s/balance", c.baseURL, walletID.String())
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("billing service returned status %d", resp.StatusCode)
	}

	var balanceResp BalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&balanceResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &balanceResp, nil
}

// DepositTokens processes a dGPU token deposit
func (c *Client) DepositTokens(ctx context.Context, walletID uuid.UUID, req *DepositRequest) (*TransactionResponse, error) {
	c.logger.Info("Processing token deposit",
		zap.String("wallet_id", walletID.String()),
		zap.String("amount", req.Amount.String()),
	)

	req.WalletID = walletID

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deposit request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/wallet/%s/deposit", c.baseURL, walletID.String())
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to deposit tokens: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("billing service returned status %d", resp.StatusCode)
	}

	var txnResp TransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&txnResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("Token deposit processed successfully", zap.String("transaction_id", txnResp.ID.String()))
	return &txnResp, nil
}

// WithdrawTokens processes a dGPU token withdrawal
func (c *Client) WithdrawTokens(ctx context.Context, walletID uuid.UUID, req *WithdrawalRequest) (*TransactionResponse, error) {
	c.logger.Info("Processing token withdrawal",
		zap.String("wallet_id", walletID.String()),
		zap.String("amount", req.Amount.String()),
		zap.String("to_address", req.ToAddress),
	)

	req.WalletID = walletID

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal withdrawal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/wallet/%s/withdraw", c.baseURL, walletID.String())
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to withdraw tokens: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("billing service returned status %d", resp.StatusCode)
	}

	var txnResp TransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&txnResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("Token withdrawal processed successfully", zap.String("transaction_id", txnResp.ID.String()))
	return &txnResp, nil
}

// GetPricingRates gets current GPU rental pricing rates
func (c *Client) GetPricingRates(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/pricing/rates", c.baseURL)
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("billing service returned status %d", resp.StatusCode)
	}

	var rates map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rates); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return rates, nil
}

// CalculatePricing calculates pricing for specific GPU requirements
func (c *Client) CalculatePricing(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pricing request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/pricing/calculate", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate pricing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("billing service returned status %d", resp.StatusCode)
	}

	var pricing map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&pricing); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return pricing, nil
}
