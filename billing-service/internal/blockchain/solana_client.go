package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// SolanaClient handles Solana blockchain interactions
type SolanaClient struct {
	rpcClient    *rpc.Client
	wsClient     *ws.Client
	tokenMint    solana.PublicKey
	platformKey  solana.PrivateKey
	commitment   rpc.CommitmentType
	maxRetries   int
	retryDelay   time.Duration
}

// Config holds Solana client configuration
type Config struct {
	RPCURL         string
	WSURL          string
	TokenMint      string
	PlatformKey    string
	Commitment     string
	MaxRetries     int
	RetryDelay     time.Duration
}

// TransactionResult represents a transaction result
type TransactionResult struct {
	Signature   string
	BlockTime   *time.Time
	Slot        uint64
	Confirmed   bool
	Error       error
}

// NewSolanaClient creates a new Solana client
func NewSolanaClient(config *Config) (*SolanaClient, error) {
	// Create RPC client
	rpcClient := rpc.New(config.RPCURL)

	// Create WebSocket client
	wsClient, err := ws.Connect(context.Background(), config.WSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	// Parse token mint address
	tokenMint, err := solana.PublicKeyFromBase58(config.TokenMint)
	if err != nil {
		return nil, fmt.Errorf("invalid token mint address: %w", err)
	}

	// Parse platform private key
	platformKey, err := solana.PrivateKeyFromBase58(config.PlatformKey)
	if err != nil {
		return nil, fmt.Errorf("invalid platform private key: %w", err)
	}

	// Parse commitment level
	commitment := rpc.CommitmentConfirmed
	switch config.Commitment {
	case "finalized":
		commitment = rpc.CommitmentFinalized
	case "confirmed":
		commitment = rpc.CommitmentConfirmed
	case "processed":
		commitment = rpc.CommitmentProcessed
	}

	return &SolanaClient{
		rpcClient:   rpcClient,
		wsClient:    wsClient,
		tokenMint:   tokenMint,
		platformKey: platformKey,
		commitment:  commitment,
		maxRetries:  config.MaxRetries,
		retryDelay:  config.RetryDelay,
	}, nil
}

// CreateWallet creates a new Solana wallet
func (c *SolanaClient) CreateWallet(ctx context.Context) (*solana.PrivateKey, *solana.PublicKey, error) {
	// Generate new keypair
	privateKey := solana.NewWallet()
	publicKey := privateKey.PublicKey()

	return &privateKey, &publicKey, nil
}

// GetBalance gets the dGPU token balance for an address
func (c *SolanaClient) GetBalance(ctx context.Context, address solana.PublicKey) (uint64, error) {
	// Get associated token account
	ata, _, err := solana.FindAssociatedTokenAddress(address, c.tokenMint)
	if err != nil {
		return 0, fmt.Errorf("failed to find associated token address: %w", err)
	}

	// Get token account balance
	balance, err := c.rpcClient.GetTokenAccountBalance(ctx, ata, c.commitment)
	if err != nil {
		// Account might not exist yet
		return 0, nil
	}

	return balance.Value.Amount, nil
}

// Transfer transfers dGPU tokens
func (c *SolanaClient) Transfer(ctx context.Context, from solana.PrivateKey, to solana.PublicKey, amount uint64) (*TransactionResult, error) {
	fromPubkey := from.PublicKey()

	// Get associated token accounts
	fromATA, _, err := solana.FindAssociatedTokenAddress(fromPubkey, c.tokenMint)
	if err != nil {
		return nil, fmt.Errorf("failed to find from ATA: %w", err)
	}

	toATA, _, err := solana.FindAssociatedTokenAddress(to, c.tokenMint)
	if err != nil {
		return nil, fmt.Errorf("failed to find to ATA: %w", err)
	}

	// Check if destination ATA exists, create if not
	exists, err := c.accountExists(ctx, toATA)
	if err != nil {
		return nil, fmt.Errorf("failed to check ATA existence: %w", err)
	}

	var instructions []solana.Instruction

	if !exists {
		// Create associated token account
		createATAIx := token.NewCreateAssociatedTokenAccountInstruction(
			fromPubkey,
			to,
			c.tokenMint,
		).Build()
		instructions = append(instructions, createATAIx)
	}

	// Create transfer instruction
	transferIx := token.NewTransferInstruction(
		amount,
		fromATA,
		toATA,
		fromPubkey,
		[]solana.PublicKey{},
	).Build()
	instructions = append(instructions, transferIx)

	// Send transaction
	return c.sendTransaction(ctx, instructions, []solana.PrivateKey{from})
}

// CreateEscrow creates an escrow account for rental payment
func (c *SolanaClient) CreateEscrow(ctx context.Context, user solana.PrivateKey, amount uint64, rentalID string) (*TransactionResult, error) {
	// In production, this would create a PDA (Program Derived Address) for escrow
	// For now, we'll transfer to platform wallet with memo
	
	platformPubkey := c.platformKey.PublicKey()
	
	// Transfer to platform wallet (escrow)
	result, err := c.Transfer(ctx, user, platformPubkey, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to create escrow: %w", err)
	}

	return result, nil
}

// ReleaseEscrow releases funds from escrow
func (c *SolanaClient) ReleaseEscrow(ctx context.Context, provider solana.PublicKey, amount uint64, platformFee uint64) (*TransactionResult, error) {
	// Calculate amounts
	providerAmount := amount - platformFee

	// Transfer to provider
	result, err := c.Transfer(ctx, c.platformKey, provider, providerAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to release escrow: %w", err)
	}

	// Platform fee stays in platform wallet

	return result, nil
}

// RefundEscrow refunds remaining escrow to user
func (c *SolanaClient) RefundEscrow(ctx context.Context, user solana.PublicKey, amount uint64) (*TransactionResult, error) {
	// Transfer back to user
	result, err := c.Transfer(ctx, c.platformKey, user, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to refund escrow: %w", err)
	}

	return result, nil
}

// GetTransaction gets transaction details
func (c *SolanaClient) GetTransaction(ctx context.Context, signature string) (*TransactionResult, error) {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	tx, err := c.rpcClient.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Commitment: c.commitment,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	result := &TransactionResult{
		Signature: signature,
		Slot:      tx.Slot,
		Confirmed: tx.Meta != nil && tx.Meta.Err == nil,
	}

	if tx.BlockTime != nil {
		blockTime := time.Unix(int64(*tx.BlockTime), 0)
		result.BlockTime = &blockTime
	}

	if tx.Meta != nil && tx.Meta.Err != nil {
		result.Error = fmt.Errorf("transaction failed: %v", tx.Meta.Err)
	}

	return result, nil
}

// WaitForConfirmation waits for transaction confirmation
func (c *SolanaClient) WaitForConfirmation(ctx context.Context, signature string, timeout time.Duration) error {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Subscribe to signature status
	sub, err := c.wsClient.SignatureSubscribe(sig, c.commitment)
	if err != nil {
		return fmt.Errorf("failed to subscribe to signature: %w", err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("confirmation timeout")
		case notification := <-sub.Response():
			if notification.Value.Err != nil {
				return fmt.Errorf("transaction failed: %v", notification.Value.Err)
			}
			return nil
		}
	}
}

// Helper functions

func (c *SolanaClient) sendTransaction(ctx context.Context, instructions []solana.Instruction, signers []solana.PrivateKey) (*TransactionResult, error) {
	// Get recent blockhash
	recent, err := c.rpcClient.GetRecentBlockhash(ctx, c.commitment)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Build transaction
	tx, err := solana.NewTransaction(
		instructions,
		recent.Value.Blockhash,
		solana.TransactionPayer(signers[0].PublicKey()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		for _, signer := range signers {
			if signer.PublicKey().Equals(key) {
				return &signer
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction with retries
	var signature solana.Signature
	for i := 0; i < c.maxRetries; i++ {
		signature, err = c.rpcClient.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
			SkipPreflight:       false,
			PreflightCommitment: c.commitment,
		})
		if err == nil {
			break
		}
		if i < c.maxRetries-1 {
			time.Sleep(c.retryDelay)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction after %d retries: %w", c.maxRetries, err)
	}

	return &TransactionResult{
		Signature: signature.String(),
		Confirmed: false,
	}, nil
}

func (c *SolanaClient) accountExists(ctx context.Context, address solana.PublicKey) (bool, error) {
	info, err := c.rpcClient.GetAccountInfo(ctx, address)
	if err != nil {
		return false, nil
	}
	return info != nil && info.Value != nil, nil
}

// Close closes the client connections
func (c *SolanaClient) Close() error {
	if c.wsClient != nil {
		c.wsClient.Close()
	}
	return nil
}

