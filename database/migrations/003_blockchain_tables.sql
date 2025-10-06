-- DanteGPU Platform - Blockchain Integration Tables
-- Migration 003: Solana blockchain, dGPU token, and wallet management

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Wallets table for dGPU token management
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    wallet_type VARCHAR(20) NOT NULL CHECK (wallet_type IN ('user', 'provider', 'platform', 'escrow')),
    solana_address VARCHAR(44) UNIQUE NOT NULL,
    encrypted_private_key TEXT, -- Encrypted with platform key, NULL for external wallets
    balance DECIMAL(20,9) NOT NULL DEFAULT 0,
    locked_balance DECIMAL(20,9) NOT NULL DEFAULT 0,
    pending_balance DECIMAL(20,9) NOT NULL DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    is_external BOOLEAN DEFAULT false, -- True if user connected their own wallet
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT positive_balance CHECK (balance >= 0),
    CONSTRAINT positive_locked_balance CHECK (locked_balance >= 0),
    CONSTRAINT positive_pending_balance CHECK (pending_balance >= 0),
    CONSTRAINT locked_not_exceeds_balance CHECK (locked_balance <= balance)
);

-- Blockchain transactions table
CREATE TABLE IF NOT EXISTS blockchain_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_wallet_id UUID REFERENCES wallets(id) ON DELETE SET NULL,
    to_wallet_id UUID REFERENCES wallets(id) ON DELETE SET NULL,
    transaction_type VARCHAR(50) NOT NULL CHECK (transaction_type IN (
        'deposit', 'withdrawal', 'transfer', 'rental_payment', 
        'provider_payout', 'platform_fee', 'refund', 'escrow_lock', 'escrow_release'
    )),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'processing', 'confirmed', 'failed', 'cancelled'
    )),
    amount DECIMAL(20,9) NOT NULL,
    fee DECIMAL(20,9) NOT NULL DEFAULT 0,
    description TEXT,
    solana_signature VARCHAR(88), -- Solana transaction signature
    solana_slot BIGINT, -- Solana slot number
    confirmation_count INTEGER DEFAULT 0,
    session_id UUID, -- Link to rental session if applicable
    job_id VARCHAR(255), -- Link to job if applicable
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    confirmed_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT positive_amount CHECK (amount > 0),
    CONSTRAINT positive_fee CHECK (fee >= 0)
);

-- Rental sessions table for billing
CREATE TABLE IF NOT EXISTS rental_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    provider_id UUID NOT NULL,
    job_id VARCHAR(255),
    status VARCHAR(50) NOT NULL CHECK (status IN (
        'active', 'completed', 'cancelled', 'suspended', 'terminated'
    )),
    
    -- GPU allocation details
    gpu_model VARCHAR(255) NOT NULL,
    allocated_vram_mb BIGINT NOT NULL,
    total_vram_mb BIGINT NOT NULL,
    vram_percentage DECIMAL(5,2) NOT NULL,
    
    -- Pricing information (in dGPU tokens)
    hourly_rate DECIMAL(20,9) NOT NULL,
    vram_rate DECIMAL(20,9) NOT NULL,
    power_rate DECIMAL(20,9) NOT NULL,
    platform_fee_rate DECIMAL(5,2) NOT NULL DEFAULT 5.0,
    
    -- Power consumption
    estimated_power_w INTEGER NOT NULL,
    actual_power_w INTEGER,
    
    -- Time tracking
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ended_at TIMESTAMP WITH TIME ZONE,
    last_billed_at TIMESTAMP WITH TIME ZONE,
    
    -- Cost tracking
    total_cost DECIMAL(20,9) NOT NULL DEFAULT 0,
    total_platform_fee DECIMAL(20,9) NOT NULL DEFAULT 0,
    total_provider_earnings DECIMAL(20,9) NOT NULL DEFAULT 0,
    
    -- Escrow
    escrow_wallet_id UUID REFERENCES wallets(id),
    escrow_amount DECIMAL(20,9) NOT NULL DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT positive_vram CHECK (allocated_vram_mb > 0 AND total_vram_mb > 0),
    CONSTRAINT valid_vram_percentage CHECK (vram_percentage > 0 AND vram_percentage <= 100),
    CONSTRAINT positive_rates CHECK (hourly_rate > 0 AND vram_rate >= 0 AND power_rate >= 0),
    CONSTRAINT valid_platform_fee CHECK (platform_fee_rate >= 0 AND platform_fee_rate <= 100),
    CONSTRAINT positive_power CHECK (estimated_power_w > 0),
    CONSTRAINT positive_costs CHECK (total_cost >= 0 AND total_platform_fee >= 0 AND total_provider_earnings >= 0),
    CONSTRAINT positive_escrow CHECK (escrow_amount >= 0)
);

-- Usage records for minute-based billing
CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES rental_sessions(id) ON DELETE CASCADE,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- GPU utilization metrics
    gpu_utilization_percent SMALLINT CHECK (gpu_utilization_percent >= 0 AND gpu_utilization_percent <= 100),
    vram_utilization_percent SMALLINT CHECK (vram_utilization_percent >= 0 AND vram_utilization_percent <= 100),
    power_draw_w INTEGER NOT NULL,
    temperature_c SMALLINT,
    
    -- Billing calculations for this period
    period_minutes INTEGER NOT NULL,
    period_cost DECIMAL(20,9) NOT NULL,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CHECK (power_draw_w >= 0),
    CHECK (period_minutes > 0),
    CHECK (period_cost >= 0)
);

-- Escrow transactions for secure payments
CREATE TABLE IF NOT EXISTS escrow_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES rental_sessions(id) ON DELETE CASCADE,
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('lock', 'release', 'refund')),
    amount DECIMAL(20,9) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'failed')),
    blockchain_tx_id UUID REFERENCES blockchain_transactions(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT positive_amount CHECK (amount > 0)
);

-- Platform fees collection tracking
CREATE TABLE IF NOT EXISTS platform_fees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID REFERENCES rental_sessions(id) ON DELETE SET NULL,
    transaction_id UUID REFERENCES blockchain_transactions(id),
    fee_type VARCHAR(50) NOT NULL CHECK (fee_type IN ('rental', 'withdrawal', 'transfer')),
    amount_dgpu DECIMAL(20,9) NOT NULL,
    amount_usd DECIMAL(20,8),
    collected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    distributed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'collected' CHECK (status IN ('collected', 'distributed', 'pending')),
    CONSTRAINT positive_amount CHECK (amount_dgpu > 0)
);

-- Provider payouts tracking
CREATE TABLE IF NOT EXISTS provider_payouts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL,
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    amount_dgpu DECIMAL(20,9) NOT NULL,
    amount_usd DECIMAL(20,8),
    sessions_count INTEGER NOT NULL DEFAULT 0,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'processing', 'completed', 'failed', 'cancelled'
    )),
    blockchain_tx_id UUID REFERENCES blockchain_transactions(id),
    requested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    failure_reason TEXT,
    CONSTRAINT positive_amount CHECK (amount_dgpu > 0),
    CONSTRAINT positive_sessions CHECK (sessions_count >= 0),
    CONSTRAINT valid_period CHECK (period_end > period_start)
);

-- Create indexes for performance
CREATE INDEX idx_wallets_user_id ON wallets(user_id);
CREATE INDEX idx_wallets_solana_address ON wallets(solana_address);
CREATE INDEX idx_wallets_type_active ON wallets(wallet_type, is_active);

CREATE INDEX idx_blockchain_tx_from_wallet ON blockchain_transactions(from_wallet_id);
CREATE INDEX idx_blockchain_tx_to_wallet ON blockchain_transactions(to_wallet_id);
CREATE INDEX idx_blockchain_tx_signature ON blockchain_transactions(solana_signature);
CREATE INDEX idx_blockchain_tx_status_created ON blockchain_transactions(status, created_at);
CREATE INDEX idx_blockchain_tx_session ON blockchain_transactions(session_id);

CREATE INDEX idx_rental_sessions_user ON rental_sessions(user_id);
CREATE INDEX idx_rental_sessions_provider ON rental_sessions(provider_id);
CREATE INDEX idx_rental_sessions_status ON rental_sessions(status);
CREATE INDEX idx_rental_sessions_started ON rental_sessions(started_at);
CREATE INDEX idx_rental_sessions_job ON rental_sessions(job_id);

CREATE INDEX idx_usage_records_session ON usage_records(session_id);
CREATE INDEX idx_usage_records_recorded ON usage_records(recorded_at);
CREATE INDEX idx_usage_records_session_recorded ON usage_records(session_id, recorded_at);

CREATE INDEX idx_escrow_tx_session ON escrow_transactions(session_id);
CREATE INDEX idx_escrow_tx_wallet ON escrow_transactions(wallet_id);
CREATE INDEX idx_escrow_tx_status ON escrow_transactions(status);

CREATE INDEX idx_platform_fees_session ON platform_fees(session_id);
CREATE INDEX idx_platform_fees_collected ON platform_fees(collected_at);
CREATE INDEX idx_platform_fees_status ON platform_fees(status);

CREATE INDEX idx_provider_payouts_provider ON provider_payouts(provider_id);
CREATE INDEX idx_provider_payouts_wallet ON provider_payouts(wallet_id);
CREATE INDEX idx_provider_payouts_status ON provider_payouts(status);
CREATE INDEX idx_provider_payouts_period ON provider_payouts(period_start, period_end);

-- Create updated_at triggers
CREATE TRIGGER update_wallets_updated_at 
    BEFORE UPDATE ON wallets 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_blockchain_transactions_updated_at 
    BEFORE UPDATE ON blockchain_transactions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rental_sessions_updated_at 
    BEFORE UPDATE ON rental_sessions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function to calculate available balance
CREATE OR REPLACE FUNCTION wallet_available_balance(wallet_id UUID)
RETURNS DECIMAL(20,9) AS $$
DECLARE
    available DECIMAL(20,9);
BEGIN
    SELECT balance - locked_balance INTO available
    FROM wallets
    WHERE id = wallet_id;
    
    RETURN COALESCE(available, 0);
END;
$$ LANGUAGE plpgsql;

-- Create function to lock funds in wallet
CREATE OR REPLACE FUNCTION lock_wallet_funds(
    p_wallet_id UUID,
    p_amount DECIMAL(20,9)
) RETURNS BOOLEAN AS $$
DECLARE
    current_available DECIMAL(20,9);
BEGIN
    -- Get current available balance
    SELECT balance - locked_balance INTO current_available
    FROM wallets
    WHERE id = p_wallet_id
    FOR UPDATE;
    
    -- Check if sufficient funds
    IF current_available < p_amount THEN
        RETURN FALSE;
    END IF;
    
    -- Lock the funds
    UPDATE wallets
    SET locked_balance = locked_balance + p_amount,
        updated_at = NOW()
    WHERE id = p_wallet_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Create function to release locked funds
CREATE OR REPLACE FUNCTION release_wallet_funds(
    p_wallet_id UUID,
    p_amount DECIMAL(20,9)
) RETURNS BOOLEAN AS $$
BEGIN
    UPDATE wallets
    SET locked_balance = GREATEST(0, locked_balance - p_amount),
        updated_at = NOW()
    WHERE id = p_wallet_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

