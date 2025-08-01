-- Extended Database Schema for DanteGPU Platform
-- This script creates comprehensive tables for GPU rental, reservations, and payment tracking

-- Connect to dante_auth database for user management extensions
\c dante_auth;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enhanced users table with additional fields
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(20),
    organization VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,
    is_provider BOOLEAN DEFAULT false,
    role VARCHAR(50) DEFAULT 'user',
    subscription_tier VARCHAR(50) DEFAULT 'basic',
    credit_balance DECIMAL(20,8) DEFAULT 0,
    total_spent DECIMAL(20,8) DEFAULT 0,
    reputation_score INTEGER DEFAULT 100,
    kyc_status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login TIMESTAMP,
    metadata JSONB DEFAULT '{}'
);

-- User sessions with enhanced tracking
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    refresh_token VARCHAR(255) NOT NULL,
    access_token_hash VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    device_fingerprint VARCHAR(255),
    expires_at TIMESTAMP NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    last_used TIMESTAMP DEFAULT NOW()
);

-- User preferences and settings
CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    notification_email BOOLEAN DEFAULT true,
    notification_sms BOOLEAN DEFAULT false,
    notification_push BOOLEAN DEFAULT true,
    preferred_currency VARCHAR(10) DEFAULT 'USD',
    preferred_gpu_types TEXT[] DEFAULT '{}',
    auto_renewal BOOLEAN DEFAULT false,
    max_hourly_spend DECIMAL(10,4) DEFAULT 100,
    timezone VARCHAR(50) DEFAULT 'UTC',
    language VARCHAR(10) DEFAULT 'en',
    theme VARCHAR(20) DEFAULT 'light',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Connect to dante_registry database for provider and GPU management
\c dante_registry;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis"; -- For geographic data

-- Enhanced providers table
CREATE TABLE IF NOT EXISTS providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    website VARCHAR(255),
    contact_email VARCHAR(255),
    location VARCHAR(100),
    coordinates POINT, -- Geographic coordinates
    country_code VARCHAR(3),
    timezone VARCHAR(50),
    status VARCHAR(20) DEFAULT 'offline',
    verification_status VARCHAR(20) DEFAULT 'pending',
    total_earnings_dgpu DECIMAL(20,8) DEFAULT 0,
    total_earnings_usd DECIMAL(20,8) DEFAULT 0,
    rating DECIMAL(3,2) DEFAULT 0,
    total_jobs_completed INTEGER DEFAULT 0,
    total_uptime_hours INTEGER DEFAULT 0,
    last_heartbeat TIMESTAMP,
    commission_rate DECIMAL(5,4) DEFAULT 0.05, -- 5% platform fee
    min_rental_duration INTEGER DEFAULT 60, -- minutes
    max_rental_duration INTEGER DEFAULT 10080, -- 1 week in minutes
    auto_accept_jobs BOOLEAN DEFAULT false,
    supported_frameworks TEXT[] DEFAULT '{}',
    hardware_specs JSONB DEFAULT '{}',
    network_specs JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Enhanced GPUs table with detailed specifications
CREATE TABLE IF NOT EXISTS gpus (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID REFERENCES providers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL,
    manufacturer VARCHAR(100),
    architecture VARCHAR(100),
    vram_mb INTEGER NOT NULL,
    compute_capability VARCHAR(10),
    cuda_cores INTEGER,
    tensor_cores INTEGER,
    rt_cores INTEGER,
    base_clock_mhz INTEGER,
    boost_clock_mhz INTEGER,
    memory_bandwidth_gbps INTEGER,
    pcie_generation INTEGER,
    power_consumption_w INTEGER,
    thermal_design_power_w INTEGER,
    driver_version VARCHAR(50),
    cuda_version VARCHAR(20),
    opencl_version VARCHAR(20),
    is_available BOOLEAN DEFAULT true,
    is_fractional_supported BOOLEAN DEFAULT true,
    max_concurrent_jobs INTEGER DEFAULT 1,
    hourly_rate_dgpu DECIMAL(10,4),
    hourly_rate_usd DECIMAL(10,4),
    current_utilization DECIMAL(5,2) DEFAULT 0,
    current_memory_usage DECIMAL(5,2) DEFAULT 0,
    temperature_c INTEGER,
    power_draw_w INTEGER,
    fan_speed_percent INTEGER,
    last_health_check TIMESTAMP,
    health_status VARCHAR(20) DEFAULT 'healthy',
    maintenance_mode BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- GPU performance benchmarks
CREATE TABLE IF NOT EXISTS gpu_benchmarks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    gpu_id UUID REFERENCES gpus(id) ON DELETE CASCADE,
    benchmark_type VARCHAR(50) NOT NULL, -- e.g., 'tensorflow', 'pytorch', 'cuda'
    benchmark_name VARCHAR(100) NOT NULL,
    score DECIMAL(15,6),
    unit VARCHAR(20),
    test_date TIMESTAMP DEFAULT NOW(),
    test_duration_seconds INTEGER,
    test_conditions JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Connect to dante_scheduler database for job and rental management
\c dante_scheduler;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enhanced jobs table with comprehensive tracking
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    provider_id UUID,
    gpu_id UUID,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    job_type VARCHAR(50) DEFAULT 'custom', -- custom, training, inference, mining
    framework VARCHAR(50), -- tensorflow, pytorch, etc.
    docker_image VARCHAR(255),
    docker_tag VARCHAR(100) DEFAULT 'latest',
    command TEXT,
    environment_vars JSONB DEFAULT '{}',
    resource_requirements JSONB DEFAULT '{}',
    input_data_urls TEXT[],
    output_data_urls TEXT[],
    status VARCHAR(20) DEFAULT 'queued',
    priority INTEGER DEFAULT 5,
    progress_percent INTEGER DEFAULT 0,
    estimated_duration_minutes INTEGER,
    actual_duration_minutes INTEGER,
    max_duration_minutes INTEGER DEFAULT 1440, -- 24 hours
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    cost_estimate_dgpu DECIMAL(20,8),
    actual_cost_dgpu DECIMAL(20,8),
    cost_estimate_usd DECIMAL(20,8),
    actual_cost_usd DECIMAL(20,8),
    created_at TIMESTAMP DEFAULT NOW(),
    queued_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    failed_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    last_heartbeat TIMESTAMP,
    metadata JSONB DEFAULT '{}'
);

-- Job execution logs
CREATE TABLE IF NOT EXISTS job_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    log_type VARCHAR(20) NOT NULL, -- stdout, stderr, system, error
    message TEXT NOT NULL,
    log_level VARCHAR(10) DEFAULT 'info',
    source VARCHAR(50), -- container, system, scheduler
    timestamp TIMESTAMP DEFAULT NOW(),
    sequence_number INTEGER,
    metadata JSONB DEFAULT '{}'
);

-- GPU reservations system
CREATE TABLE IF NOT EXISTS gpu_reservations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    gpu_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    reservation_type VARCHAR(20) DEFAULT 'immediate', -- immediate, scheduled, recurring
    status VARCHAR(20) DEFAULT 'pending', -- pending, confirmed, active, completed, cancelled
    requested_start_time TIMESTAMP NOT NULL,
    requested_end_time TIMESTAMP NOT NULL,
    actual_start_time TIMESTAMP,
    actual_end_time TIMESTAMP,
    duration_minutes INTEGER,
    gpu_fraction DECIMAL(3,2) DEFAULT 1.0, -- 0.1 to 1.0
    memory_allocation_mb INTEGER,
    hourly_rate_dgpu DECIMAL(10,4),
    hourly_rate_usd DECIMAL(10,4),
    total_cost_dgpu DECIMAL(20,8),
    total_cost_usd DECIMAL(20,8),
    deposit_amount_dgpu DECIMAL(20,8),
    refund_amount_dgpu DECIMAL(20,8),
    cancellation_fee_dgpu DECIMAL(20,8) DEFAULT 0,
    auto_extend BOOLEAN DEFAULT false,
    max_extension_hours INTEGER DEFAULT 0,
    notification_sent BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    cancelled_at TIMESTAMP,
    cancellation_reason TEXT
);

-- Rental history for comprehensive tracking
CREATE TABLE IF NOT EXISTS rental_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    gpu_id UUID NOT NULL,
    job_id UUID,
    reservation_id UUID REFERENCES gpu_reservations(id),
    rental_type VARCHAR(20) DEFAULT 'on-demand', -- on-demand, reserved, spot
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    duration_minutes INTEGER,
    gpu_utilization_avg DECIMAL(5,2),
    memory_utilization_avg DECIMAL(5,2),
    power_consumption_avg INTEGER,
    cost_per_minute_dgpu DECIMAL(15,8),
    cost_per_minute_usd DECIMAL(15,8),
    total_cost_dgpu DECIMAL(20,8),
    total_cost_usd DECIMAL(20,8),
    billing_session_id UUID,
    payment_status VARCHAR(20) DEFAULT 'pending',
    rating_given INTEGER, -- 1-5 stars
    review_text TEXT,
    issues_reported TEXT[],
    performance_metrics JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Connect to dante_billing database for payment and financial tracking
\c dante_billing;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enhanced wallets table with multi-currency support
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    wallet_type VARCHAR(20) DEFAULT 'primary', -- primary, escrow, earnings
    blockchain VARCHAR(20) DEFAULT 'solana',
    public_key VARCHAR(88) NOT NULL,
    encrypted_private_key TEXT NOT NULL,
    encryption_method VARCHAR(50) DEFAULT 'AES-256',
    balance_dgpu DECIMAL(20,8) DEFAULT 0,
    balance_usdc DECIMAL(20,8) DEFAULT 0,
    balance_sol DECIMAL(20,8) DEFAULT 0,
    pending_balance_dgpu DECIMAL(20,8) DEFAULT 0,
    locked_balance_dgpu DECIMAL(20,8) DEFAULT 0,
    total_earned_dgpu DECIMAL(20,8) DEFAULT 0,
    total_spent_dgpu DECIMAL(20,8) DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,
    last_sync TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Enhanced transactions table with comprehensive tracking
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_wallet_id UUID REFERENCES wallets(id),
    to_wallet_id UUID REFERENCES wallets(id),
    transaction_type VARCHAR(30) NOT NULL, -- payment, refund, deposit, withdrawal, fee, reward
    amount_dgpu DECIMAL(20,8) NOT NULL,
    amount_usd DECIMAL(20,8),
    fee_dgpu DECIMAL(20,8) DEFAULT 0,
    fee_usd DECIMAL(20,8) DEFAULT 0,
    exchange_rate_usd DECIMAL(15,8),
    blockchain VARCHAR(20) DEFAULT 'solana',
    transaction_hash VARCHAR(88),
    block_number BIGINT,
    confirmation_count INTEGER DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending', -- pending, confirmed, failed, cancelled
    job_id UUID,
    reservation_id UUID,
    billing_session_id UUID,
    payment_method VARCHAR(30), -- crypto, stripe, paypal
    payment_gateway_id VARCHAR(100),
    gateway_response JSONB DEFAULT '{}',
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    expires_at TIMESTAMP,
    confirmed_at TIMESTAMP,
    failed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

-- Billing sessions for real-time usage tracking
CREATE TABLE IF NOT EXISTS billing_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL,
    user_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    gpu_id UUID NOT NULL,
    reservation_id UUID REFERENCES gpu_reservations(id),
    session_type VARCHAR(20) DEFAULT 'on-demand', -- on-demand, reserved, spot
    hourly_rate_dgpu DECIMAL(10,4) NOT NULL,
    hourly_rate_usd DECIMAL(10,4),
    minute_rate_dgpu DECIMAL(15,8),
    minute_rate_usd DECIMAL(15,8),
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    last_billed_at TIMESTAMP,
    total_minutes INTEGER DEFAULT 0,
    billed_minutes INTEGER DEFAULT 0,
    total_cost_dgpu DECIMAL(20,8) DEFAULT 0,
    total_cost_usd DECIMAL(20,8) DEFAULT 0,
    platform_fee_dgpu DECIMAL(20,8) DEFAULT 0,
    provider_earnings_dgpu DECIMAL(20,8) DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active', -- active, paused, completed, cancelled
    auto_billing_enabled BOOLEAN DEFAULT true,
    billing_interval_minutes INTEGER DEFAULT 1,
    grace_period_minutes INTEGER DEFAULT 5,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Payment methods for users
CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    method_type VARCHAR(20) NOT NULL, -- crypto, stripe, paypal
    provider VARCHAR(30) NOT NULL, -- solana, stripe, paypal
    external_id VARCHAR(100), -- Stripe customer ID, PayPal account, etc.
    is_default BOOLEAN DEFAULT false,
    is_verified BOOLEAN DEFAULT false,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Invoices and receipts
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    invoice_number VARCHAR(50) UNIQUE NOT NULL,
    billing_session_id UUID REFERENCES billing_sessions(id),
    invoice_type VARCHAR(20) DEFAULT 'rental', -- rental, deposit, fee
    subtotal_dgpu DECIMAL(20,8) NOT NULL,
    subtotal_usd DECIMAL(20,8),
    tax_amount_dgpu DECIMAL(20,8) DEFAULT 0,
    tax_amount_usd DECIMAL(20,8) DEFAULT 0,
    total_amount_dgpu DECIMAL(20,8) NOT NULL,
    total_amount_usd DECIMAL(20,8),
    currency VARCHAR(10) DEFAULT 'dGPU',
    status VARCHAR(20) DEFAULT 'draft', -- draft, sent, paid, overdue, cancelled
    due_date TIMESTAMP,
    paid_at TIMESTAMP,
    payment_transaction_id UUID REFERENCES transactions(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);

CREATE INDEX IF NOT EXISTS idx_providers_status ON providers(status);
CREATE INDEX IF NOT EXISTS idx_providers_location ON providers(location);
CREATE INDEX IF NOT EXISTS idx_providers_rating ON providers(rating);
CREATE INDEX IF NOT EXISTS idx_gpus_provider_id ON gpus(provider_id);
CREATE INDEX IF NOT EXISTS idx_gpus_is_available ON gpus(is_available);
CREATE INDEX IF NOT EXISTS idx_gpus_model ON gpus(model);

CREATE INDEX IF NOT EXISTS idx_jobs_user_id ON jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);
CREATE INDEX IF NOT EXISTS idx_job_logs_job_id ON job_logs(job_id);
CREATE INDEX IF NOT EXISTS idx_job_logs_timestamp ON job_logs(timestamp);

CREATE INDEX IF NOT EXISTS idx_reservations_user_id ON gpu_reservations(user_id);
CREATE INDEX IF NOT EXISTS idx_reservations_gpu_id ON gpu_reservations(gpu_id);
CREATE INDEX IF NOT EXISTS idx_reservations_status ON gpu_reservations(status);
CREATE INDEX IF NOT EXISTS idx_reservations_start_time ON gpu_reservations(requested_start_time);

CREATE INDEX IF NOT EXISTS idx_rental_history_user_id ON rental_history(user_id);
CREATE INDEX IF NOT EXISTS idx_rental_history_provider_id ON rental_history(provider_id);
CREATE INDEX IF NOT EXISTS idx_rental_history_started_at ON rental_history(started_at);

CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);
CREATE INDEX IF NOT EXISTS idx_wallets_public_key ON wallets(public_key);
CREATE INDEX IF NOT EXISTS idx_transactions_from_wallet ON transactions(from_wallet_id);
CREATE INDEX IF NOT EXISTS idx_transactions_to_wallet ON transactions(to_wallet_id);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);

CREATE INDEX IF NOT EXISTS idx_billing_sessions_job_id ON billing_sessions(job_id);
CREATE INDEX IF NOT EXISTS idx_billing_sessions_user_id ON billing_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_billing_sessions_status ON billing_sessions(status);

SELECT 'Extended DanteGPU database schema created successfully.' AS status;
