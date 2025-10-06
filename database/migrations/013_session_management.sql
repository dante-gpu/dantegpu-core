-- Migration: Session Management System
-- Description: Add tables for session management, login attempts, and 2FA sessions
-- Version: 013
-- Date: 2024-01-15

BEGIN;

-- ============================================================================
-- TWO FACTOR SESSIONS (temporary sessions for 2FA verification)
-- ============================================================================

CREATE TABLE IF NOT EXISTS two_factor_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    session_token TEXT NOT NULL,
    ip_address INET,
    user_agent TEXT,
    verified BOOLEAN DEFAULT false,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_2fa_session_user FOREIGN KEY (user_id) 
        REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_2fa_sessions_user_id ON two_factor_sessions(user_id);
CREATE INDEX idx_2fa_sessions_token ON two_factor_sessions(session_token);
CREATE INDEX idx_2fa_sessions_expires_at ON two_factor_sessions(expires_at);

-- ============================================================================
-- REVOKED TOKENS (for token blacklisting)
-- ============================================================================

CREATE TABLE IF NOT EXISTS revoked_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_id TEXT NOT NULL UNIQUE,
    user_id VARCHAR(255) NOT NULL,
    reason VARCHAR(255),
    revoked_by VARCHAR(255),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_revoked_token_user FOREIGN KEY (user_id) 
        REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_revoked_tokens_token_id ON revoked_tokens(token_id);
CREATE INDEX idx_revoked_tokens_user_id ON revoked_tokens(user_id);
CREATE INDEX idx_revoked_tokens_expires_at ON revoked_tokens(expires_at);

-- ============================================================================
-- API REQUESTS (for rate limiting)
-- ============================================================================

CREATE TABLE IF NOT EXISTS api_requests (
    id BIGSERIAL PRIMARY KEY,
    ip_address INET NOT NULL,
    path TEXT NOT NULL,
    method VARCHAR(10),
    status_code INTEGER,
    user_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_requests_ip_address ON api_requests(ip_address, created_at);
CREATE INDEX idx_api_requests_user_id ON api_requests(user_id, created_at);
CREATE INDEX idx_api_requests_created_at ON api_requests(created_at);

-- Partition by month for api_requests
CREATE TABLE IF NOT EXISTS api_requests_partitioned (
    LIKE api_requests INCLUDING ALL
) PARTITION BY RANGE (created_at);

-- Create partitions for 2024-2025
CREATE TABLE IF NOT EXISTS api_requests_2024_01 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE IF NOT EXISTS api_requests_2024_02 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE IF NOT EXISTS api_requests_2024_03 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE IF NOT EXISTS api_requests_2024_04 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE IF NOT EXISTS api_requests_2024_05 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE IF NOT EXISTS api_requests_2024_06 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE IF NOT EXISTS api_requests_2024_07 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE IF NOT EXISTS api_requests_2024_08 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE IF NOT EXISTS api_requests_2024_09 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE IF NOT EXISTS api_requests_2024_10 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE IF NOT EXISTS api_requests_2024_11 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE IF NOT EXISTS api_requests_2024_12 PARTITION OF api_requests_partitioned
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to clean up expired 2FA sessions
CREATE OR REPLACE FUNCTION cleanup_expired_2fa_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM two_factor_sessions
    WHERE expires_at < CURRENT_TIMESTAMP;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up expired revoked tokens
CREATE OR REPLACE FUNCTION cleanup_expired_revoked_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM revoked_tokens
    WHERE expires_at < CURRENT_TIMESTAMP;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old API requests (keep last 30 days)
CREATE OR REPLACE FUNCTION cleanup_old_api_requests()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM api_requests
    WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '30 days';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get user session count
CREATE OR REPLACE FUNCTION get_user_session_count(p_user_id VARCHAR)
RETURNS INTEGER AS $$
DECLARE
    session_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO session_count
    FROM active_sessions
    WHERE user_id = p_user_id
    AND revoked_at IS NULL
    AND expires_at > CURRENT_TIMESTAMP;
    
    RETURN session_count;
END;
$$ LANGUAGE plpgsql;

-- Function to revoke all user sessions
CREATE OR REPLACE FUNCTION revoke_all_user_sessions(p_user_id VARCHAR)
RETURNS INTEGER AS $$
DECLARE
    revoked_count INTEGER;
BEGIN
    UPDATE active_sessions
    SET revoked_at = CURRENT_TIMESTAMP
    WHERE user_id = p_user_id
    AND revoked_at IS NULL;
    
    GET DIAGNOSTICS revoked_count = ROW_COUNT;
    RETURN revoked_count;
END;
$$ LANGUAGE plpgsql;

-- Function to check rate limit
CREATE OR REPLACE FUNCTION check_rate_limit(
    p_ip_address INET,
    p_window_minutes INTEGER DEFAULT 1,
    p_max_requests INTEGER DEFAULT 60
)
RETURNS BOOLEAN AS $$
DECLARE
    request_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO request_count
    FROM api_requests
    WHERE ip_address = p_ip_address
    AND created_at > CURRENT_TIMESTAMP - (p_window_minutes || ' minutes')::INTERVAL;
    
    RETURN request_count < p_max_requests;
END;
$$ LANGUAGE plpgsql;

-- Function to get user login history
CREATE OR REPLACE FUNCTION get_user_login_history(
    p_user_id VARCHAR,
    p_limit INTEGER DEFAULT 10
)
RETURNS TABLE (
    email VARCHAR,
    success BOOLEAN,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        la.email,
        la.success,
        la.ip_address,
        la.user_agent,
        la.created_at
    FROM login_attempts la
    JOIN users u ON u.email = la.email
    WHERE u.id = p_user_id
    ORDER BY la.created_at DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Function to detect suspicious login activity
CREATE OR REPLACE FUNCTION detect_suspicious_login(
    p_user_id VARCHAR,
    p_ip_address INET
)
RETURNS BOOLEAN AS $$
DECLARE
    recent_ips INTEGER;
    failed_attempts INTEGER;
BEGIN
    -- Check for multiple different IPs in last hour
    SELECT COUNT(DISTINCT ip_address) INTO recent_ips
    FROM login_attempts
    WHERE email = (SELECT email FROM users WHERE id = p_user_id)
    AND created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour';
    
    -- Check for failed attempts
    SELECT COUNT(*) INTO failed_attempts
    FROM login_attempts
    WHERE email = (SELECT email FROM users WHERE id = p_user_id)
    AND success = false
    AND created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour';
    
    -- Suspicious if more than 3 different IPs or more than 3 failed attempts
    RETURN recent_ips > 3 OR failed_attempts > 3;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SCHEDULED CLEANUP JOBS (to be run by cron or scheduler)
-- ============================================================================

-- Create a function to run all cleanup tasks
CREATE OR REPLACE FUNCTION run_session_cleanup()
RETURNS TABLE (
    task VARCHAR,
    deleted_count INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 'expired_2fa_sessions'::VARCHAR, cleanup_expired_2fa_sessions();
    
    RETURN QUERY
    SELECT 'expired_revoked_tokens'::VARCHAR, cleanup_expired_revoked_tokens();
    
    RETURN QUERY
    SELECT 'old_api_requests'::VARCHAR, cleanup_old_api_requests();
    
    RETURN QUERY
    SELECT 'expired_sessions'::VARCHAR, (
        SELECT COUNT(*)::INTEGER FROM (
            UPDATE active_sessions
            SET revoked_at = CURRENT_TIMESTAMP
            WHERE expires_at < CURRENT_TIMESTAMP
            AND revoked_at IS NULL
            RETURNING 1
        ) AS subquery
    );
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE two_factor_sessions IS 'Temporary sessions for 2FA verification during login';
COMMENT ON TABLE revoked_tokens IS 'Blacklist of revoked JWT tokens';
COMMENT ON TABLE api_requests IS 'API request logs for rate limiting and analytics';

COMMENT ON FUNCTION cleanup_expired_2fa_sessions() IS 'Removes expired 2FA sessions';
COMMENT ON FUNCTION cleanup_expired_revoked_tokens() IS 'Removes expired revoked tokens';
COMMENT ON FUNCTION cleanup_old_api_requests() IS 'Removes API requests older than 30 days';
COMMENT ON FUNCTION get_user_session_count(VARCHAR) IS 'Gets count of active sessions for a user';
COMMENT ON FUNCTION revoke_all_user_sessions(VARCHAR) IS 'Revokes all sessions for a user';
COMMENT ON FUNCTION check_rate_limit(INET, INTEGER, INTEGER) IS 'Checks if IP has exceeded rate limit';
COMMENT ON FUNCTION get_user_login_history(VARCHAR, INTEGER) IS 'Gets login history for a user';
COMMENT ON FUNCTION detect_suspicious_login(VARCHAR, INET) IS 'Detects suspicious login activity';
COMMENT ON FUNCTION run_session_cleanup() IS 'Runs all session cleanup tasks';

COMMIT;

-- ============================================================================
-- ROLLBACK
-- ============================================================================

-- To rollback this migration, run:
/*
BEGIN;

DROP FUNCTION IF EXISTS run_session_cleanup();
DROP FUNCTION IF EXISTS detect_suspicious_login(VARCHAR, INET);
DROP FUNCTION IF EXISTS get_user_login_history(VARCHAR, INTEGER);
DROP FUNCTION IF EXISTS check_rate_limit(INET, INTEGER, INTEGER);
DROP FUNCTION IF EXISTS revoke_all_user_sessions(VARCHAR);
DROP FUNCTION IF EXISTS get_user_session_count(VARCHAR);
DROP FUNCTION IF EXISTS cleanup_old_api_requests();
DROP FUNCTION IF EXISTS cleanup_expired_revoked_tokens();
DROP FUNCTION IF EXISTS cleanup_expired_2fa_sessions();

DROP TABLE IF EXISTS api_requests_partitioned CASCADE;
DROP TABLE IF EXISTS api_requests;
DROP TABLE IF EXISTS revoked_tokens;
DROP TABLE IF EXISTS two_factor_sessions;

COMMIT;
*/

