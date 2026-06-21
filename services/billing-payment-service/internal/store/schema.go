package store

// Database schema definitions for the billing and payment service

const createWalletsTable = `
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    wallet_type VARCHAR(50) NOT NULL CHECK (wallet_type IN ('user', 'provider', 'platform')),
    solana_address VARCHAR(255) NOT NULL,
    balance DECIMAL(20,9) NOT NULL DEFAULT 0,
    locked_balance DECIMAL(20,9) NOT NULL DEFAULT 0,
    pending_balance DECIMAL(20,9) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ,
    
    UNIQUE(user_id, wallet_type),
    UNIQUE(solana_address)
);
`

const createTransactionsTable = `
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    from_wallet_id UUID REFERENCES wallets(id),
    to_wallet_id UUID REFERENCES wallets(id),
    type VARCHAR(50) NOT NULL CHECK (type IN (
        'deposit', 'withdrawal', 'payment', 'payout', 'refund', 
        'platform_fee', 'session_start', 'session_end', 'session_billing'
    )),
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'confirmed', 'failed', 'cancelled')),
    amount DECIMAL(20,9) NOT NULL,
    fee DECIMAL(20,9) NOT NULL DEFAULT 0,
    description TEXT NOT NULL,
    solana_signature VARCHAR(255),
    session_id UUID,
    job_id VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    
    CHECK (amount > 0),
    CHECK (fee >= 0),
    CHECK (from_wallet_id IS NOT NULL OR to_wallet_id IS NOT NULL)
);
`

const createRentalSessionsTable = `
CREATE TABLE IF NOT EXISTS rental_sessions (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    provider_id UUID NOT NULL,
    job_id VARCHAR(255),
    status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'completed', 'cancelled', 'suspended', 'terminated')),
    
    -- GPU allocation details
    gpu_model VARCHAR(255) NOT NULL,
    allocated_vram_mb BIGINT NOT NULL,
    total_vram_mb BIGINT NOT NULL,
    vram_percentage DECIMAL(5,2) NOT NULL,
    
    -- Pricing information (in dGPU tokens)
    hourly_rate DECIMAL(20,9) NOT NULL,
    vram_rate DECIMAL(20,9) NOT NULL,
    power_rate DECIMAL(20,9) NOT NULL,
    platform_fee_rate DECIMAL(5,2) NOT NULL,
    
    -- Power consumption
    estimated_power_w INTEGER NOT NULL,
    actual_power_w INTEGER,
    
    -- Session timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    last_billed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Financial tracking
    total_cost DECIMAL(20,9) NOT NULL DEFAULT 0,
    platform_fee DECIMAL(20,9) NOT NULL DEFAULT 0,
    provider_earnings DECIMAL(20,9) NOT NULL DEFAULT 0,
    
    -- Metadata
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CHECK (allocated_vram_mb > 0),
    CHECK (total_vram_mb > 0),
    CHECK (allocated_vram_mb <= total_vram_mb),
    CHECK (vram_percentage >= 0 AND vram_percentage <= 100),
    CHECK (hourly_rate >= 0),
    CHECK (vram_rate >= 0),
    CHECK (power_rate >= 0),
    CHECK (platform_fee_rate >= 0 AND platform_fee_rate <= 100),
    CHECK (estimated_power_w > 0),
    CHECK (total_cost >= 0),
    CHECK (platform_fee >= 0),
    CHECK (provider_earnings >= 0)
);
`

const createUsageRecordsTable = `
CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES rental_sessions(id) ON DELETE CASCADE,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- GPU utilization metrics
    gpu_utilization_percent SMALLINT CHECK (gpu_utilization_percent >= 0 AND gpu_utilization_percent <= 100),
    vram_utilization_percent SMALLINT CHECK (vram_utilization_percent >= 0 AND vram_utilization_percent <= 100),
    power_draw_w INTEGER NOT NULL,
    temperature_c SMALLINT,
    
    -- Billing calculations for this period
    period_minutes INTEGER NOT NULL,
    period_cost DECIMAL(20,9) NOT NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CHECK (power_draw_w >= 0),
    CHECK (period_minutes > 0),
    CHECK (period_cost >= 0)
);
`

const createBillingRecordsTable = `
CREATE TABLE IF NOT EXISTS billing_records (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    provider_id UUID NOT NULL,
    session_id UUID NOT NULL REFERENCES rental_sessions(id),
    
    -- Billing period
    billing_period_start TIMESTAMPTZ NOT NULL,
    billing_period_end TIMESTAMPTZ NOT NULL,
    
    -- Usage summary
    total_minutes INTEGER NOT NULL,
    avg_gpu_utilization DECIMAL(5,2),
    avg_vram_utilization DECIMAL(5,2),
    avg_power_draw DECIMAL(8,2),
    
    -- Cost breakdown (in dGPU tokens)
    base_cost DECIMAL(20,9) NOT NULL,
    vram_cost DECIMAL(20,9) NOT NULL,
    power_cost DECIMAL(20,9) NOT NULL,
    total_cost DECIMAL(20,9) NOT NULL,
    platform_fee DECIMAL(20,9) NOT NULL,
    provider_earnings DECIMAL(20,9) NOT NULL,
    
    -- Transaction reference
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CHECK (billing_period_start < billing_period_end),
    CHECK (total_minutes > 0),
    CHECK (avg_gpu_utilization >= 0 AND avg_gpu_utilization <= 100),
    CHECK (avg_vram_utilization >= 0 AND avg_vram_utilization <= 100),
    CHECK (avg_power_draw >= 0),
    CHECK (base_cost >= 0),
    CHECK (vram_cost >= 0),
    CHECK (power_cost >= 0),
    CHECK (total_cost >= 0),
    CHECK (platform_fee >= 0),
    CHECK (provider_earnings >= 0)
);
`

const createProviderRatesTable = `
CREATE TABLE IF NOT EXISTS provider_rates (
    id UUID PRIMARY KEY,
    provider_id UUID NOT NULL,
    gpu_model VARCHAR(255) NOT NULL,
    
    -- Custom rates set by provider (in dGPU tokens)
    hourly_rate DECIMAL(20,9),
    vram_rate_per_gb DECIMAL(20,9),
    power_rate_per_watt DECIMAL(20,9),
    minimum_session_minutes INTEGER,
    maximum_session_hours INTEGER,
    
    -- Availability settings
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    available_vram_mb BIGINT,
    max_concurrent_sessions INTEGER DEFAULT 1,
    
    -- Metadata
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(provider_id, gpu_model),
    CHECK (hourly_rate >= 0),
    CHECK (vram_rate_per_gb >= 0),
    CHECK (power_rate_per_watt >= 0),
    CHECK (minimum_session_minutes > 0),
    CHECK (maximum_session_hours > 0),
    CHECK (available_vram_mb > 0),
    CHECK (max_concurrent_sessions > 0)
);
`

const createIndexes = `
-- Wallet indexes
CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);
CREATE INDEX IF NOT EXISTS idx_wallets_solana_address ON wallets(solana_address);
CREATE INDEX IF NOT EXISTS idx_wallets_type_active ON wallets(wallet_type, is_active);

-- Transaction indexes
CREATE INDEX IF NOT EXISTS idx_transactions_from_wallet ON transactions(from_wallet_id);
CREATE INDEX IF NOT EXISTS idx_transactions_to_wallet ON transactions(to_wallet_id);
CREATE INDEX IF NOT EXISTS idx_transactions_type_status ON transactions(type, status);
CREATE INDEX IF NOT EXISTS idx_transactions_session_id ON transactions(session_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);
CREATE INDEX IF NOT EXISTS idx_transactions_solana_signature ON transactions(solana_signature);

-- Rental session indexes
CREATE INDEX IF NOT EXISTS idx_rental_sessions_user_id ON rental_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_rental_sessions_provider_id ON rental_sessions(provider_id);
CREATE INDEX IF NOT EXISTS idx_rental_sessions_status ON rental_sessions(status);
CREATE INDEX IF NOT EXISTS idx_rental_sessions_job_id ON rental_sessions(job_id);
CREATE INDEX IF NOT EXISTS idx_rental_sessions_started_at ON rental_sessions(started_at);
CREATE INDEX IF NOT EXISTS idx_rental_sessions_gpu_model ON rental_sessions(gpu_model);

-- Usage record indexes
CREATE INDEX IF NOT EXISTS idx_usage_records_session_id ON usage_records(session_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_recorded_at ON usage_records(recorded_at);
CREATE INDEX IF NOT EXISTS idx_usage_records_session_recorded ON usage_records(session_id, recorded_at);

-- Billing record indexes
CREATE INDEX IF NOT EXISTS idx_billing_records_user_id ON billing_records(user_id);
CREATE INDEX IF NOT EXISTS idx_billing_records_provider_id ON billing_records(provider_id);
CREATE INDEX IF NOT EXISTS idx_billing_records_session_id ON billing_records(session_id);
CREATE INDEX IF NOT EXISTS idx_billing_records_period_start ON billing_records(billing_period_start);
CREATE INDEX IF NOT EXISTS idx_billing_records_transaction_id ON billing_records(transaction_id);

-- Provider rates indexes
CREATE INDEX IF NOT EXISTS idx_provider_rates_provider_id ON provider_rates(provider_id);
CREATE INDEX IF NOT EXISTS idx_provider_rates_gpu_model ON provider_rates(gpu_model);
CREATE INDEX IF NOT EXISTS idx_provider_rates_active ON provider_rates(is_active);
`
