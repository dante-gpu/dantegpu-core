package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// WalletType represents the type of wallet
type WalletType string

const (
	WalletTypeUser     WalletType = "user"
	WalletTypeProvider WalletType = "provider"
	WalletTypePlatform WalletType = "platform"
)

// Wallet represents a dGPU token wallet for users and providers
type Wallet struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	UserID          string          `json:"user_id" db:"user_id"`
	WalletType      WalletType      `json:"wallet_type" db:"wallet_type"`
	SolanaAddress   string          `json:"solana_address" db:"solana_address"`
	Balance         decimal.Decimal `json:"balance" db:"balance"`
	LockedBalance   decimal.Decimal `json:"locked_balance" db:"locked_balance"`   // Funds locked for active sessions
	PendingBalance  decimal.Decimal `json:"pending_balance" db:"pending_balance"` // Pending deposits/withdrawals
	IsActive        bool            `json:"is_active" db:"is_active"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
	LastActivityAt  *time.Time      `json:"last_activity_at,omitempty" db:"last_activity_at"`
}

// AvailableBalance returns the balance available for spending
func (w *Wallet) AvailableBalance() decimal.Decimal {
	return w.Balance.Sub(w.LockedBalance)
}

// TotalBalance returns the total balance including locked and pending
func (w *Wallet) TotalBalance() decimal.Decimal {
	return w.Balance.Add(w.PendingBalance)
}

// CanSpend checks if the wallet has sufficient available balance
func (w *Wallet) CanSpend(amount decimal.Decimal) bool {
	return w.AvailableBalance().GreaterThanOrEqual(amount)
}

// LockFunds locks the specified amount for a session
func (w *Wallet) LockFunds(amount decimal.Decimal) error {
	if !w.CanSpend(amount) {
		return ErrInsufficientFunds
	}
	w.LockedBalance = w.LockedBalance.Add(amount)
	w.UpdatedAt = time.Now().UTC()
	return nil
}

// UnlockFunds unlocks the specified amount
func (w *Wallet) UnlockFunds(amount decimal.Decimal) {
	w.LockedBalance = w.LockedBalance.Sub(amount)
	if w.LockedBalance.LessThan(decimal.Zero) {
		w.LockedBalance = decimal.Zero
	}
	w.UpdatedAt = time.Now().UTC()
}

// DeductFunds deducts the specified amount from the wallet
func (w *Wallet) DeductFunds(amount decimal.Decimal) error {
	if w.Balance.LessThan(amount) {
		return ErrInsufficientFunds
	}
	w.Balance = w.Balance.Sub(amount)
	w.UpdatedAt = time.Now().UTC()
	return nil
}

// AddFunds adds the specified amount to the wallet
func (w *Wallet) AddFunds(amount decimal.Decimal) {
	w.Balance = w.Balance.Add(amount)
	w.UpdatedAt = time.Now().UTC()
}

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeDeposit          TransactionType = "deposit"
	TransactionTypeWithdrawal       TransactionType = "withdrawal"
	TransactionTypePayment          TransactionType = "payment"
	TransactionTypePayout           TransactionType = "payout"
	TransactionTypeRefund           TransactionType = "refund"
	TransactionTypePlatformFee      TransactionType = "platform_fee"
	TransactionTypeSessionStart     TransactionType = "session_start"
	TransactionTypeSessionEnd       TransactionType = "session_end"
	TransactionTypeSessionBilling   TransactionType = "session_billing"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusConfirmed TransactionStatus = "confirmed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusCancelled TransactionStatus = "cancelled"
)

// Transaction represents a dGPU token transaction
type Transaction struct {
	ID                uuid.UUID         `json:"id" db:"id"`
	FromWalletID      *uuid.UUID        `json:"from_wallet_id,omitempty" db:"from_wallet_id"`
	ToWalletID        *uuid.UUID        `json:"to_wallet_id,omitempty" db:"to_wallet_id"`
	Type              TransactionType   `json:"type" db:"type"`
	Status            TransactionStatus `json:"status" db:"status"`
	Amount            decimal.Decimal   `json:"amount" db:"amount"`
	Fee               decimal.Decimal   `json:"fee" db:"fee"`
	Description       string            `json:"description" db:"description"`
	SolanaSignature   *string           `json:"solana_signature,omitempty" db:"solana_signature"`
	SessionID         *uuid.UUID        `json:"session_id,omitempty" db:"session_id"`
	JobID             *string           `json:"job_id,omitempty" db:"job_id"`
	Metadata          map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt         time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at" db:"updated_at"`
	ConfirmedAt       *time.Time        `json:"confirmed_at,omitempty" db:"confirmed_at"`
}

// WalletCreateRequest represents a request to create a new wallet
type WalletCreateRequest struct {
	UserID        string     `json:"user_id" validate:"required"`
	WalletType    WalletType `json:"wallet_type" validate:"required"`
	SolanaAddress string     `json:"solana_address" validate:"required"`
}

// WalletUpdateRequest represents a request to update wallet information
type WalletUpdateRequest struct {
	SolanaAddress *string `json:"solana_address,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
}

// TransactionCreateRequest represents a request to create a new transaction
type TransactionCreateRequest struct {
	FromWalletID    *uuid.UUID             `json:"from_wallet_id,omitempty"`
	ToWalletID      *uuid.UUID             `json:"to_wallet_id,omitempty"`
	Type            TransactionType        `json:"type" validate:"required"`
	Amount          decimal.Decimal        `json:"amount" validate:"required,gt=0"`
	Description     string                 `json:"description" validate:"required"`
	SessionID       *uuid.UUID             `json:"session_id,omitempty"`
	JobID           *string                `json:"job_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
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

// TransactionHistoryRequest represents a request for transaction history
type TransactionHistoryRequest struct {
	WalletID    *uuid.UUID         `json:"wallet_id,omitempty"`
	Type        *TransactionType   `json:"type,omitempty"`
	Status      *TransactionStatus `json:"status,omitempty"`
	StartDate   *time.Time         `json:"start_date,omitempty"`
	EndDate     *time.Time         `json:"end_date,omitempty"`
	Limit       int                `json:"limit,omitempty"`
	Offset      int                `json:"offset,omitempty"`
}

// TransactionHistoryResponse represents a transaction history response
type TransactionHistoryResponse struct {
	Transactions []Transaction `json:"transactions"`
	Total        int           `json:"total"`
	Limit        int           `json:"limit"`
	Offset       int           `json:"offset"`
}

// DepositRequest represents a request to deposit dGPU tokens
type DepositRequest struct {
	WalletID        uuid.UUID       `json:"wallet_id" validate:"required"`
	Amount          decimal.Decimal `json:"amount" validate:"required,gt=0"`
	SolanaSignature string          `json:"solana_signature" validate:"required"`
}

// WithdrawalRequest represents a request to withdraw dGPU tokens
type WithdrawalRequest struct {
	WalletID      uuid.UUID       `json:"wallet_id" validate:"required"`
	Amount        decimal.Decimal `json:"amount" validate:"required,gt=0"`
	ToAddress     string          `json:"to_address" validate:"required"`
}

// PayoutRequest represents a request for provider payout
type PayoutRequest struct {
	ProviderWalletID uuid.UUID       `json:"provider_wallet_id" validate:"required"`
	Amount           decimal.Decimal `json:"amount" validate:"required,gt=0"`
	ToAddress        string          `json:"to_address" validate:"required"`
}
