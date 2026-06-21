package solana

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/mr-tron/base58"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Client represents a Solana blockchain client for dGPU token operations
type Client struct {
	rpcClient *rpc.Client
	wsClient  *ws.Client
	logger    *zap.Logger

	// dGPU token configuration
	tokenMint      solana.PublicKey
	platformWallet solana.PublicKey
	privateKey     solana.PrivateKey

	// Configuration
	commitment rpc.CommitmentType
	timeout    time.Duration
	maxRetries int
}

// Config represents Solana client configuration
type Config struct {
	RPCURL         string        `yaml:"rpc_url"`
	WSURL          string        `yaml:"ws_url"`
	TokenAddress   string        `yaml:"dgpu_token_address"`
	PlatformWallet string        `yaml:"platform_wallet"`
	PrivateKeyPath string        `yaml:"private_key_path"`
	Commitment     string        `yaml:"commitment"`
	Timeout        time.Duration `yaml:"timeout"`
	MaxRetries     int           `yaml:"max_retries"`
}

// NewClient creates a new Solana client for dGPU token operations
func NewClient(cfg *Config, logger *zap.Logger) (*Client, error) {
	// Parse token mint address
	tokenMint, err := solana.PublicKeyFromBase58(cfg.TokenAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid token address: %w", err)
	}

	// Parse platform wallet address
	platformWallet, err := solana.PublicKeyFromBase58(cfg.PlatformWallet)
	if err != nil {
		return nil, fmt.Errorf("invalid platform wallet address: %w", err)
	}

	// Load private key (in production, this should be from secure key management)
	privateKey, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	// Create RPC client
	rpcClient := rpc.New(cfg.RPCURL)

	// Create WebSocket client for real-time updates
	wsClient, err := ws.Connect(context.Background(), cfg.WSURL)
	if err != nil {
		logger.Warn("Failed to connect to Solana WebSocket", zap.Error(err))
		// Continue without WebSocket - it's not critical for basic operations
	}

	// Parse commitment level
	commitment := rpc.CommitmentConfirmed
	switch cfg.Commitment {
	case "processed":
		commitment = rpc.CommitmentProcessed
	case "confirmed":
		commitment = rpc.CommitmentConfirmed
	case "finalized":
		commitment = rpc.CommitmentFinalized
	}

	client := &Client{
		rpcClient:      rpcClient,
		wsClient:       wsClient,
		logger:         logger,
		tokenMint:      tokenMint,
		platformWallet: platformWallet,
		privateKey:     privateKey,
		commitment:     commitment,
		timeout:        cfg.Timeout,
		maxRetries:     cfg.MaxRetries,
	}

	// Test connection
	if err := client.testConnection(); err != nil {
		return nil, fmt.Errorf("failed to connect to Solana: %w", err)
	}

	logger.Info("Solana client initialized successfully",
		zap.String("rpc_url", cfg.RPCURL),
		zap.String("token_mint", tokenMint.String()),
		zap.String("platform_wallet", platformWallet.String()),
	)

	return client, nil
}

// testConnection tests the connection to Solana RPC
func (c *Client) testConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	_, err := c.rpcClient.GetHealth(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// GetTokenBalance gets the dGPU token balance for a given wallet address
func (c *Client) GetTokenBalance(ctx context.Context, walletAddress string) (decimal.Decimal, error) {
	pubKey, err := solana.PublicKeyFromBase58(walletAddress)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid wallet address: %w", err)
	}

	// Get associated token account
	ata, _, err := solana.FindAssociatedTokenAddress(pubKey, c.tokenMint)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to find associated token account: %w", err)
	}

	// Get token account balance
	balance, err := c.rpcClient.GetTokenAccountBalance(ctx, ata, c.commitment)
	if err != nil {
		// If account doesn't exist, balance is zero
		if strings.Contains(err.Error(), "could not find account") {
			return decimal.Zero, nil
		}
		return decimal.Zero, fmt.Errorf("failed to get token balance: %w", err)
	}

	// Convert to decimal (assuming 9 decimals for dGPU token)
	amount, err := decimal.NewFromString(balance.Value.Amount)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to parse balance amount: %w", err)
	}

	// Adjust for token decimals
	decimals := decimal.NewFromInt(int64(balance.Value.Decimals))
	divisor := decimal.NewFromInt(10).Pow(decimals)

	return amount.Div(divisor), nil
}

// TransferTokens transfers dGPU tokens between wallets
func (c *Client) TransferTokens(ctx context.Context, fromAddress, toAddress string, amount decimal.Decimal) (string, error) {
	fromPubKey, err := solana.PublicKeyFromBase58(fromAddress)
	if err != nil {
		return "", fmt.Errorf("invalid from address: %w", err)
	}

	toPubKey, err := solana.PublicKeyFromBase58(toAddress)
	if err != nil {
		return "", fmt.Errorf("invalid to address: %w", err)
	}

	// Get associated token accounts
	fromATA, _, err := solana.FindAssociatedTokenAddress(fromPubKey, c.tokenMint)
	if err != nil {
		return "", fmt.Errorf("failed to find from ATA: %w", err)
	}

	toATA, _, err := solana.FindAssociatedTokenAddress(toPubKey, c.tokenMint)
	if err != nil {
		return "", fmt.Errorf("failed to find to ATA: %w", err)
	}

	// Convert amount to token units (assuming 9 decimals)
	decimals := decimal.NewFromInt(1000000000) // 10^9
	tokenAmount := amount.Mul(decimals).BigInt().Uint64()

	// Create transfer instruction
	transferInstruction := token.NewTransferInstruction(
		tokenAmount,
		fromATA,
		toATA,
		fromPubKey,
		[]solana.PublicKey{},
	).Build()

	// Get recent blockhash
	recent, err := c.rpcClient.GetRecentBlockhash(ctx, c.commitment)
	if err != nil {
		return "", fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Create transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{transferInstruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(fromPubKey),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(fromPubKey) {
			return &c.privateKey
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	signature, err := c.rpcClient.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
		SkipPreflight:       false,
		PreflightCommitment: c.commitment,
	})
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	c.logger.Info("Token transfer transaction sent",
		zap.String("signature", signature.String()),
		zap.String("from", fromAddress),
		zap.String("to", toAddress),
		zap.String("amount", amount.String()),
	)

	return signature.String(), nil
}

// ConfirmTransaction waits for transaction confirmation
func (c *Client) ConfirmTransaction(ctx context.Context, signature string) error {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	// Wait for confirmation with timeout
	confirmCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	for {
		select {
		case <-confirmCtx.Done():
			return fmt.Errorf("transaction confirmation timeout")
		default:
			status, err := c.rpcClient.GetSignatureStatuses(confirmCtx, true, sig)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			if len(status.Value) > 0 && status.Value[0] != nil {
				if status.Value[0].Err != nil {
					return fmt.Errorf("transaction failed: %v", status.Value[0].Err)
				}

				// Check confirmation status
				switch status.Value[0].ConfirmationStatus {
				case rpc.ConfirmationStatusProcessed:
					if c.commitment == rpc.CommitmentProcessed {
						return nil
					}
				case rpc.ConfirmationStatusConfirmed:
					if c.commitment == rpc.CommitmentConfirmed || c.commitment == rpc.CommitmentProcessed {
						return nil
					}
				case rpc.ConfirmationStatusFinalized:
					return nil
				}
			}

			time.Sleep(1 * time.Second)
		}
	}
}

// CreateAssociatedTokenAccount creates an associated token account for a wallet
func (c *Client) CreateAssociatedTokenAccount(ctx context.Context, walletAddress string) (string, error) {
	pubKey, err := solana.PublicKeyFromBase58(walletAddress)
	if err != nil {
		return "", fmt.Errorf("invalid wallet address: %w", err)
	}

	// Check if ATA already exists
	ata, _, err := solana.FindAssociatedTokenAddress(pubKey, c.tokenMint)
	if err != nil {
		return "", fmt.Errorf("failed to find ATA: %w", err)
	}

	// Check if account exists
	_, err = c.rpcClient.GetAccountInfo(ctx, ata)
	if err == nil {
		// Account already exists
		return ata.String(), nil
	}

	// Create ATA instruction using associatedtokenaccount package
	createATAInstruction := associatedtokenaccount.NewCreateInstruction(
		c.platformWallet, // Payer
		pubKey,           // Wallet
		c.tokenMint,      // Token mint
	).Build()

	// Get recent blockhash
	recent, err := c.rpcClient.GetRecentBlockhash(ctx, c.commitment)
	if err != nil {
		return "", fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Create transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{createATAInstruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(c.platformWallet),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(c.platformWallet) {
			return &c.privateKey
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	signature, err := c.rpcClient.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
		SkipPreflight:       false,
		PreflightCommitment: c.commitment,
	})
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for confirmation
	if err := c.ConfirmTransaction(ctx, signature.String()); err != nil {
		return "", fmt.Errorf("failed to confirm ATA creation: %w", err)
	}

	c.logger.Info("Associated token account created",
		zap.String("wallet", walletAddress),
		zap.String("ata", ata.String()),
		zap.String("signature", signature.String()),
	)

	return ata.String(), nil
}

// Close closes the Solana client connections
func (c *Client) Close() error {
	if c.wsClient != nil {
		c.wsClient.Close()
	}
	return nil
}

// loadPrivateKey loads a private key from file or environment
func loadPrivateKey(path string) (solana.PrivateKey, error) {
	// First try to load from environment variable (for development)
	if envKey := os.Getenv("SOLANA_PRIVATE_KEY"); envKey != "" {
		keyBytes, err := base58.Decode(envKey)
		if err != nil {
			return solana.PrivateKey{}, fmt.Errorf("failed to decode private key from environment: %w", err)
		}
		if len(keyBytes) != 64 {
			return solana.PrivateKey{}, fmt.Errorf("invalid private key length: expected 64 bytes, got %d", len(keyBytes))
		}
		var privateKey solana.PrivateKey
		copy(privateKey[:], keyBytes)
		return privateKey, nil
	}

	// Try to load from file
	if path != "" {
		keyData, err := os.ReadFile(path)
		if err != nil {
			return solana.PrivateKey{}, fmt.Errorf("failed to read private key file: %w", err)
		}

		// Try to parse as base58 encoded key
		keyStr := strings.TrimSpace(string(keyData))
		keyBytes, err := base58.Decode(keyStr)
		if err != nil {
			return solana.PrivateKey{}, fmt.Errorf("failed to decode private key from file: %w", err)
		}

		if len(keyBytes) != 64 {
			return solana.PrivateKey{}, fmt.Errorf("invalid private key length: expected 64 bytes, got %d", len(keyBytes))
		}

		var privateKey solana.PrivateKey
		copy(privateKey[:], keyBytes)
		return privateKey, nil
	}

	// Generate a new keypair for development (NOT for production)
	if os.Getenv("DEVELOPMENT_MODE") == "true" {
		account := solana.NewWallet()
		fmt.Printf("Generated new development keypair. Public key: %s\n", account.PublicKey().String())
		fmt.Printf("Private key (base58): %s\n", base58.Encode(account.PrivateKey))
		return account.PrivateKey, nil
	}

	return solana.PrivateKey{}, fmt.Errorf("no private key found. Set SOLANA_PRIVATE_KEY environment variable or provide key file path")
}
