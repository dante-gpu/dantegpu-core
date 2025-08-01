-- DanteGPU Platform Database Schema
-- Initial migration for core tables

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(500),
    balance DECIMAL(10,2) DEFAULT 0.00,
    verified BOOLEAN DEFAULT FALSE,
    phone VARCHAR(50),
    company VARCHAR(255),
    bio TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- GPU providers table
CREATE TABLE IF NOT EXISTS gpu_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    location VARCHAR(100) NOT NULL,
    contact_email VARCHAR(255),
    api_endpoint VARCHAR(500),
    api_key_hash VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- GPU models table
CREATE TABLE IF NOT EXISTS gpu_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    manufacturer VARCHAR(100) NOT NULL,
    architecture VARCHAR(100),
    memory_gb INTEGER NOT NULL,
    memory_type VARCHAR(50),
    memory_bandwidth_gbps DECIMAL(8,2),
    cuda_cores INTEGER,
    tensor_cores INTEGER,
    base_clock_mhz INTEGER,
    boost_clock_mhz INTEGER,
    power_consumption_w INTEGER,
    pcie_version VARCHAR(20),
    features JSONB,
    benchmarks JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- GPU instances table
CREATE TABLE IF NOT EXISTS gpu_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID REFERENCES gpu_providers(id) ON DELETE CASCADE,
    model_id UUID REFERENCES gpu_models(id) ON DELETE CASCADE,
    instance_id VARCHAR(255) NOT NULL, -- Provider's internal instance ID
    price_per_hour DECIMAL(8,4) NOT NULL,
    status VARCHAR(50) DEFAULT 'available', -- available, busy, maintenance, offline
    location VARCHAR(100),
    specs JSONB, -- Additional specifications
    last_health_check TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(provider_id, instance_id)
);

-- GPU rentals table
CREATE TABLE IF NOT EXISTS gpu_rentals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    gpu_instance_id UUID REFERENCES gpu_instances(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'pending', -- pending, running, paused, stopped, completed, failed
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    duration_hours DECIMAL(8,4),
    price_per_hour DECIMAL(8,4) NOT NULL,
    total_cost DECIMAL(10,2),
    payment_status VARCHAR(50) DEFAULT 'pending', -- pending, paid, failed, refunded
    connection_info JSONB, -- SSH, VNC, or other connection details
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Payment transactions table
CREATE TABLE IF NOT EXISTS payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    rental_id UUID REFERENCES gpu_rentals(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL, -- rental, topup, refund
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(50) DEFAULT 'pending', -- pending, completed, failed, cancelled
    payment_method VARCHAR(50), -- stripe, paypal, crypto
    external_transaction_id VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- GPU metrics table for monitoring
CREATE TABLE IF NOT EXISTS gpu_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gpu_instance_id UUID REFERENCES gpu_instances(id) ON DELETE CASCADE,
    rental_id UUID REFERENCES gpu_rentals(id) ON DELETE SET NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    gpu_utilization DECIMAL(5,2), -- Percentage
    memory_utilization DECIMAL(5,2), -- Percentage
    temperature_celsius INTEGER,
    power_usage_watts INTEGER,
    fan_speed_rpm INTEGER,
    metrics_data JSONB -- Additional metrics
);

-- User sessions table for authentication
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- rental_started, rental_ended, payment_success, etc.
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    read BOOLEAN DEFAULT FALSE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_gpu_instances_status ON gpu_instances(status);
CREATE INDEX IF NOT EXISTS idx_gpu_rentals_user_status ON gpu_rentals(user_id, status);
CREATE INDEX IF NOT EXISTS idx_gpu_rentals_status_time ON gpu_rentals(status, start_time);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_user ON payment_transactions(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_gpu_metrics_instance_time ON gpu_metrics(gpu_instance_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_gpu_metrics_rental_time ON gpu_metrics(rental_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_expires ON user_sessions(user_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_notifications_user_read_time ON notifications(user_id, read, created_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_gpu_providers_updated_at BEFORE UPDATE ON gpu_providers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_gpu_models_updated_at BEFORE UPDATE ON gpu_models FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_gpu_instances_updated_at BEFORE UPDATE ON gpu_instances FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_gpu_rentals_updated_at BEFORE UPDATE ON gpu_rentals FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_payment_transactions_updated_at BEFORE UPDATE ON payment_transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
