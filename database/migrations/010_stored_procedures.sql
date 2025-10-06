-- DanteGPU Platform - Stored Procedures and Functions
-- Migration 010: Business logic functions for complex operations

-- ============================================================================
-- RENTAL SESSION MANAGEMENT
-- ============================================================================

-- Function to start a rental session
CREATE OR REPLACE FUNCTION start_rental_session(
    p_user_id VARCHAR,
    p_provider_id UUID,
    p_gpu_capability_id UUID,
    p_job_id VARCHAR,
    p_hourly_rate DECIMAL,
    p_escrow_amount DECIMAL
) RETURNS UUID AS $$
DECLARE
    v_session_id UUID;
    v_escrow_wallet_id UUID;
    v_user_wallet_id UUID;
    v_gpu_model VARCHAR;
    v_vram_total BIGINT;
BEGIN
    -- Get GPU details
    SELECT model_name, vram_total_mb INTO v_gpu_model, v_vram_total
    FROM gpu_capabilities
    WHERE id = p_gpu_capability_id;
    
    -- Get user's wallet
    SELECT id INTO v_user_wallet_id
    FROM wallets
    WHERE user_id = p_user_id AND wallet_type = 'user' AND is_active = true
    LIMIT 1;
    
    -- Create escrow wallet for this session
    INSERT INTO wallets (user_id, wallet_type, solana_address, balance, is_active)
    VALUES (p_user_id, 'escrow', 'ESCROW_' || gen_random_uuid()::text, 0, true)
    RETURNING id INTO v_escrow_wallet_id;
    
    -- Lock funds in user wallet
    IF NOT lock_wallet_funds(v_user_wallet_id, p_escrow_amount) THEN
        RAISE EXCEPTION 'Insufficient funds to start rental session';
    END IF;
    
    -- Transfer to escrow
    UPDATE wallets SET balance = balance + p_escrow_amount WHERE id = v_escrow_wallet_id;
    UPDATE wallets SET balance = balance - p_escrow_amount WHERE id = v_user_wallet_id;
    
    -- Create rental session
    INSERT INTO rental_sessions (
        user_id, provider_id, job_id, status,
        gpu_model, allocated_vram_mb, total_vram_mb, vram_percentage,
        hourly_rate, vram_rate, power_rate,
        estimated_power_w, started_at, last_billed_at,
        escrow_wallet_id, escrow_amount
    ) VALUES (
        p_user_id, p_provider_id, p_job_id, 'active',
        v_gpu_model, v_vram_total, v_vram_total, 100,
        p_hourly_rate, 0, 0,
        300, NOW(), NOW(),
        v_escrow_wallet_id, p_escrow_amount
    ) RETURNING id INTO v_session_id;
    
    -- Record escrow transaction
    INSERT INTO escrow_transactions (session_id, wallet_id, transaction_type, amount, status)
    VALUES (v_session_id, v_escrow_wallet_id, 'lock', p_escrow_amount, 'completed');
    
    -- Update GPU status
    UPDATE gpu_capabilities SET is_available = false, status = 'busy', current_job_id = v_session_id::text
    WHERE id = p_gpu_capability_id;
    
    RETURN v_session_id;
END;
$$ LANGUAGE plpgsql;

-- Function to process minute-based billing
CREATE OR REPLACE FUNCTION process_rental_billing(p_session_id UUID)
RETURNS DECIMAL AS $$
DECLARE
    v_session RECORD;
    v_minutes_elapsed INTEGER;
    v_cost_per_minute DECIMAL;
    v_period_cost DECIMAL;
    v_platform_fee DECIMAL;
    v_provider_earnings DECIMAL;
BEGIN
    -- Get session details
    SELECT * INTO v_session
    FROM rental_sessions
    WHERE id = p_session_id AND status = 'active'
    FOR UPDATE;
    
    IF NOT FOUND THEN
        RETURN 0;
    END IF;
    
    -- Calculate minutes since last billing
    v_minutes_elapsed := EXTRACT(EPOCH FROM (NOW() - v_session.last_billed_at)) / 60;
    
    IF v_minutes_elapsed < 1 THEN
        RETURN 0;
    END IF;
    
    -- Calculate costs
    v_cost_per_minute := v_session.hourly_rate / 60;
    v_period_cost := v_cost_per_minute * v_minutes_elapsed;
    v_platform_fee := v_period_cost * (v_session.platform_fee_rate / 100);
    v_provider_earnings := v_period_cost - v_platform_fee;
    
    -- Check escrow balance
    IF v_period_cost > (SELECT balance FROM wallets WHERE id = v_session.escrow_wallet_id) THEN
        -- Insufficient funds, suspend session
        UPDATE rental_sessions SET status = 'suspended' WHERE id = p_session_id;
        RAISE NOTICE 'Session % suspended due to insufficient escrow funds', p_session_id;
        RETURN 0;
    END IF;
    
    -- Deduct from escrow
    UPDATE wallets SET balance = balance - v_period_cost WHERE id = v_session.escrow_wallet_id;
    
    -- Update session totals
    UPDATE rental_sessions SET
        total_cost = total_cost + v_period_cost,
        total_platform_fee = total_platform_fee + v_platform_fee,
        total_provider_earnings = total_provider_earnings + v_provider_earnings,
        last_billed_at = NOW()
    WHERE id = p_session_id;
    
    -- Record usage
    INSERT INTO usage_records (session_id, recorded_at, power_draw_w, period_minutes, period_cost)
    VALUES (p_session_id, NOW(), v_session.estimated_power_w, v_minutes_elapsed, v_period_cost);
    
    -- Record platform fee
    INSERT INTO platform_fees (session_id, fee_type, amount_dgpu, status)
    VALUES (p_session_id, 'rental', v_platform_fee, 'collected');
    
    RETURN v_period_cost;
END;
$$ LANGUAGE plpgsql;

-- Function to end rental session
CREATE OR REPLACE FUNCTION end_rental_session(p_session_id UUID)
RETURNS void AS $$
DECLARE
    v_session RECORD;
    v_provider_wallet_id UUID;
    v_remaining_escrow DECIMAL;
    v_user_wallet_id UUID;
BEGIN
    -- Get session details
    SELECT * INTO v_session
    FROM rental_sessions
    WHERE id = p_session_id
    FOR UPDATE;
    
    -- Process final billing
    PERFORM process_rental_billing(p_session_id);
    
    -- Get provider wallet
    SELECT id INTO v_provider_wallet_id
    FROM wallets
    WHERE user_id = v_session.provider_id::text AND wallet_type = 'provider' AND is_active = true
    LIMIT 1;
    
    -- Transfer provider earnings
    UPDATE wallets SET balance = balance + v_session.total_provider_earnings
    WHERE id = v_provider_wallet_id;
    
    -- Get remaining escrow balance
    SELECT balance INTO v_remaining_escrow
    FROM wallets WHERE id = v_session.escrow_wallet_id;
    
    -- Refund remaining escrow to user
    IF v_remaining_escrow > 0 THEN
        SELECT id INTO v_user_wallet_id
        FROM wallets
        WHERE user_id = v_session.user_id AND wallet_type = 'user' AND is_active = true
        LIMIT 1;
        
        UPDATE wallets SET balance = balance + v_remaining_escrow WHERE id = v_user_wallet_id;
        UPDATE wallets SET balance = 0 WHERE id = v_session.escrow_wallet_id;
        
        -- Record refund transaction
        INSERT INTO escrow_transactions (session_id, wallet_id, transaction_type, amount, status)
        VALUES (p_session_id, v_session.escrow_wallet_id, 'refund', v_remaining_escrow, 'completed');
    END IF;
    
    -- Update session status
    UPDATE rental_sessions SET
        status = 'completed',
        ended_at = NOW()
    WHERE id = p_session_id;
    
    -- Update provider earnings
    UPDATE providers SET
        total_earnings_dgpu = total_earnings_dgpu + v_session.total_provider_earnings,
        pending_payout_dgpu = pending_payout_dgpu + v_session.total_provider_earnings
    WHERE id = v_session.provider_id;
    
    -- Release GPU
    UPDATE gpu_capabilities SET
        is_available = true,
        status = 'available',
        current_job_id = NULL
    WHERE provider_id = v_session.provider_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- JOB MANAGEMENT
-- ============================================================================

-- Function to create and queue a job
CREATE OR REPLACE FUNCTION create_job(
    p_user_id VARCHAR,
    p_job_name VARCHAR,
    p_docker_image VARCHAR,
    p_required_vram_gb INTEGER,
    p_gpu_model VARCHAR DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_job_id UUID;
    v_job_id_str VARCHAR;
BEGIN
    -- Generate unique job ID
    v_job_id := gen_random_uuid();
    v_job_id_str := 'job_' || SUBSTRING(v_job_id::text, 1, 8);
    
    -- Create job
    INSERT INTO jobs (
        id, job_id, user_id, name, job_type, docker_image,
        required_gpu_model, required_vram_gb, status, queued_at
    ) VALUES (
        v_job_id, v_job_id_str, p_user_id, p_job_name, 'custom', p_docker_image,
        p_gpu_model, p_required_vram_gb, 'queued', NOW()
    );
    
    -- Log event
    INSERT INTO job_events (job_id, event_type, event_data, triggered_by)
    VALUES (v_job_id, 'job_created', jsonb_build_object('name', p_job_name), p_user_id);
    
    RETURN v_job_id;
END;
$$ LANGUAGE plpgsql;

-- Function to assign job to provider
CREATE OR REPLACE FUNCTION assign_job_to_provider(
    p_job_id UUID,
    p_provider_id UUID,
    p_gpu_capability_id UUID
) RETURNS BOOLEAN AS $$
DECLARE
    v_job RECORD;
    v_gpu RECORD;
BEGIN
    -- Get job details
    SELECT * INTO v_job FROM jobs WHERE id = p_job_id FOR UPDATE;
    
    -- Get GPU details
    SELECT * INTO v_gpu FROM gpu_capabilities WHERE id = p_gpu_capability_id FOR UPDATE;
    
    -- Validate GPU availability
    IF NOT v_gpu.is_available OR v_gpu.status != 'available' THEN
        RETURN FALSE;
    END IF;
    
    -- Validate VRAM requirement
    IF v_job.required_vram_gb * 1024 > v_gpu.vram_available_mb THEN
        RETURN FALSE;
    END IF;
    
    -- Update job
    UPDATE jobs SET
        provider_id = p_provider_id,
        status = 'starting',
        started_at = NOW()
    WHERE id = p_job_id;
    
    -- Log event
    INSERT INTO job_events (job_id, event_type, event_data, triggered_by)
    VALUES (p_job_id, 'job_assigned', jsonb_build_object('provider_id', p_provider_id), 'system');
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- PROVIDER MANAGEMENT
-- ============================================================================

-- Function to register a new provider
CREATE OR REPLACE FUNCTION register_provider(
    p_user_id VARCHAR,
    p_provider_name VARCHAR,
    p_country VARCHAR,
    p_contact_email VARCHAR
) RETURNS UUID AS $$
DECLARE
    v_provider_id UUID;
BEGIN
    -- Create provider
    INSERT INTO providers (
        user_id, provider_name, country, contact_email,
        status, verification_status
    ) VALUES (
        p_user_id, p_provider_name, p_country, p_contact_email,
        'pending', 'unverified'
    ) RETURNING id INTO v_provider_id;
    
    -- Assign provider role
    INSERT INTO user_roles (user_id, role, granted_by)
    VALUES (p_user_id, 'provider', 'system')
    ON CONFLICT (user_id, role) DO NOTHING;
    
    -- Create provider wallet
    INSERT INTO wallets (user_id, wallet_type, solana_address, is_active)
    VALUES (p_user_id, 'provider', 'PROVIDER_' || v_provider_id::text, true);
    
    RETURN v_provider_id;
END;
$$ LANGUAGE plpgsql;

-- Function to register GPU capability
CREATE OR REPLACE FUNCTION register_gpu_capability(
    p_provider_id UUID,
    p_gpu_id VARCHAR,
    p_model_name VARCHAR,
    p_manufacturer VARCHAR,
    p_vram_mb BIGINT,
    p_base_price DECIMAL
) RETURNS UUID AS $$
DECLARE
    v_gpu_cap_id UUID;
BEGIN
    INSERT INTO gpu_capabilities (
        provider_id, gpu_id, model_name, manufacturer,
        vram_total_mb, vram_available_mb, base_price_per_hour,
        status, is_available
    ) VALUES (
        p_provider_id, p_gpu_id, p_model_name, p_manufacturer,
        p_vram_mb, p_vram_mb, p_base_price,
        'offline', false
    ) RETURNING id INTO v_gpu_cap_id;
    
    -- Update provider GPU count
    UPDATE providers SET total_gpus = total_gpus + 1 WHERE id = p_provider_id;
    
    RETURN v_gpu_cap_id;
END;
$$ LANGUAGE plpgsql;

-- Function to update provider heartbeat
CREATE OR REPLACE FUNCTION update_provider_heartbeat(p_provider_id UUID)
RETURNS void AS $$
BEGIN
    UPDATE providers SET
        last_heartbeat_at = NOW(),
        is_online = true
    WHERE id = p_provider_id;
    
    -- Mark GPUs as online if provider is verified
    UPDATE gpu_capabilities SET
        status = 'available',
        is_available = true
    WHERE provider_id = p_provider_id
    AND provider_id IN (SELECT id FROM providers WHERE verification_status = 'verified');
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- WALLET OPERATIONS
-- ============================================================================

-- Function to create user wallet
CREATE OR REPLACE FUNCTION create_user_wallet(
    p_user_id VARCHAR,
    p_solana_address VARCHAR
) RETURNS UUID AS $$
DECLARE
    v_wallet_id UUID;
BEGIN
    INSERT INTO wallets (user_id, wallet_type, solana_address, is_active, is_external)
    VALUES (p_user_id, 'user', p_solana_address, true, true)
    RETURNING id INTO v_wallet_id;
    
    RETURN v_wallet_id;
END;
$$ LANGUAGE plpgsql;

-- Function to process deposit
CREATE OR REPLACE FUNCTION process_deposit(
    p_wallet_id UUID,
    p_amount DECIMAL,
    p_solana_signature VARCHAR
) RETURNS UUID AS $$
DECLARE
    v_tx_id UUID;
BEGIN
    -- Create blockchain transaction record
    INSERT INTO blockchain_transactions (
        to_wallet_id, transaction_type, status, amount,
        solana_signature, created_at
    ) VALUES (
        p_wallet_id, 'deposit', 'confirmed', p_amount,
        p_solana_signature, NOW()
    ) RETURNING id INTO v_tx_id;
    
    -- Update wallet balance
    UPDATE wallets SET
        balance = balance + p_amount,
        last_activity_at = NOW()
    WHERE id = p_wallet_id;
    
    RETURN v_tx_id;
END;
$$ LANGUAGE plpgsql;

-- Function to process withdrawal
CREATE OR REPLACE FUNCTION process_withdrawal(
    p_wallet_id UUID,
    p_amount DECIMAL,
    p_destination_address VARCHAR
) RETURNS UUID AS $$
DECLARE
    v_tx_id UUID;
    v_available_balance DECIMAL;
BEGIN
    -- Check available balance
    v_available_balance := wallet_available_balance(p_wallet_id);
    
    IF v_available_balance < p_amount THEN
        RAISE EXCEPTION 'Insufficient balance for withdrawal';
    END IF;
    
    -- Create blockchain transaction record
    INSERT INTO blockchain_transactions (
        from_wallet_id, transaction_type, status, amount, created_at
    ) VALUES (
        p_wallet_id, 'withdrawal', 'pending', p_amount, NOW()
    ) RETURNING id INTO v_tx_id;
    
    -- Lock funds
    PERFORM lock_wallet_funds(p_wallet_id, p_amount);
    
    RETURN v_tx_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- ANALYTICS FUNCTIONS
-- ============================================================================

-- Function to get platform statistics
CREATE OR REPLACE FUNCTION get_platform_stats(p_days INTEGER DEFAULT 30)
RETURNS TABLE (
    total_users BIGINT,
    total_providers BIGINT,
    total_gpus BIGINT,
    active_sessions BIGINT,
    total_revenue_dgpu DECIMAL,
    total_platform_fees DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        (SELECT COUNT(*) FROM users)::BIGINT,
        (SELECT COUNT(*) FROM providers WHERE status = 'active')::BIGINT,
        (SELECT COUNT(*) FROM gpu_capabilities)::BIGINT,
        (SELECT COUNT(*) FROM rental_sessions WHERE status = 'active')::BIGINT,
        (SELECT COALESCE(SUM(total_cost), 0) FROM rental_sessions 
         WHERE started_at >= CURRENT_DATE - p_days)::DECIMAL,
        (SELECT COALESCE(SUM(amount_dgpu), 0) FROM platform_fees 
         WHERE collected_at >= CURRENT_DATE - p_days)::DECIMAL;
END;
$$ LANGUAGE plpgsql;

