-- DanteGPU Platform - Additional Performance Indexes
-- Migration 007: Comprehensive indexing strategy for optimal query performance

-- ============================================================================
-- COMPOSITE INDEXES FOR COMMON QUERY PATTERNS
-- ============================================================================

-- Users table - composite indexes
CREATE INDEX IF NOT EXISTS idx_users_email_verified ON users(email, verified) WHERE verified = true;
CREATE INDEX IF NOT EXISTS idx_users_created_verified ON users(created_at DESC, verified);
CREATE INDEX IF NOT EXISTS idx_users_balance_verified ON users(balance DESC, verified) WHERE verified = true;

-- GPU instances - composite indexes for marketplace queries
CREATE INDEX IF NOT EXISTS idx_gpu_instances_status_price ON gpu_instances(status, price_per_hour) WHERE status = 'available';
CREATE INDEX IF NOT EXISTS idx_gpu_instances_provider_status ON gpu_instances(provider_id, status);
CREATE INDEX IF NOT EXISTS idx_gpu_instances_model_status ON gpu_instances(model_id, status) WHERE status = 'available';
CREATE INDEX IF NOT EXISTS idx_gpu_instances_location_status ON gpu_instances(location, status) WHERE status = 'available';

-- GPU rentals - composite indexes for user queries
CREATE INDEX IF NOT EXISTS idx_gpu_rentals_user_created ON gpu_rentals(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gpu_rentals_user_status_created ON gpu_rentals(user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gpu_rentals_instance_status ON gpu_rentals(gpu_instance_id, status);
CREATE INDEX IF NOT EXISTS idx_gpu_rentals_payment_status ON gpu_rentals(payment_status, status);

-- Payment transactions - composite indexes
CREATE INDEX IF NOT EXISTS idx_payment_tx_user_type_created ON payment_transactions(user_id, type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_tx_status_created ON payment_transactions(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_tx_rental_status ON payment_transactions(rental_id, status);
CREATE INDEX IF NOT EXISTS idx_payment_tx_external_id ON payment_transactions(external_transaction_id) WHERE external_transaction_id IS NOT NULL;

-- Notifications - composite indexes
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, created_at DESC) WHERE read = false;
CREATE INDEX IF NOT EXISTS idx_notifications_user_type_read ON notifications(user_id, type, read);

-- ============================================================================
-- PARTIAL INDEXES FOR FILTERED QUERIES
-- ============================================================================

-- Active sessions only
CREATE INDEX IF NOT EXISTS idx_user_sessions_active ON user_sessions(user_id, expires_at) WHERE expires_at > NOW();

-- Available GPUs only
CREATE INDEX IF NOT EXISTS idx_gpu_instances_available_price ON gpu_instances(price_per_hour, location) WHERE status = 'available';

-- Running rentals only
CREATE INDEX IF NOT EXISTS idx_gpu_rentals_running ON gpu_rentals(user_id, start_time) WHERE status = 'running';

-- Pending payments only
CREATE INDEX IF NOT EXISTS idx_payment_tx_pending ON payment_transactions(user_id, created_at) WHERE status = 'pending';

-- Unread notifications only
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications(user_id, created_at DESC) WHERE read = false;

-- ============================================================================
-- BLOCKCHAIN AND WALLET INDEXES
-- ============================================================================

-- Wallets - composite indexes
CREATE INDEX IF NOT EXISTS idx_wallets_user_type_active ON wallets(user_id, wallet_type, is_active);
CREATE INDEX IF NOT EXISTS idx_wallets_balance ON wallets(balance DESC) WHERE is_active = true;

-- Blockchain transactions - composite indexes
CREATE INDEX IF NOT EXISTS idx_blockchain_tx_from_status_created ON blockchain_transactions(from_wallet_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_blockchain_tx_to_status_created ON blockchain_transactions(to_wallet_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_blockchain_tx_type_status ON blockchain_transactions(transaction_type, status);
CREATE INDEX IF NOT EXISTS idx_blockchain_tx_pending ON blockchain_transactions(created_at) WHERE status = 'pending';

-- Rental sessions - composite indexes
CREATE INDEX IF NOT EXISTS idx_rental_sessions_user_status ON rental_sessions(user_id, status);
CREATE INDEX IF NOT EXISTS idx_rental_sessions_provider_status ON rental_sessions(provider_id, status);
CREATE INDEX IF NOT EXISTS idx_rental_sessions_active ON rental_sessions(started_at) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_rental_sessions_user_started ON rental_sessions(user_id, started_at DESC);

-- Usage records - time-series optimization
CREATE INDEX IF NOT EXISTS idx_usage_records_session_time_desc ON usage_records(session_id, recorded_at DESC);

-- ============================================================================
-- JOB EXECUTION INDEXES
-- ============================================================================

-- Jobs - composite indexes for queue management
CREATE INDEX IF NOT EXISTS idx_jobs_user_status_created ON jobs(user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_provider_status_created ON jobs(provider_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_status_priority_queued ON jobs(status, priority DESC, queued_at ASC);
CREATE INDEX IF NOT EXISTS idx_jobs_type_status ON jobs(job_type, status);

-- Jobs - partial indexes for active jobs
CREATE INDEX IF NOT EXISTS idx_jobs_running ON jobs(provider_id, started_at) WHERE status = 'running';
CREATE INDEX IF NOT EXISTS idx_jobs_queued_priority ON jobs(priority DESC, queued_at ASC) WHERE status = 'queued';

-- Job logs - time-series optimization
CREATE INDEX IF NOT EXISTS idx_job_logs_job_time_desc ON job_logs(job_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_job_logs_job_stream_time ON job_logs(job_id, stream, timestamp DESC);

-- Job metrics - time-series optimization
CREATE INDEX IF NOT EXISTS idx_job_metrics_job_time_desc ON job_metrics(job_id, recorded_at DESC);

-- Job files - composite indexes
CREATE INDEX IF NOT EXISTS idx_job_files_job_type_uploaded ON job_files(job_id, file_type, uploaded_at DESC);

-- ============================================================================
-- AUTHENTICATION AND SECURITY INDEXES
-- ============================================================================

-- API keys - composite indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_user_active ON api_keys(user_id, is_active) WHERE is_revoked = false;
CREATE INDEX IF NOT EXISTS idx_api_keys_active_expires ON api_keys(expires_at) WHERE is_active = true AND is_revoked = false;

-- Audit logs - composite indexes for querying
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_action_created ON audit_logs(user_id, action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_created ON audit_logs(resource_type, resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_status ON audit_logs(action, status, created_at DESC);

-- Security events - composite indexes
CREATE INDEX IF NOT EXISTS idx_security_events_type_severity ON security_events(event_type, severity, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_security_events_user_created ON security_events(user_id, created_at DESC) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_security_events_ip_created ON security_events(ip_address, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_security_events_unresolved ON security_events(severity, created_at) WHERE is_resolved = false;

-- Login attempts - composite indexes
CREATE INDEX IF NOT EXISTS idx_login_attempts_email_created ON login_attempts(email, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_login_attempts_ip_created ON login_attempts(ip_address, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_login_attempts_failed ON login_attempts(email, created_at DESC) WHERE success = false;

-- Active sessions - composite indexes
CREATE INDEX IF NOT EXISTS idx_active_sessions_user_active ON active_sessions(user_id, last_activity_at DESC) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_active_sessions_expires ON active_sessions(expires_at) WHERE is_active = true;

-- ============================================================================
-- PROVIDER AND GPU REGISTRY INDEXES
-- ============================================================================

-- Providers - composite indexes for marketplace
CREATE INDEX IF NOT EXISTS idx_providers_status_rating ON providers(status, rating DESC) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_providers_verification_rating ON providers(verification_status, rating DESC);
CREATE INDEX IF NOT EXISTS idx_providers_online_rating ON providers(is_online, rating DESC) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_providers_country_rating ON providers(country, rating DESC) WHERE status = 'active';

-- GPU capabilities - composite indexes for search
CREATE INDEX IF NOT EXISTS idx_gpu_cap_provider_available ON gpu_capabilities(provider_id, is_available);
CREATE INDEX IF NOT EXISTS idx_gpu_cap_model_available_price ON gpu_capabilities(model_name, is_available, base_price_per_hour) WHERE is_available = true;
CREATE INDEX IF NOT EXISTS idx_gpu_cap_manufacturer_available ON gpu_capabilities(manufacturer, is_available) WHERE is_available = true;
CREATE INDEX IF NOT EXISTS idx_gpu_cap_vram_available ON gpu_capabilities(vram_total_mb DESC, is_available) WHERE is_available = true;
CREATE INDEX IF NOT EXISTS idx_gpu_cap_price_available ON gpu_capabilities(base_price_per_hour, is_available) WHERE is_available = true;

-- GPU reservations - composite indexes
CREATE INDEX IF NOT EXISTS idx_gpu_res_gpu_time ON gpu_reservations(gpu_capability_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_gpu_res_user_status_created ON gpu_reservations(user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gpu_res_active ON gpu_reservations(gpu_capability_id, start_time, end_time) WHERE status IN ('confirmed', 'active');

-- Provider reviews - composite indexes
CREATE INDEX IF NOT EXISTS idx_provider_reviews_provider_approved ON provider_reviews(provider_id, created_at DESC) WHERE moderation_status = 'approved';
CREATE INDEX IF NOT EXISTS idx_provider_reviews_user_created ON provider_reviews(user_id, created_at DESC);

-- ============================================================================
-- JSONB INDEXES FOR METADATA QUERIES
-- ============================================================================

-- GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_gpu_models_features_gin ON gpu_models USING GIN (features);
CREATE INDEX IF NOT EXISTS idx_gpu_models_benchmarks_gin ON gpu_models USING GIN (benchmarks);
CREATE INDEX IF NOT EXISTS idx_gpu_instances_specs_gin ON gpu_instances USING GIN (specs);
CREATE INDEX IF NOT EXISTS idx_payment_tx_metadata_gin ON payment_transactions USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_gpu_metrics_data_gin ON gpu_metrics USING GIN (metrics_data);
CREATE INDEX IF NOT EXISTS idx_blockchain_tx_metadata_gin ON blockchain_transactions USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_jobs_environment_gin ON jobs USING GIN (environment_vars);
CREATE INDEX IF NOT EXISTS idx_jobs_metadata_gin ON jobs USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_gpu_cap_metadata_gin ON gpu_capabilities USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_providers_metadata_gin ON providers USING GIN (metadata);

-- ============================================================================
-- FULL TEXT SEARCH INDEXES
-- ============================================================================

-- Add tsvector columns for full-text search
ALTER TABLE gpu_models ADD COLUMN IF NOT EXISTS search_vector tsvector;
ALTER TABLE providers ADD COLUMN IF NOT EXISTS search_vector tsvector;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Create GIN indexes for full-text search
CREATE INDEX IF NOT EXISTS idx_gpu_models_search ON gpu_models USING GIN (search_vector);
CREATE INDEX IF NOT EXISTS idx_providers_search ON providers USING GIN (search_vector);
CREATE INDEX IF NOT EXISTS idx_jobs_search ON jobs USING GIN (search_vector);

-- Create triggers to update search vectors
CREATE OR REPLACE FUNCTION update_gpu_models_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.manufacturer, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.architecture, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trig_update_gpu_models_search_vector
    BEFORE INSERT OR UPDATE ON gpu_models
    FOR EACH ROW EXECUTE FUNCTION update_gpu_models_search_vector();

CREATE OR REPLACE FUNCTION update_providers_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.provider_name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.display_name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.country, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.city, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trig_update_providers_search_vector
    BEFORE INSERT OR UPDATE ON providers
    FOR EACH ROW EXECUTE FUNCTION update_providers_search_vector();

CREATE OR REPLACE FUNCTION update_jobs_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.job_id, '')), 'A');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trig_update_jobs_search_vector
    BEFORE INSERT OR UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION update_jobs_search_vector();

-- ============================================================================
-- COVERING INDEXES FOR INDEX-ONLY SCANS
-- ============================================================================

-- Covering indexes to avoid table lookups
CREATE INDEX IF NOT EXISTS idx_gpu_instances_marketplace_covering ON gpu_instances(status, price_per_hour, location) 
    INCLUDE (model_id, provider_id) WHERE status = 'available';

CREATE INDEX IF NOT EXISTS idx_gpu_rentals_user_list_covering ON gpu_rentals(user_id, created_at DESC) 
    INCLUDE (gpu_instance_id, status, total_cost, payment_status);

CREATE INDEX IF NOT EXISTS idx_jobs_user_list_covering ON jobs(user_id, created_at DESC) 
    INCLUDE (job_id, name, status, started_at, completed_at);

-- ============================================================================
-- STATISTICS AND MAINTENANCE
-- ============================================================================

-- Update table statistics for better query planning
ANALYZE users;
ANALYZE gpu_providers;
ANALYZE gpu_models;
ANALYZE gpu_instances;
ANALYZE gpu_rentals;
ANALYZE payment_transactions;
ANALYZE gpu_metrics;
ANALYZE wallets;
ANALYZE blockchain_transactions;
ANALYZE rental_sessions;
ANALYZE jobs;
ANALYZE providers;
ANALYZE gpu_capabilities;

-- Set statistics targets for frequently queried columns
ALTER TABLE gpu_instances ALTER COLUMN status SET STATISTICS 1000;
ALTER TABLE gpu_rentals ALTER COLUMN status SET STATISTICS 1000;
ALTER TABLE jobs ALTER COLUMN status SET STATISTICS 1000;
ALTER TABLE blockchain_transactions ALTER COLUMN status SET STATISTICS 1000;
ALTER TABLE gpu_capabilities ALTER COLUMN is_available SET STATISTICS 1000;

