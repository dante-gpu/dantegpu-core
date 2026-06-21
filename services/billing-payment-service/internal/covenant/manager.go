package covenant

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/shopspring/decimal"
)

// isoMillis matches JavaScript Date.toISOString() (millisecond precision, Z).
// The spec hash depends on this exact format, so it must not change.
const isoMillis = "2006-01-02T15:04:05.000Z"

// Manager orchestrates a GPU rental's settlement: it locks USDC in a Covenant
// escrow on-chain (custodial poster = platform) and indexes it over HTTP, then
// submits the metering receipt and finalizes the payout.
type Manager struct {
	signer           *Signer
	client           *Client
	challengeSeconds int64
}

// NewManager wires the on-chain signer and the HTTP client. challengeSeconds is
// the optimistic challenge window (default 1h, the protocol minimum).
func NewManager(signer *Signer, client *Client, challengeSeconds int64) *Manager {
	if challengeSeconds <= 0 {
		challengeSeconds = 3600
	}
	return &Manager{signer: signer, client: client, challengeSeconds: challengeSeconds}
}

// Enabled reports whether settlement is configured.
func (m *Manager) Enabled() bool { return m != nil && m.signer != nil && m.client != nil }

// NewManagerFromConfig builds the settlement manager from the covenant config,
// the Solana RPC URL, and the platform keypair file. If covenant is not
// configured it returns a disabled (nil-signer) manager so callers can no-op.
func NewManagerFromConfig(cfg Config, rpcURL, privateKeyPath string, challengeSeconds int64) (*Manager, error) {
	if cfg.APIBaseURL == "" || cfg.ProgramID == "" || cfg.USDCMint == "" {
		return &Manager{client: NewClient(cfg), challengeSeconds: challengeSeconds}, nil
	}
	key, err := solana.PrivateKeyFromSolanaKeygenFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("covenant: load platform keypair: %w", err)
	}
	signer, err := NewSigner(rpcURL, key, cfg.ProgramID, cfg.USDCMint)
	if err != nil {
		return nil, err
	}
	return NewManager(signer, NewClient(cfg), challengeSeconds), nil
}

// RentalJob is the opened-job handle DanteGPU persists on the rental session.
type RentalJob struct {
	JobPDA      string
	SpecHashHex string
	Poster      string
	TxSignature string
}

// OpenRentalJob locks `amount` USDC in a Covenant escrow for a fixed-duration
// rental, then indexes it. Phase 1: poster = platform (custodial), amount = the
// fixed block price. Returns the job PDA to store on the session.
func (m *Manager) OpenRentalJob(ctx context.Context, amount decimal.Decimal, gpuModel, sessionID string, duration time.Duration) (*RentalJob, error) {
	if !m.Enabled() {
		return nil, fmt.Errorf("covenant: settlement not configured")
	}
	now := time.Now().UTC()
	createdAtISO := now.Format(isoMillis)
	deadlineTime := now.Add(duration)
	deadlineISO := deadlineTime.Format(isoMillis)
	title := "GPU rental: " + gpuModel
	description := "DanteGPU rental session " + sessionID

	poster := m.signer.PlatformWallet()
	spec := BuildSpec(poster, amount, deadlineISO, createdAtISO, title, description)
	specHash, specJSON, err := SpecHash(spec)
	if err != nil {
		return nil, err
	}

	res, err := m.signer.CreateJob(ctx, amount, specHash, specJSON, deadlineTime.Unix(), uint64(m.challengeSeconds))
	if err != nil {
		return nil, err
	}
	job := &RentalJob{JobPDA: res.JobPDA.String(), SpecHashHex: res.SpecHashHex, Poster: res.PosterWallet, TxSignature: res.TxSignature}

	amt, _ := amount.Float64()
	if _, err := m.client.IndexCreatedJob(ctx, &CreateJobRequest{
		PosterWallet:           res.PosterWallet,
		Amount:                 amt,
		Category:               "compute",
		Title:                  title,
		Description:            description,
		Deadline:               deadlineISO,
		ChallengePeriodSeconds: m.challengeSeconds,
		TxSignature:            res.TxSignature,
		CreatedAt:              createdAtISO,
	}); err != nil {
		// The escrow is already locked on-chain; indexing can be retried. Return
		// the job so the caller persists the PDA, plus the error.
		return job, fmt.Errorf("covenant: job created on-chain but indexing failed: %w", err)
	}
	return job, nil
}

// CloseRentalJob submits the provider's delivery: the metering receipt hash and a
// URI to the stored receipt. After the challenge window, SettleRentalJob pays out.
func (m *Manager) CloseRentalJob(ctx context.Context, jobPDA, poster string, receiptHash [32]byte, receiptURI string) (string, error) {
	if !m.Enabled() {
		return "", fmt.Errorf("covenant: settlement not configured")
	}
	job, err := solana.PublicKeyFromBase58(jobPDA)
	if err != nil {
		return "", fmt.Errorf("covenant: invalid job pda: %w", err)
	}
	posterPub, err := solana.PublicKeyFromBase58(poster)
	if err != nil {
		return "", fmt.Errorf("covenant: invalid poster: %w", err)
	}
	sig, err := m.signer.SubmitWork(ctx, job, posterPub, receiptHash, receiptURI)
	if err != nil {
		return "", err
	}
	_, _ = m.client.SubmitWork(ctx, jobPDA, &SubmitWorkRequest{
		WorkHash:    fmt.Sprintf("%x", receiptHash[:]),
		DeliveryURI: receiptURI,
		TxSignature: sig,
	})
	return sig, nil
}

// SettleRentalJob finalizes the escrow to the provider after the challenge window
// (the provider payout, on-chain).
func (m *Manager) SettleRentalJob(ctx context.Context, jobPDA, poster, taker string) (string, error) {
	if !m.Enabled() {
		return "", fmt.Errorf("covenant: settlement not configured")
	}
	job, err := solana.PublicKeyFromBase58(jobPDA)
	if err != nil {
		return "", fmt.Errorf("covenant: invalid job pda: %w", err)
	}
	posterPub, err := solana.PublicKeyFromBase58(poster)
	if err != nil {
		return "", fmt.Errorf("covenant: invalid poster: %w", err)
	}
	takerPub, err := solana.PublicKeyFromBase58(taker)
	if err != nil {
		return "", fmt.Errorf("covenant: invalid taker: %w", err)
	}
	return m.signer.Finalize(ctx, job, posterPub, takerPub)
}

// DisputeRentalJob opens a dispute on a rental's escrow with the given reason,
// posting the platform's bond. The bonded 2-of-3 arbitrator resolves it using the
// metering receipt as evidence (FavorPoster refunds, Split pays for actual usage).
func (m *Manager) DisputeRentalJob(ctx context.Context, jobPDA, poster, reason string, amount decimal.Decimal) (string, error) {
	if !m.Enabled() {
		return "", fmt.Errorf("covenant: settlement not configured")
	}
	job, err := solana.PublicKeyFromBase58(jobPDA)
	if err != nil {
		return "", fmt.Errorf("covenant: invalid job pda: %w", err)
	}
	posterPub, err := solana.PublicKeyFromBase58(poster)
	if err != nil {
		return "", fmt.Errorf("covenant: invalid poster: %w", err)
	}
	reasonHash := sha256.Sum256([]byte(reason))
	return m.signer.RaiseDispute(ctx, job, posterPub, reasonHash, toAtomic(amount))
}

// SettleRipeJobs is the provider-payout crank. It finalizes every delivered,
// undisputed Covenant job (taker == platform in Phase 1) whose challenge window
// has elapsed, releasing the escrow on-chain. Returns the number settled.
// Already-finalized or transient errors are tolerated and retried next tick.
func (m *Manager) SettleRipeJobs(ctx context.Context) (int, error) {
	if !m.Enabled() {
		return 0, nil
	}
	platform := m.signer.PlatformWallet().String()
	jobs, err := m.client.ListJobs(ctx, platform)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	settled := 0
	for _, j := range jobs {
		if j.ChallengeEnd == nil || !now.After(*j.ChallengeEnd) {
			continue // window not elapsed yet
		}
		if j.WorkHash == "" {
			continue // not delivered
		}
		switch strings.ToLower(string(j.Status)) {
		case "settled", "cancelled", "disputed":
			continue
		}
		if _, ferr := m.SettleRentalJob(ctx, j.ID, platform, platform); ferr != nil {
			continue
		}
		settled++
	}
	return settled, nil
}
