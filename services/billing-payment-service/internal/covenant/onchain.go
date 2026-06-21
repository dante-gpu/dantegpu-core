package covenant

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/shopspring/decimal"
)

// newCompactEncoder returns a JSON encoder that does not HTML-escape, matching
// JavaScript JSON.stringify (used for the spec hash).
func newCompactEncoder(w io.Writer) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc
}

// Anchor instruction discriminators (from the Covenant IDL).
var (
	discCreateJob       = []byte{178, 130, 217, 110, 100, 27, 82, 119}
	discAcceptJob       = []byte{43, 201, 124, 1, 19, 189, 96, 10}
	discSubmitWork      = []byte{158, 80, 101, 51, 114, 130, 101, 253}
	discFinalizePayment = []byte{254, 254, 46, 40, 22, 126, 221, 128}
	discRaiseDispute    = []byte{41, 243, 1, 51, 150, 95, 246, 73}
)

// PDA seed prefixes used by the Covenant program.
var (
	seedConfig      = []byte("config")
	seedJob         = []byte("job")
	seedEscrowToken = []byte("escrow_token")
	seedReputation  = []byte("reputation")
	seedClaim       = []byte("claim")
	seedBond        = []byte("bond")
)

// Dispute bond defaults (from Covenant constants): 10% of escrow, minimum 1 USDC.
const (
	defaultBondBps      = 1000
	defaultMinBondAtomic = 1_000_000
)

// MinBond returns the minimum dispute bond in atomic units for an escrow amount,
// matching the program: max(amount * bps / 10_000, min_absolute).
func MinBond(amountAtomic uint64) uint64 {
	pct := amountAtomic * defaultBondBps / 10_000
	if pct < defaultMinBondAtomic {
		return defaultMinBondAtomic
	}
	return pct
}

// usdcDecimals is the SPL decimals for the settlement token (USDC = 6).
const usdcDecimals = 6

// Signer signs and submits Covenant program instructions on-chain with the
// platform keypair. This is the custodial poster: DanteGPU funds and locks the
// escrow on the renter's behalf, then indexes the job over HTTP.
type Signer struct {
	rpc       *rpc.Client
	key       solana.PrivateKey
	programID solana.PublicKey
	usdcMint  solana.PublicKey
}

// NewSigner builds an on-chain signer. rpcURL is the Solana RPC, key is the
// platform keypair, programID and usdcMint come from config.
func NewSigner(rpcURL string, key solana.PrivateKey, programID, usdcMint string) (*Signer, error) {
	pid, err := solana.PublicKeyFromBase58(programID)
	if err != nil {
		return nil, fmt.Errorf("covenant: invalid program id: %w", err)
	}
	mint, err := solana.PublicKeyFromBase58(usdcMint)
	if err != nil {
		return nil, fmt.Errorf("covenant: invalid usdc mint: %w", err)
	}
	return &Signer{rpc: rpc.New(rpcURL), key: key, programID: pid, usdcMint: mint}, nil
}

// PlatformWallet returns the platform public key (the custodial poster).
func (s *Signer) PlatformWallet() solana.PublicKey { return s.key.PublicKey() }

// ----- spec hashing (must match Covenant's buildJobSpec + JSON.stringify) -----

// jobSpec is the exact field set and order Covenant hashes. hashJobSpec uses
// JSON.stringify (insertion order), not canonical JSON, so the field order here
// is load-bearing and must match app/lib/spec.ts buildJobSpec.
type jobSpec struct {
	PosterWallet string  `json:"posterWallet"`
	Amount       float64 `json:"amount"`
	Language     string  `json:"language"`
	Deadline     string  `json:"deadline"`
	CreatedAt    string  `json:"createdAt"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Requirements string  `json:"requirements"`
}

// SpecHash computes the 32-byte spec hash exactly as Covenant does:
// sha256(JSON.stringify(spec)) with no HTML escaping. The same hash must be sent
// to the HTTP API so Covenant re-derives the same job PDA.
func SpecHash(spec *jobSpec) ([32]byte, string, error) {
	// Encode without HTML escaping and without a trailing newline to match
	// JavaScript JSON.stringify.
	var b strings.Builder
	enc := newCompactEncoder(&b)
	if err := enc.Encode(spec); err != nil {
		return [32]byte{}, "", err
	}
	js := strings.TrimRight(b.String(), "\n")
	sum := sha256.Sum256([]byte(js))
	return sum, js, nil
}

// BuildSpec assembles the canonical rental spec.
func BuildSpec(poster solana.PublicKey, amount decimal.Decimal, deadlineISO, createdAtISO, title, description string) *jobSpec {
	amt, _ := amount.Float64()
	return &jobSpec{
		PosterWallet: poster.String(),
		Amount:       amt,
		Language:     "English",
		Deadline:     deadlineISO,
		CreatedAt:    createdAtISO,
		Title:        title,
		Description:  description,
		Requirements: "",
	}
}

// ----- PDA derivations -----

func (s *Signer) configPDA() (solana.PublicKey, error) {
	pda, _, err := solana.FindProgramAddress([][]byte{seedConfig}, s.programID)
	return pda, err
}

func (s *Signer) jobPDA(poster solana.PublicKey, specHash [32]byte) (solana.PublicKey, error) {
	pda, _, err := solana.FindProgramAddress([][]byte{seedJob, poster.Bytes(), specHash[:]}, s.programID)
	return pda, err
}

func (s *Signer) escrowTokenPDA(job solana.PublicKey) (solana.PublicKey, error) {
	pda, _, err := solana.FindProgramAddress([][]byte{seedEscrowToken, job.Bytes()}, s.programID)
	return pda, err
}

func (s *Signer) reputationPDA(taker solana.PublicKey) (solana.PublicKey, error) {
	pda, _, err := solana.FindProgramAddress([][]byte{seedReputation, taker.Bytes()}, s.programID)
	return pda, err
}

func (s *Signer) claimPDA(job solana.PublicKey) (solana.PublicKey, error) {
	pda, _, err := solana.FindProgramAddress([][]byte{seedClaim, job.Bytes()}, s.programID)
	return pda, err
}

func (s *Signer) bondPDA(job solana.PublicKey) (solana.PublicKey, error) {
	pda, _, err := solana.FindProgramAddress([][]byte{seedBond, job.Bytes()}, s.programID)
	return pda, err
}

// ----- Borsh arg encoding -----

func borshU64(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

func borshI64(v int64) []byte { return borshU64(uint64(v)) }

func borshString(s string) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(len(s)))
	return append(b, []byte(s)...)
}

func toAtomic(amount decimal.Decimal) uint64 {
	mult := decimal.New(1, usdcDecimals) // 10^6
	return amount.Mul(mult).BigInt().Uint64()
}

func meta(pub solana.PublicKey, writable, signer bool) *solana.AccountMeta {
	return solana.NewAccountMeta(pub, writable, signer)
}

// ----- on-chain operations -----

// CreateJobResult carries the created job's on-chain identity.
type CreateJobResult struct {
	JobPDA       solana.PublicKey
	SpecHashHex  string
	SpecJSON     string
	TxSignature  string
	PosterWallet string
}

// CreateJob builds, signs, and submits the create_job instruction (locks USDC in
// the per-job escrow), returning the job PDA and tx signature for HTTP indexing.
func (s *Signer) CreateJob(ctx context.Context, amount decimal.Decimal, specHash [32]byte, specJSON string, deadlineUnix int64, challengePeriodSeconds uint64) (*CreateJobResult, error) {
	poster := s.key.PublicKey()

	config, err := s.configPDA()
	if err != nil {
		return nil, err
	}
	job, err := s.jobPDA(poster, specHash)
	if err != nil {
		return nil, err
	}
	escrowToken, err := s.escrowTokenPDA(job)
	if err != nil {
		return nil, err
	}
	posterATA, _, err := solana.FindAssociatedTokenAddress(poster, s.usdcMint)
	if err != nil {
		return nil, err
	}

	data := append([]byte{}, discCreateJob...)
	data = append(data, borshU64(toAtomic(amount))...)
	data = append(data, specHash[:]...)
	data = append(data, borshI64(deadlineUnix)...)
	data = append(data, borshU64(challengePeriodSeconds)...)

	createIx := &solana.GenericInstruction{
		ProgID: s.programID,
		AccountValues: solana.AccountMetaSlice{
			meta(poster, true, true),
			meta(config, false, false),
			meta(job, true, false),
			meta(escrowToken, true, false),
			meta(posterATA, true, false),
			meta(s.usdcMint, false, false),
			meta(solana.TokenProgramID, false, false),
			meta(solana.SystemProgramID, false, false),
			meta(solana.SysVarRentPubkey, false, false),
		},
		DataBytes: data,
	}

	// Phase 1 custodial: the platform is both poster and taker. Accept the job in
	// the same transaction so it is immediately submittable and finalizable.
	acceptData := append([]byte{}, discAcceptJob...)
	acceptData = append(acceptData, specHash[:]...)
	acceptIx := &solana.GenericInstruction{
		ProgID: s.programID,
		AccountValues: solana.AccountMetaSlice{
			meta(poster, true, true), // taker == poster (platform)
			meta(job, true, false),
			meta(poster, false, false),
		},
		DataBytes: acceptData,
	}

	sig, err := s.signAndSend(ctx, createIx, acceptIx)
	if err != nil {
		return nil, fmt.Errorf("covenant: create+accept job: %w", err)
	}
	sumHex := fmt.Sprintf("%x", specHash[:])
	return &CreateJobResult{JobPDA: job, SpecHashHex: sumHex, SpecJSON: specJSON, TxSignature: sig, PosterWallet: poster.String()}, nil
}

// SubmitWork submits the provider's delivery (work hash + receipt uri) for a job.
// The signer here is the taker (provider). When DanteGPU submits on the provider's
// behalf it uses the provider's key; in the custodial Phase 1 the platform may act
// as both poster and the submitting party for its own managed providers.
func (s *Signer) SubmitWork(ctx context.Context, job, poster solana.PublicKey, workHash [32]byte, deliveryURI string) (string, error) {
	taker := s.key.PublicKey()
	if len(deliveryURI) > 128 {
		deliveryURI = deliveryURI[:128]
	}
	data := append([]byte{}, discSubmitWork...)
	data = append(data, workHash[:]...)
	data = append(data, borshString(deliveryURI)...)

	ix := &solana.GenericInstruction{
		ProgID: s.programID,
		AccountValues: solana.AccountMetaSlice{
			meta(taker, true, true),
			meta(job, true, false),
			meta(poster, false, false),
		},
		DataBytes: data,
	}
	return s.signAndSend(ctx, ix)
}

// AcceptJob accepts an open job as the taker (provider side).
func (s *Signer) AcceptJob(ctx context.Context, job, poster solana.PublicKey, specHash [32]byte) (string, error) {
	taker := s.key.PublicKey()
	data := append([]byte{}, discAcceptJob...)
	data = append(data, specHash[:]...)
	ix := &solana.GenericInstruction{
		ProgID: s.programID,
		AccountValues: solana.AccountMetaSlice{
			meta(taker, true, true),
			meta(job, true, false),
			meta(poster, false, false),
		},
		DataBytes: data,
	}
	return s.signAndSend(ctx, ix)
}

// Finalize releases the escrow to the taker after the challenge window. This is
// the provider payout, executed on-chain. crank is the platform (payer).
func (s *Signer) Finalize(ctx context.Context, job, poster, taker solana.PublicKey) (string, error) {
	crank := s.key.PublicKey()
	escrowToken, err := s.escrowTokenPDA(job)
	if err != nil {
		return "", err
	}
	takerATA, _, err := solana.FindAssociatedTokenAddress(taker, s.usdcMint)
	if err != nil {
		return "", err
	}
	rep, err := s.reputationPDA(taker)
	if err != nil {
		return "", err
	}
	claim, err := s.claimPDA(job)
	if err != nil {
		return "", err
	}

	ix := &solana.GenericInstruction{
		ProgID: s.programID,
		AccountValues: solana.AccountMetaSlice{
			meta(crank, true, true),
			meta(job, true, false),
			meta(poster, true, false),
			meta(escrowToken, true, false),
			meta(takerATA, true, false),
			meta(taker, true, false),
			meta(rep, true, false),
			meta(claim, true, false),
			meta(solana.TokenProgramID, false, false),
			meta(solana.SystemProgramID, false, false),
		},
		DataBytes: append([]byte{}, discFinalizePayment...),
	}
	return s.signAndSend(ctx, ix)
}

// RaiseDispute opens a dispute on a delivered job, posting a bond from the
// platform (poster). reasonHash is the SHA-256 of the dispute reason; the metering
// receipt is the evidence the bonded arbitrator weighs. amountAtomic is the escrow
// amount in atomic units, used to size the minimum bond.
func (s *Signer) RaiseDispute(ctx context.Context, job, poster solana.PublicKey, reasonHash [32]byte, amountAtomic uint64) (string, error) {
	disputer := s.key.PublicKey() // poster == platform in Phase 1
	config, err := s.configPDA()
	if err != nil {
		return "", err
	}
	bondToken, err := s.bondPDA(job)
	if err != nil {
		return "", err
	}
	posterATA, _, err := solana.FindAssociatedTokenAddress(disputer, s.usdcMint)
	if err != nil {
		return "", err
	}

	data := append([]byte{}, discRaiseDispute...)
	data = append(data, reasonHash[:]...)
	data = append(data, borshU64(MinBond(amountAtomic))...)

	ix := &solana.GenericInstruction{
		ProgID: s.programID,
		AccountValues: solana.AccountMetaSlice{
			meta(disputer, true, true),
			meta(config, false, false),
			meta(job, true, false),
			meta(bondToken, true, false),
			meta(posterATA, true, false),
			meta(s.usdcMint, false, false),
			meta(solana.TokenProgramID, false, false),
			meta(solana.SystemProgramID, false, false),
			meta(solana.SysVarRentPubkey, false, false),
		},
		DataBytes: data,
	}
	return s.signAndSend(ctx, ix)
}

// signAndSend assembles a transaction from one or more instructions, signs it
// with the platform key, and submits it. Multiple instructions land atomically.
func (s *Signer) signAndSend(ctx context.Context, ixs ...solana.Instruction) (string, error) {
	recent, err := s.rpc.GetLatestBlockhash(ctx, rpc.CommitmentConfirmed)
	if err != nil {
		return "", fmt.Errorf("get blockhash: %w", err)
	}
	tx, err := solana.NewTransaction(ixs, recent.Value.Blockhash, solana.TransactionPayer(s.key.PublicKey()))
	if err != nil {
		return "", fmt.Errorf("new tx: %w", err)
	}
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(s.key.PublicKey()) {
			return &s.key
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	sig, err := s.rpc.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{PreflightCommitment: rpc.CommitmentConfirmed})
	if err != nil {
		return "", fmt.Errorf("send: %w", err)
	}
	return sig.String(), nil
}
