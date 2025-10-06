-- Migration: Email Verification System
-- Description: Add email verification tokens and user preferences tables
-- Version: 012
-- Date: 2024-01-15

BEGIN;

-- ============================================================================
-- EMAIL VERIFICATION TOKENS
-- ============================================================================

CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_email_verification_user FOREIGN KEY (user_id) 
        REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for email verification tokens
CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);
CREATE INDEX idx_email_verification_tokens_token_hash ON email_verification_tokens(token_hash);
CREATE INDEX idx_email_verification_tokens_email ON email_verification_tokens(email);
CREATE INDEX idx_email_verification_tokens_expires_at ON email_verification_tokens(expires_at);
CREATE INDEX idx_email_verification_tokens_verified_at ON email_verification_tokens(verified_at) WHERE verified_at IS NULL;

-- ============================================================================
-- PASSWORD RESET TOKENS
-- ============================================================================

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_password_reset_user FOREIGN KEY (user_id) 
        REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for password reset tokens
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_token_hash ON password_reset_tokens(token_hash);
CREATE INDEX idx_password_reset_tokens_email ON password_reset_tokens(email);
CREATE INDEX idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);
CREATE INDEX idx_password_reset_tokens_used_at ON password_reset_tokens(used_at) WHERE used_at IS NULL;

-- ============================================================================
-- USER PREFERENCES
-- ============================================================================

CREATE TABLE IF NOT EXISTS user_preferences (
    user_id VARCHAR(255) PRIMARY KEY,
    language VARCHAR(10) DEFAULT 'en',
    timezone VARCHAR(50) DEFAULT 'UTC',
    currency VARCHAR(10) DEFAULT 'USD',
    
    -- Notification preferences
    email_notifications BOOLEAN DEFAULT true,
    sms_notifications BOOLEAN DEFAULT false,
    push_notifications BOOLEAN DEFAULT true,
    
    -- Email notification types
    notify_job_started BOOLEAN DEFAULT true,
    notify_job_completed BOOLEAN DEFAULT true,
    notify_job_failed BOOLEAN DEFAULT true,
    notify_payment_received BOOLEAN DEFAULT true,
    notify_payment_sent BOOLEAN DEFAULT true,
    notify_low_balance BOOLEAN DEFAULT true,
    notify_security_alerts BOOLEAN DEFAULT true,
    notify_marketing BOOLEAN DEFAULT false,
    
    -- Display preferences
    theme VARCHAR(20) DEFAULT 'light',
    items_per_page INTEGER DEFAULT 20,
    
    -- Privacy preferences
    profile_public BOOLEAN DEFAULT false,
    show_email BOOLEAN DEFAULT false,
    show_stats BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_user_preferences_user FOREIGN KEY (user_id) 
        REFERENCES users(id) ON DELETE CASCADE
);

-- ============================================================================
-- EMAIL CHANGE REQUESTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS email_change_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    old_email VARCHAR(255) NOT NULL,
    new_email VARCHAR(255) NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    confirmed_at TIMESTAMP WITH TIME ZONE,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_email_change_user FOREIGN KEY (user_id) 
        REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for email change requests
CREATE INDEX idx_email_change_requests_user_id ON email_change_requests(user_id);
CREATE INDEX idx_email_change_requests_token_hash ON email_change_requests(token_hash);
CREATE INDEX idx_email_change_requests_new_email ON email_change_requests(new_email);
CREATE INDEX idx_email_change_requests_expires_at ON email_change_requests(expires_at);

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to clean up expired verification tokens
CREATE OR REPLACE FUNCTION cleanup_expired_verification_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM email_verification_tokens
    WHERE expires_at < CURRENT_TIMESTAMP
    AND verified_at IS NULL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up expired password reset tokens
CREATE OR REPLACE FUNCTION cleanup_expired_reset_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM password_reset_tokens
    WHERE expires_at < CURRENT_TIMESTAMP
    AND used_at IS NULL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up expired email change requests
CREATE OR REPLACE FUNCTION cleanup_expired_email_changes()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM email_change_requests
    WHERE expires_at < CURRENT_TIMESTAMP
    AND confirmed_at IS NULL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get user verification status
CREATE OR REPLACE FUNCTION get_user_verification_status(p_user_id VARCHAR)
RETURNS TABLE (
    is_verified BOOLEAN,
    verification_pending BOOLEAN,
    last_verification_sent TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        u.is_verified,
        EXISTS(
            SELECT 1 FROM email_verification_tokens evt
            WHERE evt.user_id = p_user_id
            AND evt.verified_at IS NULL
            AND evt.expires_at > CURRENT_TIMESTAMP
        ) as verification_pending,
        (
            SELECT MAX(created_at)
            FROM email_verification_tokens
            WHERE user_id = p_user_id
        ) as last_verification_sent
    FROM users u
    WHERE u.id = p_user_id;
END;
$$ LANGUAGE plpgsql;

-- Function to check if user can request new verification
CREATE OR REPLACE FUNCTION can_request_verification(
    p_user_id VARCHAR,
    p_cooldown_minutes INTEGER DEFAULT 5
)
RETURNS BOOLEAN AS $$
DECLARE
    last_request TIMESTAMP WITH TIME ZONE;
BEGIN
    SELECT MAX(created_at) INTO last_request
    FROM email_verification_tokens
    WHERE user_id = p_user_id;
    
    IF last_request IS NULL THEN
        RETURN true;
    END IF;
    
    RETURN (CURRENT_TIMESTAMP - last_request) > (p_cooldown_minutes || ' minutes')::INTERVAL;
END;
$$ LANGUAGE plpgsql;

-- Function to check if user can request password reset
CREATE OR REPLACE FUNCTION can_request_password_reset(
    p_user_id VARCHAR,
    p_cooldown_minutes INTEGER DEFAULT 5
)
RETURNS BOOLEAN AS $$
DECLARE
    last_request TIMESTAMP WITH TIME ZONE;
BEGIN
    SELECT MAX(created_at) INTO last_request
    FROM password_reset_tokens
    WHERE user_id = p_user_id;
    
    IF last_request IS NULL THEN
        RETURN true;
    END IF;
    
    RETURN (CURRENT_TIMESTAMP - last_request) > (p_cooldown_minutes || ' minutes')::INTERVAL;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Trigger to update user_preferences updated_at
CREATE OR REPLACE FUNCTION update_user_preferences_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_preferences_timestamp
    BEFORE UPDATE ON user_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_user_preferences_timestamp();

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE email_verification_tokens IS 'Stores email verification tokens for new user registrations';
COMMENT ON TABLE password_reset_tokens IS 'Stores password reset tokens for password recovery';
COMMENT ON TABLE user_preferences IS 'Stores user preferences and notification settings';
COMMENT ON TABLE email_change_requests IS 'Stores pending email change requests';

COMMENT ON FUNCTION cleanup_expired_verification_tokens() IS 'Removes expired email verification tokens';
COMMENT ON FUNCTION cleanup_expired_reset_tokens() IS 'Removes expired password reset tokens';
COMMENT ON FUNCTION cleanup_expired_email_changes() IS 'Removes expired email change requests';
COMMENT ON FUNCTION get_user_verification_status(VARCHAR) IS 'Gets user email verification status';
COMMENT ON FUNCTION can_request_verification(VARCHAR, INTEGER) IS 'Checks if user can request new verification email';
COMMENT ON FUNCTION can_request_password_reset(VARCHAR, INTEGER) IS 'Checks if user can request password reset';

COMMIT;

-- ============================================================================
-- ROLLBACK
-- ============================================================================

-- To rollback this migration, run:
/*
BEGIN;

DROP TRIGGER IF EXISTS trigger_update_user_preferences_timestamp ON user_preferences;
DROP FUNCTION IF EXISTS update_user_preferences_timestamp();
DROP FUNCTION IF EXISTS can_request_password_reset(VARCHAR, INTEGER);
DROP FUNCTION IF EXISTS can_request_verification(VARCHAR, INTEGER);
DROP FUNCTION IF EXISTS get_user_verification_status(VARCHAR);
DROP FUNCTION IF EXISTS cleanup_expired_email_changes();
DROP FUNCTION IF EXISTS cleanup_expired_reset_tokens();
DROP FUNCTION IF EXISTS cleanup_expired_verification_tokens();

DROP TABLE IF EXISTS email_change_requests;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS email_verification_tokens;

COMMIT;
*/

