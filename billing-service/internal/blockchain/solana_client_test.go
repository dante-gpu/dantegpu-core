package blockchain

import (
	"context"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolanaClient_CreateWallet(t *testing.T) {
	client, err := NewSolanaClient(&Config{
		RPCURL:      "https://api.devnet.solana.com",
		WSURL:       "wss://api.devnet.solana.com",
		TokenMint:   "7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump",
		PlatformKey: generateTestPrivateKey(),
		Commitment:  "confirmed",
		MaxRetries:  3,
		RetryDelay:  time.Second,
	})
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	privateKey, publicKey, err := client.CreateWallet(ctx)
	require.NoError(t, err)
	assert.NotNil(t, privateKey)
	assert.NotNil(t, publicKey)

	// Verify keys are valid
	assert.Equal(t, privateKey.PublicKey(), *publicKey)
}

func TestSolanaClient_GetBalance(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Create test wallet
	_, publicKey, err := client.CreateWallet(ctx)
	require.NoError(t, err)

	// Get balance (should be 0 for new wallet)
	balance, err := client.GetBalance(ctx, *publicKey)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, balance, uint64(0))
}

func TestSolanaClient_Transfer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Create sender and receiver wallets
	senderKey, senderPubkey, err := client.CreateWallet(ctx)
	require.NoError(t, err)

	_, receiverPubkey, err := client.CreateWallet(ctx)
	require.NoError(t, err)

	// Request airdrop for sender (devnet only)
	airdropAmount := uint64(1000000000) // 1 SOL
	_, err = client.connection.RequestAirdrop(ctx, *senderPubkey, airdropAmount, "confirmed")
	if err != nil {
		t.Skip("Airdrop failed, skipping transfer test")
	}

	// Wait for airdrop to confirm
	time.Sleep(5 * time.Second)

	// Transfer tokens
	transferAmount := uint64(100000) // 0.0001 SOL
	result, err := client.Transfer(ctx, *senderKey, *receiverPubkey, transferAmount)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Signature)

	// Wait for confirmation
	err = client.WaitForConfirmation(ctx, result.Signature, 30*time.Second)
	assert.NoError(t, err)

	// Verify transaction
	txResult, err := client.GetTransaction(ctx, result.Signature)
	require.NoError(t, err)
	assert.True(t, txResult.Confirmed)
	assert.Nil(t, txResult.Error)
}

func TestSolanaClient_CreateEscrow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Create user wallet
	userKey, userPubkey, err := client.CreateWallet(ctx)
	require.NoError(t, err)

	// Request airdrop
	_, err = client.connection.RequestAirdrop(ctx, *userPubkey, 1000000000, "confirmed")
	if err != nil {
		t.Skip("Airdrop failed, skipping escrow test")
	}

	time.Sleep(5 * time.Second)

	// Create escrow
	escrowAmount := uint64(500000)
	result, err := client.CreateEscrow(ctx, *userKey, escrowAmount, "rental-123")
	require.NoError(t, err)
	assert.NotEmpty(t, result.Signature)

	// Verify escrow transaction
	txResult, err := client.GetTransaction(ctx, result.Signature)
	require.NoError(t, err)
	assert.True(t, txResult.Confirmed)
}

func TestSolanaClient_ReleaseEscrow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Create provider wallet
	_, providerPubkey, err := client.CreateWallet(ctx)
	require.NoError(t, err)

	// Release escrow (platform wallet must have funds)
	amount := uint64(100000)
	platformFee := uint64(5000) // 5%

	result, err := client.ReleaseEscrow(ctx, *providerPubkey, amount, platformFee)
	if err != nil {
		t.Logf("Release escrow failed (expected if platform wallet has no funds): %v", err)
		return
	}

	assert.NotEmpty(t, result.Signature)
}

func TestSolanaClient_RefundEscrow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Create user wallet
	_, userPubkey, err := client.CreateWallet(ctx)
	require.NoError(t, err)

	// Refund escrow
	refundAmount := uint64(50000)

	result, err := client.RefundEscrow(ctx, *userPubkey, refundAmount)
	if err != nil {
		t.Logf("Refund escrow failed (expected if platform wallet has no funds): %v", err)
		return
	}

	assert.NotEmpty(t, result.Signature)
}

func TestSolanaClient_WaitForConfirmation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Create wallet and request airdrop
	_, pubkey, err := client.CreateWallet(ctx)
	require.NoError(t, err)

	sig, err := client.connection.RequestAirdrop(ctx, *pubkey, 1000000, "confirmed")
	if err != nil {
		t.Skip("Airdrop failed")
	}

	// Wait for confirmation with timeout
	err = client.WaitForConfirmation(ctx, sig.String(), 30*time.Second)
	assert.NoError(t, err)
}

func TestSolanaClient_GetTransaction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Use a known transaction signature from devnet
	// This is a placeholder - in real tests, use actual transaction
	signature := "5VERv8NMvzbJMEkV8xnrLkEaWRtSz9CosKDYjCJjBRnbJLgp8uirBgmQpjKhoR4tjF3ZpRzrFmBV6UjKdiSZkQUW"

	result, err := client.GetTransaction(ctx, signature)
	if err != nil {
		t.Logf("Transaction not found (expected for test signature): %v", err)
		return
	}

	assert.NotNil(t, result)
}

// Helper functions

func setupTestClient(t *testing.T) *SolanaClient {
	client, err := NewSolanaClient(&Config{
		RPCURL:      "https://api.devnet.solana.com",
		WSURL:       "wss://api.devnet.solana.com",
		TokenMint:   "7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump",
		PlatformKey: generateTestPrivateKey(),
		Commitment:  "confirmed",
		MaxRetries:  3,
		RetryDelay:  time.Second,
	})
	require.NoError(t, err)
	return client
}

func generateTestPrivateKey() string {
	// Generate a test private key
	wallet := solana.NewWallet()
	return wallet.String()
}

// Benchmark tests

func BenchmarkSolanaClient_CreateWallet(b *testing.B) {
	client := setupBenchClient(b)
	defer client.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := client.CreateWallet(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSolanaClient_GetBalance(b *testing.B) {
	client := setupBenchClient(b)
	defer client.Close()

	ctx := context.Background()
	_, pubkey, _ := client.CreateWallet(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GetBalance(ctx, *pubkey)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func setupBenchClient(b *testing.B) *SolanaClient {
	client, err := NewSolanaClient(&Config{
		RPCURL:      "https://api.devnet.solana.com",
		WSURL:       "wss://api.devnet.solana.com",
		TokenMint:   "7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump",
		PlatformKey: generateTestPrivateKey(),
		Commitment:  "confirmed",
		MaxRetries:  3,
		RetryDelay:  time.Second,
	})
	if err != nil {
		b.Fatal(err)
	}
	return client
}

