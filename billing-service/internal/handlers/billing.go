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

// BillingHandler handles billing operations
type BillingHandler struct {
	db           *sql.DB
	solanaClient *blockchain.SolanaClient
	platformFee  float64 // 5%
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(db *sql.DB, solanaClient *blockchain.SolanaClient, platformFee float64) *BillingHandler {
	return &BillingHandler{
		db:           db,
		solanaClient: solanaClient,
		platformFee:  platformFee,
	}
}

// StartRental starts a rental session with escrow
func (h *BillingHandler) StartRental(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		ProviderID       string  `json:"provider_id"`
		GPUCapabilityID  string  `json:"gpu_capability_id"`
		JobID            string  `json:"job_id"`
		HourlyRate       float64 `json:"hourly_rate"`
		EstimatedMinutes int     `json:"estimated_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Calculate escrow amount (estimated cost + buffer)
	minuteRate := req.HourlyRate / 60
	escrowAmount := minuteRate * float64(req.EstimatedMinutes) * 1.2 // 20% buffer

	// Get user wallet
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
	if escrowAmount > available {
		respondError(w, http.StatusBadRequest, "Insufficient balance")
		return
	}

	// Lock funds
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Update locked balance
	_, err = tx.ExecContext(ctx, `
		UPDATE wallets
		SET locked_balance = locked_balance + $1, updated_at = $2
		WHERE id = $3
	`, escrowAmount, time.Now(), walletID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to lock funds")
		return
	}

	// Create rental session
	sessionID := uuid.New().String()
	now := time.Now()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO rental_sessions (
			id, user_id, provider_id, gpu_capability_id, job_id,
			hourly_rate, escrow_amount, status, started_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, sessionID, userID, req.ProviderID, req.GPUCapabilityID, req.JobID,
		req.HourlyRate, escrowAmount, "active", now, now)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Create escrow transaction
	_, err = tx.ExecContext(ctx, `
		INSERT INTO escrow_transactions (
			id, session_id, wallet_id, amount, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New().String(), sessionID, walletID, escrowAmount, "locked", now)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create escrow")
		return
	}

	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"session_id":    sessionID,
		"escrow_amount": escrowAmount,
		"status":        "active",
		"started_at":    now,
	})
}

// ProcessBilling processes minute-based billing
func (h *BillingHandler) ProcessBilling(ctx context.Context, sessionID string) error {
	// Get session details
	var userID, providerID string
	var hourlyRate, escrowAmount, totalCost float64
	var startedAt time.Time

	err := h.db.QueryRowContext(ctx, `
		SELECT user_id, provider_id, hourly_rate, escrow_amount, 
		       total_cost, started_at
		FROM rental_sessions
		WHERE id = $1 AND status = 'active'
	`, sessionID).Scan(&userID, &providerID, &hourlyRate, &escrowAmount, &totalCost, &startedAt)

	if err != nil {
		return err
	}

	// Calculate minutes elapsed
	minutesElapsed := int(time.Since(startedAt).Minutes())
	minuteRate := hourlyRate / 60
	newCost := minuteRate * float64(minutesElapsed)
	costIncrement := newCost - totalCost

	// Check if escrow has enough
	if newCost > escrowAmount {
		// End session due to insufficient funds
		return h.EndRental(ctx, sessionID, "insufficient_funds")
	}

	// Update total cost
	_, err = h.db.ExecContext(ctx, `
		UPDATE rental_sessions
		SET total_cost = $1, last_billed_at = $2
		WHERE id = $3
	`, newCost, time.Now(), sessionID)

	if err != nil {
		return err
	}

	// Record usage
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO usage_records (
			id, session_id, minutes_used, cost, created_at
		)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New().String(), sessionID, 1, costIncrement, time.Now())

	return err
}

// EndRental ends a rental session and processes payment
func (h *BillingHandler) EndRental(ctx context.Context, sessionID, reason string) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get session details
	var userID, providerID, walletID, providerWalletAddr string
	var escrowAmount, totalCost float64

	err = tx.QueryRowContext(ctx, `
		SELECT rs.user_id, rs.provider_id, rs.escrow_amount, rs.total_cost,
		       w.id, pw.address
		FROM rental_sessions rs
		JOIN wallets w ON w.user_id = rs.user_id
		JOIN providers p ON p.id = rs.provider_id
		JOIN wallets pw ON pw.user_id = p.user_id
		WHERE rs.id = $1
	`, sessionID).Scan(&userID, &providerID, &escrowAmount, &totalCost, &walletID, &providerWalletAddr)

	if err != nil {
		return err
	}

	// Calculate amounts
	platformFeeAmount := totalCost * h.platformFee
	providerAmount := totalCost - platformFeeAmount
	refundAmount := escrowAmount - totalCost

	// Update session
	now := time.Now()
	_, err = tx.ExecContext(ctx, `
		UPDATE rental_sessions
		SET status = $1, ended_at = $2, end_reason = $3
		WHERE id = $4
	`, "completed", now, reason, sessionID)
	if err != nil {
		return err
	}

	// Release locked funds
	_, err = tx.ExecContext(ctx, `
		UPDATE wallets
		SET locked_balance = locked_balance - $1,
		    balance = balance - $2,
		    updated_at = $3
		WHERE id = $4
	`, escrowAmount, totalCost, now, walletID)
	if err != nil {
		return err
	}

	// Record platform fee
	_, err = tx.ExecContext(ctx, `
		INSERT INTO platform_fees (
			id, session_id, amount, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New().String(), sessionID, platformFeeAmount, "collected", now)
	if err != nil {
		return err
	}

	// Create provider payout
	_, err = tx.ExecContext(ctx, `
		INSERT INTO provider_payouts (
			id, provider_id, session_id, amount, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New().String(), providerID, sessionID, providerAmount, "pending", now)
	if err != nil {
		return err
	}

	// Update escrow status
	_, err = tx.ExecContext(ctx, `
		UPDATE escrow_transactions
		SET status = $1, released_at = $2
		WHERE session_id = $3
	`, "released", now, sessionID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Process blockchain payout (async)
	go h.processProviderPayout(context.Background(), providerID, providerWalletAddr, providerAmount)

	return nil
}

// processProviderPayout sends payment to provider
func (h *BillingHandler) processProviderPayout(ctx context.Context, providerID, address string, amount float64) {
	// Parse provider address
	providerAddr, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return
	}

	// Convert to lamports
	lamports := uint64(amount * 1e9)

	// Transfer from platform wallet
	result, err := h.solanaClient.ReleaseEscrow(ctx, providerAddr, lamports, 0)
	if err != nil {
		// Log error and mark payout as failed
		h.db.ExecContext(ctx, `
			UPDATE provider_payouts
			SET status = $1, error = $2
			WHERE provider_id = $3 AND status = 'pending'
		`, "failed", err.Error(), providerID)
		return
	}

	// Mark payout as completed
	h.db.ExecContext(ctx, `
		UPDATE provider_payouts
		SET status = $1, signature = $2, paid_at = $3
		WHERE provider_id = $4 AND status = 'pending'
	`, "completed", result.Signature, time.Now(), providerID)
}

// GetBillingHistory gets billing history for user
func (h *BillingHandler) GetBillingHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	rows, err := h.db.QueryContext(ctx, `
		SELECT rs.id, rs.job_id, rs.hourly_rate, rs.total_cost,
		       rs.status, rs.started_at, rs.ended_at,
		       p.name as provider_name
		FROM rental_sessions rs
		JOIN providers p ON p.id = rs.provider_id
		WHERE rs.user_id = $1
		ORDER BY rs.created_at DESC
		LIMIT 100
	`, userID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get history")
		return
	}
	defer rows.Close()

	var sessions []map[string]interface{}
	for rows.Next() {
		var s struct {
			ID           string
			JobID        string
			HourlyRate   float64
			TotalCost    float64
			Status       string
			StartedAt    time.Time
			EndedAt      sql.NullTime
			ProviderName string
		}
		rows.Scan(&s.ID, &s.JobID, &s.HourlyRate, &s.TotalCost, &s.Status, &s.StartedAt, &s.EndedAt, &s.ProviderName)

		session := map[string]interface{}{
			"id":            s.ID,
			"job_id":        s.JobID,
			"hourly_rate":   s.HourlyRate,
			"total_cost":    s.TotalCost,
			"status":        s.Status,
			"started_at":    s.StartedAt,
			"provider_name": s.ProviderName,
		}
		if s.EndedAt.Valid {
			session["ended_at"] = s.EndedAt.Time
			session["duration_minutes"] = int(s.EndedAt.Time.Sub(s.StartedAt).Minutes())
		}
		sessions = append(sessions, session)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

