package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dantegpu/billing-service/internal/blockchain"
	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
)

// WalletHandler handles wallet operations
type WalletHandler struct {
	db             *sql.DB
	solanaClient   *blockchain.SolanaClient
	encryptionKey  []byte
}

// CreateWalletRequest represents wallet creation request
type CreateWalletRequest struct {
	UserID string `json:"user_id"`
}

// WalletResponse represents wallet response
type WalletResponse struct {
	ID              string  `json:"id"`
	Address         string  `json:"address"`
	Balance         float64 `json:"balance"`
	LockedBalance   float64 `json:"locked_balance"`
	AvailableBalance float64 `json:"available_balance"`
	CreatedAt       time.Time `json:"created_at"`
}

// NewWalletHandler creates a new wallet handler
func NewWalletHandler(db *sql.DB, solanaClient *blockchain.SolanaClient, encryptionKey []byte) *WalletHandler {
	return &WalletHandler{
		db:            db,
		solanaClient:  solanaClient,
		encryptionKey: encryptionKey,
	}
}

// CreateWallet creates a new Solana wallet
func (h *WalletHandler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check if wallet already exists
	var exists bool
	err := h.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM wallets WHERE user_id = $1)
	`, userID).Scan(&exists)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check wallet")
		return
	}
	if exists {
		respondError(w, http.StatusConflict, "Wallet already exists")
		return
	}

	// Generate Solana wallet
	privateKey, publicKey, err := h.solanaClient.CreateWallet(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create wallet")
		return
	}

	// Encrypt private key
	encryptedKey, err := encrypt(privateKey.String(), h.encryptionKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to encrypt key")
		return
	}

	// Store wallet
	walletID := uuid.New().String()
	now := time.Now()

	_, err = h.db.ExecContext(ctx, `
		INSERT INTO wallets (
			id, user_id, address, encrypted_private_key, 
			balance, locked_balance, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, walletID, userID, publicKey.String(), encryptedKey, 0, 0, now, now)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to store wallet")
		return
	}

	respondJSON(w, http.StatusCreated, WalletResponse{
		ID:               walletID,
		Address:          publicKey.String(),
		Balance:          0,
		LockedBalance:    0,
		AvailableBalance: 0,
		CreatedAt:        now,
	})
}

// GetWallet gets wallet details
func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var wallet WalletResponse
	err := h.db.QueryRowContext(ctx, `
		SELECT id, address, balance, locked_balance, created_at
		FROM wallets
		WHERE user_id = $1
	`, userID).Scan(&wallet.ID, &wallet.Address, &wallet.Balance, &wallet.LockedBalance, &wallet.CreatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Wallet not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get wallet")
		return
	}

	wallet.AvailableBalance = wallet.Balance - wallet.LockedBalance

	// Get real-time balance from blockchain
	address, _ := solana.PublicKeyFromBase58(wallet.Address)
	blockchainBalance, err := h.solanaClient.GetBalance(ctx, address)
	if err == nil {
		wallet.Balance = float64(blockchainBalance) / 1e9 // Convert lamports to SOL
		wallet.AvailableBalance = wallet.Balance - wallet.LockedBalance
	}

	respondJSON(w, http.StatusOK, wallet)
}

// Deposit initiates a deposit
func (h *WalletHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Get wallet address
	var address string
	err := h.db.QueryRowContext(ctx, `
		SELECT address FROM wallets WHERE user_id = $1
	`, userID).Scan(&address)

	if err != nil {
		respondError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	// Return deposit instructions
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"address": address,
		"amount":  req.Amount,
		"message": "Send dGPU tokens to this address",
		"token_mint": "7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump",
	})
}

// Withdraw processes a withdrawal
func (h *WalletHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Amount  float64 `json:"amount"`
		Address string  `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Get wallet
	var walletID, encryptedKey string
	var balance, lockedBalance float64
	err := h.db.QueryRowContext(ctx, `
		SELECT id, encrypted_private_key, balance, locked_balance
		FROM wallets WHERE user_id = $1
	`, userID).Scan(&walletID, &encryptedKey, &balance, &lockedBalance)

	if err != nil {
		respondError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	// Check available balance
	available := balance - lockedBalance
	if req.Amount > available {
		respondError(w, http.StatusBadRequest, "Insufficient balance")
		return
	}

	// Decrypt private key
	privateKeyStr, err := decrypt(encryptedKey, h.encryptionKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decrypt key")
		return
	}

	privateKey, err := solana.PrivateKeyFromBase58(privateKeyStr)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Invalid private key")
		return
	}

	// Parse destination address
	toAddress, err := solana.PublicKeyFromBase58(req.Address)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid destination address")
		return
	}

	// Convert amount to lamports
	amount := uint64(req.Amount * 1e9)

	// Execute transfer
	result, err := h.solanaClient.Transfer(ctx, privateKey, toAddress, amount)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Transfer failed: "+err.Error())
		return
	}

	// Update balance
	_, err = h.db.ExecContext(ctx, `
		UPDATE wallets
		SET balance = balance - $1, updated_at = $2
		WHERE id = $3
	`, req.Amount, time.Now(), walletID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update balance")
		return
	}

	// Record transaction
	h.recordTransaction(ctx, walletID, "withdrawal", req.Amount, result.Signature)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"signature": result.Signature,
		"amount":    req.Amount,
		"message":   "Withdrawal successful",
	})
}

// GetTransactions gets wallet transaction history
func (h *WalletHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	rows, err := h.db.QueryContext(ctx, `
		SELECT bt.id, bt.transaction_type, bt.amount, bt.signature, 
		       bt.status, bt.created_at
		FROM blockchain_transactions bt
		JOIN wallets w ON w.id = bt.wallet_id
		WHERE w.user_id = $1
		ORDER BY bt.created_at DESC
		LIMIT 100
	`, userID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get transactions")
		return
	}
	defer rows.Close()

	var transactions []map[string]interface{}
	for rows.Next() {
		var tx struct {
			ID        string
			Type      string
			Amount    float64
			Signature string
			Status    string
			CreatedAt time.Time
		}
		rows.Scan(&tx.ID, &tx.Type, &tx.Amount, &tx.Signature, &tx.Status, &tx.CreatedAt)
		
		transactions = append(transactions, map[string]interface{}{
			"id":         tx.ID,
			"type":       tx.Type,
			"amount":     tx.Amount,
			"signature":  tx.Signature,
			"status":     tx.Status,
			"created_at": tx.CreatedAt,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": transactions,
		"count":        len(transactions),
	})
}

// Helper functions

func (h *WalletHandler) recordTransaction(ctx context.Context, walletID, txType string, amount float64, signature string) {
	h.db.ExecContext(ctx, `
		INSERT INTO blockchain_transactions (
			id, wallet_id, transaction_type, amount, signature, 
			status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New().String(), walletID, txType, amount, signature, "confirmed", time.Now())
}

func encrypt(plaintext string, key []byte) (string, error) {
	// Implement AES-256-GCM encryption
	// This is a placeholder - use proper encryption in production
	return plaintext, nil
}

func decrypt(ciphertext string, key []byte) (string, error) {
	// Implement AES-256-GCM decryption
	// This is a placeholder - use proper decryption in production
	return ciphertext, nil
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

