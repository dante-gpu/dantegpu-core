-- DanteGPU Platform - Provider and GPU Registry Tables
-- Migration 006: Enhanced provider management, GPU capabilities, and marketplace

-- Enhanced providers table
CREATE TABLE IF NOT EXISTS providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL UNIQUE,
    
    -- Basic information
    provider_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    website VARCHAR(500),
    logo_url VARCHAR(500),
    
    -- Contact information
    contact_email VARCHAR(255) NOT NULL,
    contact_phone VARCHAR(50),
    support_email VARCHAR(255),
    
    -- Location
    country VARCHAR(100) NOT NULL,
    region VARCHAR(100),
    city VARCHAR(100),
    datacenter_name VARCHAR(255),
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Status and verification
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'active', 'inactive', 'suspended', 'banned'
    )),
    verification_status VARCHAR(50) DEFAULT 'unverified' CHECK (verification_status IN (
        'unverified', 'pending', 'verified', 'rejected'
    )),
    verification_level INTEGER DEFAULT 0 CHECK (verification_level >= 0 AND verification_level <= 5),
    
    -- Business information
    business_type VARCHAR(50) CHECK (business_type IN ('individual', 'company', 'datacenter')),
    tax_id VARCHAR(100),
    business_license VARCHAR(255),
    
    -- Performance metrics
    total_gpus INTEGER DEFAULT 0,
    active_gpus INTEGER DEFAULT 0,
    total_jobs_completed INTEGER DEFAULT 0,
    total_jobs_failed INTEGER DEFAULT 0,
    total_uptime_hours DECIMAL(12,2) DEFAULT 0,
    average_response_time_ms INTEGER DEFAULT 0,
    
    -- Ratings and reviews
    rating DECIMAL(3,2) DEFAULT 0 CHECK (rating >= 0 AND rating <= 5),
    total_reviews INTEGER DEFAULT 0,
    
    -- Earnings
    total_earnings_dgpu DECIMAL(20,9) DEFAULT 0,
    total_earnings_usd DECIMAL(20,8) DEFAULT 0,
    pending_payout_dgpu DECIMAL(20,9) DEFAULT 0,
    
    -- Configuration
    commission_rate DECIMAL(5,4) DEFAULT 0.05, -- Platform fee (5%)
    min_rental_minutes INTEGER DEFAULT 60,
    max_rental_minutes INTEGER DEFAULT 43200, -- 30 days
    auto_accept_jobs BOOLEAN DEFAULT false,
    instant_availability BOOLEAN DEFAULT true,
    
    -- API configuration
    api_endpoint VARCHAR(500),
    api_key_hash VARCHAR(255),
    webhook_url VARCHAR(500),
    webhook_secret_hash VARCHAR(255),
    
    -- Heartbeat and monitoring
    last_heartbeat_at TIMESTAMP WITH TIME ZONE,
    heartbeat_interval_seconds INTEGER DEFAULT 60,
    is_online BOOLEAN DEFAULT false,
    
    -- Metadata
    supported_frameworks TEXT[] DEFAULT '{}',
    certifications TEXT[] DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    verified_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT positive_gpus CHECK (active_gpus >= 0 AND active_gpus <= total_gpus),
    CONSTRAINT positive_jobs CHECK (total_jobs_completed >= 0 AND total_jobs_failed >= 0),
    CONSTRAINT positive_earnings CHECK (total_earnings_dgpu >= 0 AND pending_payout_dgpu >= 0)
);

-- GPU capabilities and specifications
CREATE TABLE IF NOT EXISTS gpu_capabilities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    
    -- GPU identification
    gpu_id VARCHAR(255) NOT NULL, -- Provider's internal GPU ID
    gpu_uuid VARCHAR(255), -- Hardware UUID
    
    -- GPU model information
    model_name VARCHAR(255) NOT NULL,
    manufacturer VARCHAR(100) NOT NULL CHECK (manufacturer IN ('NVIDIA', 'AMD', 'Intel', 'Apple')),
    architecture VARCHAR(100),
    compute_capability VARCHAR(20), -- CUDA compute capability
    
    -- Memory specifications
    vram_total_mb BIGINT NOT NULL,
    vram_available_mb BIGINT NOT NULL,
    memory_type VARCHAR(50), -- GDDR6, GDDR6X, HBM2, HBM3
    memory_bandwidth_gbps DECIMAL(10,2),
    memory_bus_width INTEGER,
    
    -- Compute specifications
    cuda_cores INTEGER,
    tensor_cores INTEGER,
    rt_cores INTEGER,
    stream_processors INTEGER,
    compute_units INTEGER,
    
    -- Clock speeds
    base_clock_mhz INTEGER,
    boost_clock_mhz INTEGER,
    memory_clock_mhz INTEGER,
    
    -- Power and thermal
    tdp_watts INTEGER,
    max_power_limit_watts INTEGER,
    max_temperature_c INTEGER DEFAULT 90,
    
    -- Interface
    pcie_generation VARCHAR(20),
    pcie_lanes INTEGER,
    
    -- Features and capabilities
    supports_cuda BOOLEAN DEFAULT false,
    supports_opencl BOOLEAN DEFAULT false,
    supports_vulkan BOOLEAN DEFAULT false,
    supports_directx VARCHAR(20),
    supports_ray_tracing BOOLEAN DEFAULT false,
    supports_tensor_ops BOOLEAN DEFAULT false,
    supports_fp16 BOOLEAN DEFAULT false,
    supports_fp64 BOOLEAN DEFAULT false,
    supports_int8 BOOLEAN DEFAULT false,
    
    -- Virtualization
    supports_mig BOOLEAN DEFAULT false, -- Multi-Instance GPU
    mig_profiles TEXT[] DEFAULT '{}',
    supports_vgpu BOOLEAN DEFAULT false,
    
    -- Benchmarks
    benchmark_fp32_tflops DECIMAL(10,2),
    benchmark_fp16_tflops DECIMAL(10,2),
    benchmark_int8_tops DECIMAL(10,2),
    benchmark_memory_bandwidth_gbps DECIMAL(10,2),
    
    -- Pricing
    base_price_per_hour DECIMAL(10,4) NOT NULL,
    vram_price_per_gb_hour DECIMAL(10,4) DEFAULT 0,
    power_price_per_kwh DECIMAL(10,4) DEFAULT 0,
    
    -- Availability
    status VARCHAR(50) NOT NULL DEFAULT 'offline' CHECK (status IN (
        'available', 'busy', 'maintenance', 'offline', 'error'
    )),
    is_available BOOLEAN DEFAULT false,
    current_job_id UUID,
    
    -- System information
    driver_version VARCHAR(50),
    cuda_version VARCHAR(50),
    os_type VARCHAR(50),
    os_version VARCHAR(100),
    
    -- Host system specs
    cpu_model VARCHAR(255),
    cpu_cores INTEGER,
    ram_total_gb INTEGER,
    storage_total_gb INTEGER,
    storage_type VARCHAR(50), -- SSD, NVMe, HDD
    network_speed_mbps INTEGER,
    
    -- Monitoring
    last_health_check_at TIMESTAMP WITH TIME ZONE,
    health_status VARCHAR(50) DEFAULT 'unknown' CHECK (health_status IN (
        'healthy', 'degraded', 'unhealthy', 'unknown'
    )),
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(provider_id, gpu_id),
    CONSTRAINT positive_vram CHECK (vram_available_mb >= 0 AND vram_available_mb <= vram_total_mb),
    CONSTRAINT positive_price CHECK (base_price_per_hour > 0)
);

-- GPU availability schedule
CREATE TABLE IF NOT EXISTS gpu_availability_schedule (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    gpu_capability_id UUID NOT NULL REFERENCES gpu_capabilities(id) ON DELETE CASCADE,
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6), -- 0 = Sunday
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_available BOOLEAN DEFAULT true,
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- GPU reservations
CREATE TABLE IF NOT EXISTS gpu_reservations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    gpu_capability_id UUID NOT NULL REFERENCES gpu_capabilities(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    
    -- Reservation details
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_minutes INTEGER NOT NULL,
    
    -- Pricing
    hourly_rate DECIMAL(10,4) NOT NULL,
    total_cost DECIMAL(20,9) NOT NULL,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'confirmed', 'active', 'completed', 'cancelled', 'expired'
    )),
    
    -- Payment
    payment_status VARCHAR(50) DEFAULT 'pending' CHECK (payment_status IN (
        'pending', 'paid', 'refunded', 'failed'
    )),
    escrow_tx_id UUID,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    confirmed_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT valid_time_range CHECK (end_time > start_time),
    CONSTRAINT positive_duration CHECK (duration_minutes > 0),
    CONSTRAINT positive_cost CHECK (total_cost > 0)
);

-- Provider reviews and ratings
CREATE TABLE IF NOT EXISTS provider_reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    job_id UUID REFERENCES jobs(id) ON DELETE SET NULL,
    
    -- Rating
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    
    -- Review details
    title VARCHAR(255),
    comment TEXT,
    
    -- Specific ratings
    performance_rating INTEGER CHECK (performance_rating >= 1 AND performance_rating <= 5),
    reliability_rating INTEGER CHECK (reliability_rating >= 1 AND reliability_rating <= 5),
    support_rating INTEGER CHECK (support_rating >= 1 AND support_rating <= 5),
    value_rating INTEGER CHECK (value_rating >= 1 AND value_rating <= 5),
    
    -- Moderation
    is_verified BOOLEAN DEFAULT false,
    is_flagged BOOLEAN DEFAULT false,
    moderation_status VARCHAR(50) DEFAULT 'pending' CHECK (moderation_status IN (
        'pending', 'approved', 'rejected', 'hidden'
    )),
    
    -- Response
    provider_response TEXT,
    provider_responded_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, job_id)
);

-- Provider payout requests
CREATE TABLE IF NOT EXISTS provider_payout_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    
    -- Amount
    amount_dgpu DECIMAL(20,9) NOT NULL,
    amount_usd DECIMAL(20,8),
    
    -- Destination
    destination_wallet_address VARCHAR(44) NOT NULL,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'approved', 'processing', 'completed', 'rejected', 'failed'
    )),
    
    -- Processing
    approved_by VARCHAR(255),
    approved_at TIMESTAMP WITH TIME ZONE,
    processed_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Transaction
    blockchain_tx_id UUID REFERENCES blockchain_transactions(id),
    
    -- Rejection
    rejection_reason TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT positive_amount CHECK (amount_dgpu > 0)
);

-- Create indexes
CREATE INDEX idx_providers_user_id ON providers(user_id);
CREATE INDEX idx_providers_status ON providers(status);
CREATE INDEX idx_providers_verification ON providers(verification_status);
CREATE INDEX idx_providers_online ON providers(is_online);
CREATE INDEX idx_providers_rating ON providers(rating DESC);

CREATE INDEX idx_gpu_capabilities_provider ON gpu_capabilities(provider_id);
CREATE INDEX idx_gpu_capabilities_status ON gpu_capabilities(status);
CREATE INDEX idx_gpu_capabilities_available ON gpu_capabilities(is_available);
CREATE INDEX idx_gpu_capabilities_model ON gpu_capabilities(model_name);
CREATE INDEX idx_gpu_capabilities_manufacturer ON gpu_capabilities(manufacturer);
CREATE INDEX idx_gpu_capabilities_price ON gpu_capabilities(base_price_per_hour);

CREATE INDEX idx_gpu_availability_gpu ON gpu_availability_schedule(gpu_capability_id);
CREATE INDEX idx_gpu_availability_day ON gpu_availability_schedule(day_of_week);

CREATE INDEX idx_gpu_reservations_gpu ON gpu_reservations(gpu_capability_id);
CREATE INDEX idx_gpu_reservations_user ON gpu_reservations(user_id);
CREATE INDEX idx_gpu_reservations_status ON gpu_reservations(status);
CREATE INDEX idx_gpu_reservations_time ON gpu_reservations(start_time, end_time);

CREATE INDEX idx_provider_reviews_provider ON provider_reviews(provider_id);
CREATE INDEX idx_provider_reviews_user ON provider_reviews(user_id);
CREATE INDEX idx_provider_reviews_rating ON provider_reviews(rating);
CREATE INDEX idx_provider_reviews_status ON provider_reviews(moderation_status);

CREATE INDEX idx_provider_payout_requests_provider ON provider_payout_requests(provider_id);
CREATE INDEX idx_provider_payout_requests_status ON provider_payout_requests(status);

-- Create triggers
CREATE TRIGGER update_providers_updated_at 
    BEFORE UPDATE ON providers 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_gpu_capabilities_updated_at 
    BEFORE UPDATE ON gpu_capabilities 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_gpu_reservations_updated_at 
    BEFORE UPDATE ON gpu_reservations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_provider_reviews_updated_at 
    BEFORE UPDATE ON provider_reviews 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_provider_payout_requests_updated_at 
    BEFORE UPDATE ON provider_payout_requests 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update provider rating
CREATE OR REPLACE FUNCTION update_provider_rating(p_provider_id UUID)
RETURNS VOID AS $$
DECLARE
    avg_rating DECIMAL(3,2);
    review_count INTEGER;
BEGIN
    SELECT 
        COALESCE(AVG(rating), 0),
        COUNT(*)
    INTO avg_rating, review_count
    FROM provider_reviews
    WHERE provider_id = p_provider_id
    AND moderation_status = 'approved';
    
    UPDATE providers
    SET rating = avg_rating,
        total_reviews = review_count,
        updated_at = NOW()
    WHERE id = p_provider_id;
END;
$$ LANGUAGE plpgsql;

-- Function to check GPU availability
CREATE OR REPLACE FUNCTION is_gpu_available(
    p_gpu_id UUID,
    p_start_time TIMESTAMP WITH TIME ZONE,
    p_end_time TIMESTAMP WITH TIME ZONE
) RETURNS BOOLEAN AS $$
DECLARE
    conflicts INTEGER;
BEGIN
    SELECT COUNT(*) INTO conflicts
    FROM gpu_reservations
    WHERE gpu_capability_id = p_gpu_id
    AND status IN ('confirmed', 'active')
    AND (
        (start_time <= p_start_time AND end_time > p_start_time)
        OR (start_time < p_end_time AND end_time >= p_end_time)
        OR (start_time >= p_start_time AND end_time <= p_end_time)
    );
    
    RETURN conflicts = 0;
END;
$$ LANGUAGE plpgsql;

