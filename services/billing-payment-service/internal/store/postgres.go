package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/models"
)

// PostgresStore implements billing data storage using PostgreSQL
type PostgresStore struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(db *pgxpool.Pool, logger *zap.Logger) *PostgresStore {
	return &PostgresStore{
		db:     db,
		logger: logger,
	}
}

// Initialize creates the necessary database tables
func (s *PostgresStore) Initialize(ctx context.Context) error {
	queries := []string{
		createWalletsTable,
		createTransactionsTable,
		createRentalSessionsTable,
		createUsageRecordsTable,
		createBillingRecordsTable,
		createProviderRatesTable,
		createIndexes,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	s.logger.Info("Database tables initialized successfully")
	return nil
}

// Wallet operations

// CreateWallet creates a new wallet
func (s *PostgresStore) CreateWallet(ctx context.Context, req *models.WalletCreateRequest) (*models.Wallet, error) {
	wallet := &models.Wallet{
		ID:             uuid.New(),
		UserID:         req.UserID,
		WalletType:     req.WalletType,
		SolanaAddress:  req.SolanaAddress,
		Balance:        decimal.Zero,
		LockedBalance:  decimal.Zero,
		PendingBalance: decimal.Zero,
		IsActive:       true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	query := `
		INSERT INTO wallets (id, user_id, wallet_type, solana_address, balance, locked_balance, pending_balance, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.Exec(ctx, query,
		wallet.ID, wallet.UserID, wallet.WalletType, wallet.SolanaAddress,
		wallet.Balance, wallet.LockedBalance, wallet.PendingBalance,
		wallet.IsActive, wallet.CreatedAt, wallet.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	s.logger.Info("Wallet created", zap.String("wallet_id", wallet.ID.String()), zap.String("user_id", wallet.UserID))
	return wallet, nil
}

// GetWallet retrieves a wallet by ID
func (s *PostgresStore) GetWallet(ctx context.Context, walletID uuid.UUID) (*models.Wallet, error) {
	wallet := &models.Wallet{}
	query := `
		SELECT id, user_id, wallet_type, solana_address, balance, locked_balance, pending_balance,
		       is_active, created_at, updated_at, last_activity_at
		FROM wallets WHERE id = $1
	`

	var lastActivityAt sql.NullTime
	err := s.db.QueryRow(ctx, query, walletID).Scan(
		&wallet.ID, &wallet.UserID, &wallet.WalletType, &wallet.SolanaAddress,
		&wallet.Balance, &wallet.LockedBalance, &wallet.PendingBalance,
		&wallet.IsActive, &wallet.CreatedAt, &wallet.UpdatedAt, &lastActivityAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	if lastActivityAt.Valid {
		wallet.LastActivityAt = &lastActivityAt.Time
	}

	return wallet, nil
}

// GetWalletByUserID retrieves a wallet by user ID and type
func (s *PostgresStore) GetWalletByUserID(ctx context.Context, userID string, walletType models.WalletType) (*models.Wallet, error) {
	wallet := &models.Wallet{}
	query := `
		SELECT id, user_id, wallet_type, solana_address, balance, locked_balance, pending_balance,
		       is_active, created_at, updated_at, last_activity_at
		FROM wallets WHERE user_id = $1 AND wallet_type = $2
	`

	var lastActivityAt sql.NullTime
	err := s.db.QueryRow(ctx, query, userID, walletType).Scan(
		&wallet.ID, &wallet.UserID, &wallet.WalletType, &wallet.SolanaAddress,
		&wallet.Balance, &wallet.LockedBalance, &wallet.PendingBalance,
		&wallet.IsActive, &wallet.CreatedAt, &wallet.UpdatedAt, &lastActivityAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	if lastActivityAt.Valid {
		wallet.LastActivityAt = &lastActivityAt.Time
	}

	return wallet, nil
}

// UpdateWalletBalance updates wallet balance and locked balance
func (s *PostgresStore) UpdateWalletBalance(ctx context.Context, walletID uuid.UUID, balance, lockedBalance decimal.Decimal) error {
	query := `
		UPDATE wallets
		SET balance = $2, locked_balance = $3, updated_at = $4, last_activity_at = $4
		WHERE id = $1
	`

	result, err := s.db.Exec(ctx, query, walletID, balance, lockedBalance, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	if result.RowsAffected() == 0 {
		return models.ErrWalletNotFound
	}

	return nil
}

// Transaction operations

// CreateTransaction creates a new transaction
func (s *PostgresStore) CreateTransaction(ctx context.Context, req *models.TransactionCreateRequest) (*models.Transaction, error) {
	transaction := &models.Transaction{
		ID:           uuid.New(),
		FromWalletID: req.FromWalletID,
		ToWalletID:   req.ToWalletID,
		Type:         req.Type,
		Status:       models.TransactionStatusPending,
		Amount:       req.Amount,
		Fee:          decimal.Zero, // Will be calculated based on transaction type
		Description:  req.Description,
		SessionID:    req.SessionID,
		JobID:        req.JobID,
		Metadata:     req.Metadata,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	metadataJSON, err := json.Marshal(transaction.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO transactions (id, from_wallet_id, to_wallet_id, type, status, amount, fee, description,
		                         session_id, job_id, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = s.db.Exec(ctx, query,
		transaction.ID, transaction.FromWalletID, transaction.ToWalletID,
		transaction.Type, transaction.Status, transaction.Amount, transaction.Fee,
		transaction.Description, transaction.SessionID, transaction.JobID,
		metadataJSON, transaction.CreatedAt, transaction.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	s.logger.Info("Transaction created",
		zap.String("transaction_id", transaction.ID.String()),
		zap.String("type", string(transaction.Type)),
		zap.String("amount", transaction.Amount.String()),
	)

	return transaction, nil
}

// UpdateTransactionStatus updates transaction status and signature
func (s *PostgresStore) UpdateTransactionStatus(ctx context.Context, transactionID uuid.UUID, status models.TransactionStatus, signature *string) error {
	var confirmedAt *time.Time
	if status == models.TransactionStatusConfirmed {
		now := time.Now().UTC()
		confirmedAt = &now
	}

	query := `
		UPDATE transactions
		SET status = $2, solana_signature = $3, confirmed_at = $4, updated_at = $5
		WHERE id = $1
	`

	result, err := s.db.Exec(ctx, query, transactionID, status, signature, confirmedAt, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return models.ErrTransactionNotFound
	}

	return nil
}

// GetTransaction retrieves a transaction by ID
func (s *PostgresStore) GetTransaction(ctx context.Context, transactionID uuid.UUID) (*models.Transaction, error) {
	transaction := &models.Transaction{}
	query := `
		SELECT id, from_wallet_id, to_wallet_id, type, status, amount, fee, description,
		       solana_signature, session_id, job_id, metadata, created_at, updated_at, confirmed_at
		FROM transactions WHERE id = $1
	`

	var metadataJSON []byte
	var confirmedAt sql.NullTime
	err := s.db.QueryRow(ctx, query, transactionID).Scan(
		&transaction.ID, &transaction.FromWalletID, &transaction.ToWalletID,
		&transaction.Type, &transaction.Status, &transaction.Amount, &transaction.Fee,
		&transaction.Description, &transaction.SolanaSignature, &transaction.SessionID,
		&transaction.JobID, &metadataJSON, &transaction.CreatedAt, &transaction.UpdatedAt,
		&confirmedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrTransactionNotFound
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if confirmedAt.Valid {
		transaction.ConfirmedAt = &confirmedAt.Time
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &transaction.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return transaction, nil
}

// Rental Session operations

// CreateRentalSession creates a new rental session
func (s *PostgresStore) CreateRentalSession(ctx context.Context, session *models.RentalSession) error {
	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO rental_sessions (
			id, user_id, provider_id, job_id, status, gpu_model, allocated_vram_mb, total_vram_mb,
			vram_percentage, hourly_rate, vram_rate, power_rate, platform_fee_rate, estimated_power_w,
			actual_power_w, started_at, ended_at, last_billed_at, total_cost, platform_fee,
			provider_earnings, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
	`

	_, err = s.db.Exec(ctx, query,
		session.ID, session.UserID, session.ProviderID, session.JobID, session.Status,
		session.GPUModel, session.AllocatedVRAM, session.TotalVRAM, session.VRAMPercentage,
		session.HourlyRate, session.VRAMRate, session.PowerRate, session.PlatformFeeRate,
		session.EstimatedPowerW, session.ActualPowerW, session.StartedAt, session.EndedAt,
		session.LastBilledAt, session.TotalCost, session.PlatformFee, session.ProviderEarnings,
		metadataJSON, session.CreatedAt, session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create rental session: %w", err)
	}

	s.logger.Info("Rental session created", zap.String("session_id", session.ID.String()))
	return nil
}

// GetRentalSession retrieves a rental session by ID
func (s *PostgresStore) GetRentalSession(ctx context.Context, sessionID uuid.UUID) (*models.RentalSession, error) {
	session := &models.RentalSession{}
	query := `
		SELECT id, user_id, provider_id, job_id, status, gpu_model, allocated_vram_mb, total_vram_mb,
		       vram_percentage, hourly_rate, vram_rate, power_rate, platform_fee_rate, estimated_power_w,
		       actual_power_w, started_at, ended_at, last_billed_at, total_cost, platform_fee,
		       provider_earnings, metadata, created_at, updated_at
		FROM rental_sessions WHERE id = $1
	`

	var metadataJSON []byte
	var endedAt sql.NullTime
	var actualPowerW sql.NullInt32
	err := s.db.QueryRow(ctx, query, sessionID).Scan(
		&session.ID, &session.UserID, &session.ProviderID, &session.JobID, &session.Status,
		&session.GPUModel, &session.AllocatedVRAM, &session.TotalVRAM, &session.VRAMPercentage,
		&session.HourlyRate, &session.VRAMRate, &session.PowerRate, &session.PlatformFeeRate,
		&session.EstimatedPowerW, &actualPowerW, &session.StartedAt, &endedAt,
		&session.LastBilledAt, &session.TotalCost, &session.PlatformFee, &session.ProviderEarnings,
		&metadataJSON, &session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get rental session: %w", err)
	}

	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	if actualPowerW.Valid {
		actualPower := uint32(actualPowerW.Int32)
		session.ActualPowerW = &actualPower
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return session, nil
}

// UpdateRentalSession updates a rental session
func (s *PostgresStore) UpdateRentalSession(ctx context.Context, session *models.RentalSession) error {
	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE rental_sessions SET
			status = $2, actual_power_w = $3, ended_at = $4, last_billed_at = $5,
			total_cost = $6, platform_fee = $7, provider_earnings = $8, metadata = $9, updated_at = $10
		WHERE id = $1
	`

	result, err := s.db.Exec(ctx, query,
		session.ID, session.Status, session.ActualPowerW, session.EndedAt, session.LastBilledAt,
		session.TotalCost, session.PlatformFee, session.ProviderEarnings, metadataJSON, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to update rental session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return models.ErrSessionNotFound
	}

	return nil
}

// GetActiveSessionsByUser retrieves active sessions for a user
func (s *PostgresStore) GetActiveSessionsByUser(ctx context.Context, userID string) ([]models.RentalSession, error) {
	query := `
		SELECT id, user_id, provider_id, job_id, status, gpu_model, allocated_vram_mb, total_vram_mb,
		       vram_percentage, hourly_rate, vram_rate, power_rate, platform_fee_rate, estimated_power_w,
		       actual_power_w, started_at, ended_at, last_billed_at, total_cost, platform_fee,
		       provider_earnings, metadata, created_at, updated_at
		FROM rental_sessions
		WHERE user_id = $1 AND status = 'active'
		ORDER BY started_at DESC
	`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.RentalSession
	for rows.Next() {
		var session models.RentalSession
		var metadataJSON []byte
		var endedAt sql.NullTime
		var actualPowerW sql.NullInt32

		err := rows.Scan(
			&session.ID, &session.UserID, &session.ProviderID, &session.JobID, &session.Status,
			&session.GPUModel, &session.AllocatedVRAM, &session.TotalVRAM, &session.VRAMPercentage,
			&session.HourlyRate, &session.VRAMRate, &session.PowerRate, &session.PlatformFeeRate,
			&session.EstimatedPowerW, &actualPowerW, &session.StartedAt, &endedAt,
			&session.LastBilledAt, &session.TotalCost, &session.PlatformFee, &session.ProviderEarnings,
			&metadataJSON, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if endedAt.Valid {
			session.EndedAt = &endedAt.Time
		}
		if actualPowerW.Valid {
			actualPower := uint32(actualPowerW.Int32)
			session.ActualPowerW = &actualPower
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
				s.logger.Warn("Failed to unmarshal session metadata", zap.Error(err))
			}
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Usage Record operations

// CreateUsageRecord creates a new usage record
func (s *PostgresStore) CreateUsageRecord(ctx context.Context, record *models.UsageRecord) error {
	query := `
		INSERT INTO usage_records (
			id, session_id, recorded_at, gpu_utilization_percent, vram_utilization_percent,
			power_draw_w, temperature_c, period_minutes, period_cost, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.Exec(ctx, query,
		record.ID, record.SessionID, record.RecordedAt, record.GPUUtilization,
		record.VRAMUtilization, record.PowerDraw, record.Temperature,
		record.PeriodMinutes, record.PeriodCost, record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create usage record: %w", err)
	}

	return nil
}

// GetUsageRecordsBySession retrieves usage records for a session
func (s *PostgresStore) GetUsageRecordsBySession(ctx context.Context, sessionID uuid.UUID, limit int) ([]models.UsageRecord, error) {
	query := `
		SELECT id, session_id, recorded_at, gpu_utilization_percent, vram_utilization_percent,
		       power_draw_w, temperature_c, period_minutes, period_cost, created_at
		FROM usage_records
		WHERE session_id = $1
		ORDER BY recorded_at DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage records: %w", err)
	}
	defer rows.Close()

	var records []models.UsageRecord
	for rows.Next() {
		var record models.UsageRecord
		err := rows.Scan(
			&record.ID, &record.SessionID, &record.RecordedAt, &record.GPUUtilization,
			&record.VRAMUtilization, &record.PowerDraw, &record.Temperature,
			&record.PeriodMinutes, &record.PeriodCost, &record.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage record: %w", err)
		}
		records = append(records, record)
	}

	return records, nil
}

// Billing Record operations

// CreateBillingRecord creates a new billing record
func (s *PostgresStore) CreateBillingRecord(ctx context.Context, record *models.BillingRecord) error {
	query := `
		INSERT INTO billing_records (
			id, user_id, provider_id, session_id, billing_period_start, billing_period_end,
			total_minutes, avg_gpu_utilization, avg_vram_utilization, avg_power_draw,
			base_cost, vram_cost, power_cost, total_cost, platform_fee, provider_earnings,
			transaction_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err := s.db.Exec(ctx, query,
		record.ID, record.UserID, record.ProviderID, record.SessionID,
		record.BillingPeriodStart, record.BillingPeriodEnd, record.TotalMinutes,
		record.AvgGPUUtil, record.AvgVRAMUtil, record.AvgPowerDraw,
		record.BaseCost, record.VRAMCost, record.PowerCost, record.TotalCost,
		record.PlatformFee, record.ProviderEarnings, record.TransactionID, record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create billing record: %w", err)
	}

	return nil
}

// GetBillingHistory retrieves billing history with pagination
func (s *PostgresStore) GetBillingHistory(ctx context.Context, req *models.BillingHistoryRequest) (*models.BillingHistoryResponse, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if req.UserID != nil {
		whereClause += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *req.UserID)
		argIndex++
	}

	if req.ProviderID != nil {
		whereClause += fmt.Sprintf(" AND provider_id = $%d", argIndex)
		args = append(args, *req.ProviderID)
		argIndex++
	}

	if req.StartDate != nil {
		whereClause += fmt.Sprintf(" AND billing_period_start >= $%d", argIndex)
		args = append(args, *req.StartDate)
		argIndex++
	}

	if req.EndDate != nil {
		whereClause += fmt.Sprintf(" AND billing_period_end <= $%d", argIndex)
		args = append(args, *req.EndDate)
		argIndex++
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM billing_records %s", whereClause)
	var total int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count billing records: %w", err)
	}

	// Get records with pagination
	query := fmt.Sprintf(`
		SELECT id, user_id, provider_id, session_id, billing_period_start, billing_period_end,
		       total_minutes, avg_gpu_utilization, avg_vram_utilization, avg_power_draw,
		       base_cost, vram_cost, power_cost, total_cost, platform_fee, provider_earnings,
		       transaction_id, created_at
		FROM billing_records %s
		ORDER BY billing_period_start DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, req.Limit, req.Offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query billing records: %w", err)
	}
	defer rows.Close()

	var records []models.BillingRecord
	for rows.Next() {
		var record models.BillingRecord
		err := rows.Scan(
			&record.ID, &record.UserID, &record.ProviderID, &record.SessionID,
			&record.BillingPeriodStart, &record.BillingPeriodEnd, &record.TotalMinutes,
			&record.AvgGPUUtil, &record.AvgVRAMUtil, &record.AvgPowerDraw,
			&record.BaseCost, &record.VRAMCost, &record.PowerCost, &record.TotalCost,
			&record.PlatformFee, &record.ProviderEarnings, &record.TransactionID, &record.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan billing record: %w", err)
		}
		records = append(records, record)
	}

	return &models.BillingHistoryResponse{
		Records: records,
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

// GetProviderEarnings calculates provider earnings for a given period
func (s *PostgresStore) GetProviderEarnings(ctx context.Context, req *models.ProviderEarningsRequest) (*models.ProviderEarningsResponse, error) {
	whereClause := "WHERE provider_id = $1"
	args := []interface{}{req.ProviderID}
	argIndex := 2

	if req.StartDate != nil {
		whereClause += fmt.Sprintf(" AND billing_period_start >= $%d", argIndex)
		args = append(args, *req.StartDate)
		argIndex++
	}

	if req.EndDate != nil {
		whereClause += fmt.Sprintf(" AND billing_period_end <= $%d", argIndex)
		args = append(args, *req.EndDate)
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(provider_earnings), 0) as total_earnings,
			COALESCE(SUM(CASE WHEN t.status = 'confirmed' THEN provider_earnings ELSE 0 END), 0) as paid_earnings,
			COALESCE(SUM(CASE WHEN t.status = 'pending' THEN provider_earnings ELSE 0 END), 0) as pending_earnings,
			COUNT(*) as total_sessions,
			COALESCE(SUM(total_minutes), 0) as total_minutes,
			COALESCE(AVG(total_cost / NULLIF(total_minutes, 0) * 60), 0) as avg_hourly_rate
		FROM billing_records br
		LEFT JOIN transactions t ON br.transaction_id = t.id
		%s
	`, whereClause)

	var totalEarnings, paidEarnings, pendingEarnings, avgHourlyRate decimal.Decimal
	var totalSessions, totalMinutes int

	err := s.db.QueryRow(ctx, query, args...).Scan(
		&totalEarnings, &paidEarnings, &pendingEarnings,
		&totalSessions, &totalMinutes, &avgHourlyRate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate provider earnings: %w", err)
	}

	totalHours := decimal.NewFromInt(int64(totalMinutes)).Div(decimal.NewFromInt(60))

	period := "all_time"
	if req.StartDate != nil && req.EndDate != nil {
		period = fmt.Sprintf("%s to %s", req.StartDate.Format("2006-01-02"), req.EndDate.Format("2006-01-02"))
	}

	return &models.ProviderEarningsResponse{
		ProviderID:      req.ProviderID,
		TotalEarnings:   totalEarnings,
		PendingEarnings: pendingEarnings,
		PaidEarnings:    paidEarnings,
		TotalSessions:   totalSessions,
		TotalHours:      totalHours,
		AvgHourlyRate:   avgHourlyRate,
		Period:          period,
	}, nil
}
