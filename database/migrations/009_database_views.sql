-- DanteGPU Platform - Database Views
-- Migration 009: Materialized views for reporting and analytics

-- ============================================================================
-- MARKETPLACE VIEWS
-- ============================================================================

-- Available GPUs with full details
CREATE OR REPLACE VIEW v_available_gpus AS
SELECT 
    gc.id as gpu_id,
    gc.gpu_id as internal_gpu_id,
    gc.model_name,
    gc.manufacturer,
    gc.vram_total_mb,
    gc.vram_available_mb,
    gc.base_price_per_hour,
    gc.cuda_cores,
    gc.tensor_cores,
    gc.supports_cuda,
    gc.supports_ray_tracing,
    p.id as provider_id,
    p.provider_name,
    p.country,
    p.region,
    p.city,
    p.rating as provider_rating,
    p.total_jobs_completed,
    p.verification_status,
    gc.last_health_check_at,
    gc.health_status
FROM gpu_capabilities gc
JOIN providers p ON gc.provider_id = p.id
WHERE gc.is_available = true
AND gc.status = 'available'
AND p.status = 'active'
AND p.is_online = true;

-- GPU marketplace with pricing and availability
CREATE MATERIALIZED VIEW mv_gpu_marketplace AS
SELECT 
    gc.id,
    gc.model_name,
    gc.manufacturer,
    gc.vram_total_mb / 1024 as vram_gb,
    gc.base_price_per_hour,
    gc.cuda_cores,
    gc.tensor_cores,
    gc.benchmark_fp32_tflops,
    gc.benchmark_fp16_tflops,
    p.provider_name,
    p.country,
    p.city,
    p.rating as provider_rating,
    p.total_jobs_completed as provider_jobs,
    COUNT(DISTINCT gr.id) as active_reservations,
    gc.is_available,
    gc.status,
    gc.last_health_check_at
FROM gpu_capabilities gc
JOIN providers p ON gc.provider_id = p.id
LEFT JOIN gpu_reservations gr ON gc.id = gr.gpu_capability_id 
    AND gr.status IN ('confirmed', 'active')
WHERE p.status = 'active'
GROUP BY gc.id, p.id;

CREATE UNIQUE INDEX idx_mv_gpu_marketplace_id ON mv_gpu_marketplace(id);
CREATE INDEX idx_mv_gpu_marketplace_available ON mv_gpu_marketplace(is_available, base_price_per_hour);
CREATE INDEX idx_mv_gpu_marketplace_model ON mv_gpu_marketplace(model_name);

-- ============================================================================
-- USER DASHBOARD VIEWS
-- ============================================================================

-- User rental summary
CREATE OR REPLACE VIEW v_user_rental_summary AS
SELECT 
    rs.user_id,
    COUNT(*) as total_rentals,
    COUNT(*) FILTER (WHERE rs.status = 'active') as active_rentals,
    COUNT(*) FILTER (WHERE rs.status = 'completed') as completed_rentals,
    SUM(rs.total_cost) as total_spent,
    SUM(rs.total_cost) FILTER (WHERE rs.status = 'active') as active_cost,
    AVG(EXTRACT(EPOCH FROM (rs.ended_at - rs.started_at))/3600) FILTER (WHERE rs.status = 'completed') as avg_rental_hours,
    MAX(rs.started_at) as last_rental_date
FROM rental_sessions rs
GROUP BY rs.user_id;

-- User job statistics
CREATE OR REPLACE VIEW v_user_job_stats AS
SELECT 
    j.user_id,
    COUNT(*) as total_jobs,
    COUNT(*) FILTER (WHERE j.status = 'running') as running_jobs,
    COUNT(*) FILTER (WHERE j.status = 'completed') as completed_jobs,
    COUNT(*) FILTER (WHERE j.status = 'failed') as failed_jobs,
    AVG(EXTRACT(EPOCH FROM (j.completed_at - j.started_at))/3600) FILTER (WHERE j.status = 'completed') as avg_job_hours,
    SUM(rs.total_cost) as total_job_cost
FROM jobs j
LEFT JOIN rental_sessions rs ON j.session_id = rs.id
GROUP BY j.user_id;

-- User wallet summary
CREATE OR REPLACE VIEW v_user_wallet_summary AS
SELECT 
    w.user_id,
    SUM(w.balance) as total_balance,
    SUM(w.locked_balance) as total_locked,
    SUM(w.balance - w.locked_balance) as available_balance,
    COUNT(*) as wallet_count,
    COUNT(*) FILTER (WHERE w.is_active = true) as active_wallets
FROM wallets w
GROUP BY w.user_id;

-- ============================================================================
-- PROVIDER DASHBOARD VIEWS
-- ============================================================================

-- Provider earnings summary
CREATE MATERIALIZED VIEW mv_provider_earnings AS
SELECT 
    p.id as provider_id,
    p.provider_name,
    COUNT(DISTINCT gc.id) as total_gpus,
    COUNT(DISTINCT gc.id) FILTER (WHERE gc.is_available = true) as available_gpus,
    COUNT(DISTINCT rs.id) as total_sessions,
    COUNT(DISTINCT rs.id) FILTER (WHERE rs.status = 'active') as active_sessions,
    SUM(rs.total_provider_earnings) as total_earnings_dgpu,
    SUM(rs.total_provider_earnings) FILTER (WHERE rs.status = 'active') as pending_earnings_dgpu,
    SUM(rs.total_provider_earnings) FILTER (WHERE rs.status = 'completed') as completed_earnings_dgpu,
    p.pending_payout_dgpu,
    AVG(rs.total_provider_earnings) FILTER (WHERE rs.status = 'completed') as avg_session_earnings,
    SUM(EXTRACT(EPOCH FROM (COALESCE(rs.ended_at, NOW()) - rs.started_at))/3600) as total_rental_hours,
    p.rating,
    p.total_reviews
FROM providers p
LEFT JOIN gpu_capabilities gc ON p.id = gc.provider_id
LEFT JOIN rental_sessions rs ON p.id = rs.provider_id
GROUP BY p.id;

CREATE UNIQUE INDEX idx_mv_provider_earnings_id ON mv_provider_earnings(provider_id);

-- Provider GPU utilization
CREATE OR REPLACE VIEW v_provider_gpu_utilization AS
SELECT 
    p.id as provider_id,
    p.provider_name,
    gc.id as gpu_id,
    gc.model_name,
    gc.status,
    gc.is_available,
    rs.id as current_session_id,
    rs.user_id as current_user,
    rs.started_at as session_start,
    EXTRACT(EPOCH FROM (NOW() - rs.started_at))/3600 as hours_running,
    rs.total_cost as session_cost
FROM providers p
JOIN gpu_capabilities gc ON p.id = gc.provider_id
LEFT JOIN rental_sessions rs ON gc.id = rs.provider_id AND rs.status = 'active';

-- ============================================================================
-- ANALYTICS VIEWS
-- ============================================================================

-- Platform revenue summary
CREATE MATERIALIZED VIEW mv_platform_revenue AS
SELECT 
    DATE_TRUNC('day', rs.started_at) as date,
    COUNT(DISTINCT rs.id) as total_sessions,
    COUNT(DISTINCT rs.user_id) as unique_users,
    COUNT(DISTINCT rs.provider_id) as unique_providers,
    SUM(rs.total_cost) as total_revenue_dgpu,
    SUM(rs.total_platform_fee) as platform_fees_dgpu,
    SUM(rs.total_provider_earnings) as provider_earnings_dgpu,
    AVG(rs.total_cost) as avg_session_cost,
    SUM(EXTRACT(EPOCH FROM (COALESCE(rs.ended_at, NOW()) - rs.started_at))/3600) as total_hours
FROM rental_sessions rs
WHERE rs.started_at >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY DATE_TRUNC('day', rs.started_at);

CREATE UNIQUE INDEX idx_mv_platform_revenue_date ON mv_platform_revenue(date);

-- GPU model popularity
CREATE MATERIALIZED VIEW mv_gpu_model_stats AS
SELECT 
    gc.model_name,
    gc.manufacturer,
    COUNT(DISTINCT gc.id) as total_units,
    COUNT(DISTINCT gc.id) FILTER (WHERE gc.is_available = true) as available_units,
    AVG(gc.base_price_per_hour) as avg_price_per_hour,
    MIN(gc.base_price_per_hour) as min_price,
    MAX(gc.base_price_per_hour) as max_price,
    COUNT(DISTINCT rs.id) as total_rentals,
    SUM(rs.total_cost) as total_revenue,
    AVG(EXTRACT(EPOCH FROM (rs.ended_at - rs.started_at))/3600) FILTER (WHERE rs.status = 'completed') as avg_rental_hours,
    AVG(pr.rating) as avg_provider_rating
FROM gpu_capabilities gc
LEFT JOIN rental_sessions rs ON gc.id = rs.provider_id
LEFT JOIN providers pr ON gc.provider_id = pr.id
GROUP BY gc.model_name, gc.manufacturer;

CREATE UNIQUE INDEX idx_mv_gpu_model_stats_model ON mv_gpu_model_stats(model_name, manufacturer);

-- Job execution statistics
CREATE MATERIALIZED VIEW mv_job_execution_stats AS
SELECT 
    DATE_TRUNC('day', j.created_at) as date,
    j.job_type,
    COUNT(*) as total_jobs,
    COUNT(*) FILTER (WHERE j.status = 'completed') as completed_jobs,
    COUNT(*) FILTER (WHERE j.status = 'failed') as failed_jobs,
    COUNT(*) FILTER (WHERE j.status = 'running') as running_jobs,
    AVG(EXTRACT(EPOCH FROM (j.completed_at - j.started_at))/60) FILTER (WHERE j.status = 'completed') as avg_duration_minutes,
    AVG(j.max_gpu_utilization) FILTER (WHERE j.status = 'completed') as avg_gpu_utilization,
    AVG(j.max_vram_usage_mb) FILTER (WHERE j.status = 'completed') as avg_vram_usage_mb
FROM jobs j
WHERE j.created_at >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY DATE_TRUNC('day', j.created_at), j.job_type;

CREATE INDEX idx_mv_job_execution_stats_date ON mv_job_execution_stats(date);
CREATE INDEX idx_mv_job_execution_stats_type ON mv_job_execution_stats(job_type);

-- ============================================================================
-- BLOCKCHAIN VIEWS
-- ============================================================================

-- Wallet transaction summary
CREATE OR REPLACE VIEW v_wallet_transactions AS
SELECT 
    bt.id,
    bt.transaction_type,
    bt.status,
    bt.amount,
    bt.fee,
    bt.solana_signature,
    bt.created_at,
    bt.confirmed_at,
    fw.user_id as from_user,
    fw.solana_address as from_address,
    tw.user_id as to_user,
    tw.solana_address as to_address,
    rs.id as session_id,
    j.job_id
FROM blockchain_transactions bt
LEFT JOIN wallets fw ON bt.from_wallet_id = fw.id
LEFT JOIN wallets tw ON bt.to_wallet_id = tw.id
LEFT JOIN rental_sessions rs ON bt.session_id = rs.id
LEFT JOIN jobs j ON bt.job_id = j.job_id;

-- Platform fee collection summary
CREATE MATERIALIZED VIEW mv_platform_fee_summary AS
SELECT 
    DATE_TRUNC('day', pf.collected_at) as date,
    pf.fee_type,
    COUNT(*) as transaction_count,
    SUM(pf.amount_dgpu) as total_fees_dgpu,
    SUM(pf.amount_usd) as total_fees_usd,
    COUNT(*) FILTER (WHERE pf.status = 'collected') as collected_count,
    COUNT(*) FILTER (WHERE pf.status = 'distributed') as distributed_count
FROM platform_fees pf
WHERE pf.collected_at >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY DATE_TRUNC('day', pf.collected_at), pf.fee_type;

CREATE INDEX idx_mv_platform_fee_summary_date ON mv_platform_fee_summary(date);

-- ============================================================================
-- SECURITY VIEWS
-- ============================================================================

-- Recent security events
CREATE OR REPLACE VIEW v_recent_security_events AS
SELECT 
    se.id,
    se.event_type,
    se.severity,
    se.user_id,
    se.ip_address,
    se.description,
    se.action_taken,
    se.is_resolved,
    se.created_at,
    u.email as user_email
FROM security_events se
LEFT JOIN users u ON se.user_id::uuid = u.id
WHERE se.created_at >= NOW() - INTERVAL '7 days'
ORDER BY se.created_at DESC;

-- Failed login attempts summary
CREATE MATERIALIZED VIEW mv_failed_login_summary AS
SELECT 
    DATE_TRUNC('hour', la.created_at) as hour,
    la.email,
    la.ip_address,
    COUNT(*) as attempt_count,
    MAX(la.created_at) as last_attempt
FROM login_attempts la
WHERE la.success = false
AND la.created_at >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY DATE_TRUNC('hour', la.created_at), la.email, la.ip_address
HAVING COUNT(*) >= 3;

CREATE INDEX idx_mv_failed_login_summary_hour ON mv_failed_login_summary(hour);
CREATE INDEX idx_mv_failed_login_summary_email ON mv_failed_login_summary(email);

-- ============================================================================
-- REFRESH FUNCTIONS FOR MATERIALIZED VIEWS
-- ============================================================================

-- Function to refresh all materialized views
CREATE OR REPLACE FUNCTION refresh_all_materialized_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_gpu_marketplace;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_provider_earnings;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_platform_revenue;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_gpu_model_stats;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_job_execution_stats;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_platform_fee_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_failed_login_summary;
    
    RAISE NOTICE 'All materialized views refreshed at %', NOW();
END;
$$ LANGUAGE plpgsql;

-- Function to refresh specific view
CREATE OR REPLACE FUNCTION refresh_materialized_view(view_name TEXT)
RETURNS void AS $$
BEGIN
    EXECUTE format('REFRESH MATERIALIZED VIEW CONCURRENTLY %I', view_name);
    RAISE NOTICE 'Materialized view % refreshed at %', view_name, NOW();
END;
$$ LANGUAGE plpgsql;

-- Note: Set up a cron job to refresh materialized views periodically
-- Example: SELECT cron.schedule('refresh-views', '*/15 * * * *', 'SELECT refresh_all_materialized_views()');

