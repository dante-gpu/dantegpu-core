// Package covenant is the DanteGPU client for the Covenant settlement protocol
// (https://www.covenant.run). A GPU rental settles on-chain as a Covenant job:
// the renter (poster) locks USDC in a per-job PDA escrow, the provider (taker)
// delivers, and the escrow releases after the challenge window. See
// contracts/COVENANT_INTEGRATION.md for the full mapping.
//
// Important: Covenant state changes (create / accept / submit / finalize /
// dispute) are on-chain instructions that must be SIGNED. Covenant's HTTP API
// either (a) verifies and indexes a client-signed transaction, or (b) signs with
// one of Covenant's own registered bot wallets. For DanteGPU's custodial model
// the platform keypair signs the Covenant program instruction on-chain (built
// with gagliardetto/solana-go against the Covenant program), then this client
// posts the resulting tx signature to the HTTP API for indexing. The on-chain
// instruction building is wired in M3.3; this package provides the HTTP layer
// (reads + index-after-sign) and the shared types.
package covenant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// Config configures the Covenant client.
type Config struct {
	// APIBaseURL is the Covenant HTTP API root, e.g. https://www.covenant.run
	APIBaseURL string `yaml:"api_base_url"`
	// ProgramID is the Covenant Anchor program id on Solana (used by the on-chain
	// signer in M3.3).
	ProgramID string `yaml:"program_id"`
	// USDCMint is the settlement token mint (must match the billing token mint).
	USDCMint string        `yaml:"usdc_mint"`
	Timeout  time.Duration `yaml:"timeout"`
}

// Client talks to the Covenant HTTP API.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient creates a Covenant HTTP client.
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Client{cfg: cfg, httpClient: &http.Client{Timeout: timeout}}
}

// JobStatus mirrors the Covenant on-chain JobEscrow status.
type JobStatus string

const (
	StatusOpen      JobStatus = "Open"
	StatusAccepted  JobStatus = "Accepted"
	StatusDelivered JobStatus = "Delivered"
	StatusDisputed  JobStatus = "Disputed"
	StatusSettled   JobStatus = "Settled"
	StatusCancelled JobStatus = "Cancelled"
)

// Job mirrors a Covenant job/escrow (the fields DanteGPU needs).
type Job struct {
	ID           string          `json:"id"`
	PosterWallet string          `json:"posterWallet"`
	TakerWallet  string          `json:"takerWallet"`
	TokenMint    string          `json:"tokenMint"`
	Amount       decimal.Decimal `json:"amount"`
	SpecHash     string          `json:"specHash"`
	Status       JobStatus       `json:"status"`
	Deadline     time.Time       `json:"deadline"`
	ChallengeEnd *time.Time      `json:"challengeEnd,omitempty"`
	WorkHash     string          `json:"workHash,omitempty"`
	DeliveryURI  string          `json:"deliveryUri,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
}

// CreateJobRequest is the body for POST /api/jobs. The platform must have already
// signed the on-chain create_job instruction; TxSignature is that transaction so
// Covenant can verify and index it. Amount is a JSON number and Deadline/CreatedAt
// are exact ISO strings (millisecond precision, Z) so Covenant re-derives the same
// spec hash and job PDA.
type CreateJobRequest struct {
	PosterWallet           string  `json:"posterWallet"`
	Amount                 float64 `json:"amount"`
	Category               string  `json:"category,omitempty"`
	Title                  string  `json:"title,omitempty"`
	Description            string  `json:"description,omitempty"`
	Deadline               string  `json:"deadline"`
	ChallengePeriodSeconds int64   `json:"challengePeriodSeconds"`
	TxSignature            string  `json:"txSignature,omitempty"`
	CreatedAt              string  `json:"createdAt"`
}

// SubmitWorkRequest is the body for POST /api/jobs/{id}/submit. WorkHash is the
// SHA-256 of the signed metering receipt; DeliveryURI points to the stored
// receipt. TxSignature is the on-chain submit_work transaction.
type SubmitWorkRequest struct {
	WorkHash    string `json:"workHash"`
	DeliveryURI string `json:"deliveryUri"`
	TxSignature string `json:"txSignature,omitempty"`
}

func (c *Client) url(path string) string { return c.cfg.APIBaseURL + path }

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("covenant: marshal %s: %w", path, err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.url(path), reader)
	if err != nil {
		return fmt.Errorf("covenant: build request %s: %w", path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("covenant: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("covenant: %s %s returned %d: %s", method, path, resp.StatusCode, string(msg))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("covenant: decode %s: %w", path, err)
		}
	}
	return nil
}

// GetJob fetches a job by id (GET /api/jobs/{id}). Pure read.
func (c *Client) GetJob(ctx context.Context, jobID string) (*Job, error) {
	var job Job
	if err := c.doJSON(ctx, http.MethodGet, "/api/jobs/"+jobID, nil, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// ListJobs returns jobs filtered by taker wallet (GET /api/jobs). Used by the
// settlement crank to find delivered jobs awaiting finalize.
func (c *Client) ListJobs(ctx context.Context, taker string) ([]Job, error) {
	path := "/api/jobs?limit=100"
	if taker != "" {
		path += "&taker=" + taker
	}
	var out struct {
		Jobs []Job `json:"jobs"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Jobs, nil
}

// GetBalance returns a wallet's USDC balance as Covenant sees it
// (GET /api/balance/{wallet}). Pure read.
func (c *Client) GetBalance(ctx context.Context, wallet string) (decimal.Decimal, error) {
	var out struct {
		Balance decimal.Decimal `json:"balance"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/balance/"+wallet, nil, &out); err != nil {
		return decimal.Zero, err
	}
	return out.Balance, nil
}

// IndexCreatedJob registers a job with Covenant after the platform has signed and
// submitted the on-chain create_job instruction (POST /api/jobs). The on-chain
// signing is the M3.3 companion; this records and verifies it.
func (c *Client) IndexCreatedJob(ctx context.Context, req *CreateJobRequest) (*Job, error) {
	var job Job
	if err := c.doJSON(ctx, http.MethodPost, "/api/jobs", req, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// SubmitWork records the provider's delivery (POST /api/jobs/{id}/submit) after
// the on-chain submit_work instruction is signed.
func (c *Client) SubmitWork(ctx context.Context, jobID string, req *SubmitWorkRequest) (*Job, error) {
	var job Job
	if err := c.doJSON(ctx, http.MethodPost, "/api/jobs/"+jobID+"/submit", req, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// Finalize triggers settlement after the challenge window (POST
// /api/jobs/{id}/finalize). This is the provider payout. Covenant also runs a
// finalize cron (/api/cron/finalize); DanteGPU's settlement crank calls this for
// delivered, undisputed jobs.
func (c *Client) Finalize(ctx context.Context, jobID string) (*Job, error) {
	var job Job
	if err := c.doJSON(ctx, http.MethodPost, "/api/jobs/"+jobID+"/finalize", struct{}{}, &job); err != nil {
		return nil, err
	}
	return &job, nil
}
