-- DanteGPU Platform - Authentication and Security Tables
-- Migration 005: API keys, audit logs, security events, and access control

-- API keys table for programmatic access
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    key_name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL, -- First 8 chars for identification
    key_hash VARCHAR(255) NOT NULL UNIQUE, -- Hashed full key
    key_type VARCHAR(50) DEFAULT 'standard' CHECK (key_type IN ('standard', 'admin', 'readonly')),
    
    -- Permissions
    scopes TEXT[] DEFAULT '{}', -- Array of allowed scopes
    allowed_ips INET[] DEFAULT '{}', -- IP whitelist
    rate_limit_per_minute INTEGER DEFAULT 60,
    rate_limit_per_hour INTEGER DEFAULT 1000,
    rate_limit_per_day INTEGER DEFAULT 10000,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    is_revoked BOOLEAN DEFAULT false,
    
    -- Usage tracking
    last_used_at TIMESTAMP WITH TIME ZONE,
    last_used_ip INET,
    total_requests BIGINT DEFAULT 0,
    
    -- Expiration
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    description TEXT,
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    revoke_reason TEXT,
    
    CONSTRAINT positive_rate_limits CHECK (
        rate_limit_per_minute > 0 AND 
        rate_limit_per_hour > 0 AND 
        rate_limit_per_day > 0
    )
);

-- API key usage logs
CREATE TABLE IF NOT EXISTS api_key_usage (
    id BIGSERIAL PRIMARY KEY,
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    endpoint VARCHAR(500) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    response_time_ms INTEGER,
    ip_address INET,
    user_agent TEXT,
    request_size_bytes BIGINT,
    response_size_bytes BIGINT,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Audit logs for all system actions
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    
    -- Actor information
    user_id VARCHAR(255),
    api_key_id UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    actor_type VARCHAR(50) NOT NULL CHECK (actor_type IN ('user', 'api_key', 'system', 'admin')),
    
    -- Action details
    action VARCHAR(100) NOT NULL, -- e.g., 'user.login', 'job.create', 'wallet.transfer'
    resource_type VARCHAR(100), -- e.g., 'job', 'wallet', 'user'
    resource_id VARCHAR(255), -- ID of the affected resource
    
    -- Request details
    method VARCHAR(10),
    endpoint VARCHAR(500),
    ip_address INET,
    user_agent TEXT,
    
    -- Change tracking
    old_values JSONB,
    new_values JSONB,
    
    -- Result
    status VARCHAR(20) NOT NULL CHECK (status IN ('success', 'failure', 'error')),
    error_message TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Security events for threat detection
CREATE TABLE IF NOT EXISTS security_events (
    id BIGSERIAL PRIMARY KEY,
    
    -- Event classification
    event_type VARCHAR(100) NOT NULL CHECK (event_type IN (
        'failed_login', 'suspicious_activity', 'rate_limit_exceeded',
        'unauthorized_access', 'data_breach_attempt', 'malicious_request',
        'account_takeover_attempt', 'brute_force_attempt'
    )),
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    
    -- Actor information
    user_id VARCHAR(255),
    ip_address INET NOT NULL,
    user_agent TEXT,
    
    -- Event details
    description TEXT NOT NULL,
    details JSONB DEFAULT '{}',
    
    -- Response
    action_taken VARCHAR(100), -- e.g., 'blocked', 'flagged', 'rate_limited'
    is_resolved BOOLEAN DEFAULT false,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by VARCHAR(255),
    resolution_notes TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User roles and permissions
CREATE TABLE IF NOT EXISTS user_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('user', 'provider', 'admin', 'auditor', 'support')),
    granted_by VARCHAR(255),
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    UNIQUE(user_id, role)
);

-- Permissions table for fine-grained access control
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    resource_type VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Role permissions mapping
CREATE TABLE IF NOT EXISTS role_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role VARCHAR(50) NOT NULL,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(role, permission_id)
);

-- User-specific permission overrides
CREATE TABLE IF NOT EXISTS user_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    is_granted BOOLEAN NOT NULL, -- true = grant, false = revoke
    granted_by VARCHAR(255),
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(user_id, permission_id)
);

-- Login attempts tracking
CREATE TABLE IF NOT EXISTS login_attempts (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    failure_reason VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Password reset tokens
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Email verification tokens
CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Two-factor authentication
CREATE TABLE IF NOT EXISTS two_factor_auth (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL UNIQUE,
    method VARCHAR(20) NOT NULL CHECK (method IN ('totp', 'sms', 'email')),
    secret_encrypted TEXT NOT NULL, -- Encrypted TOTP secret or phone number
    backup_codes_encrypted TEXT[], -- Encrypted backup codes
    is_enabled BOOLEAN DEFAULT false,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Session management
CREATE TABLE IF NOT EXISTS active_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    session_token_hash VARCHAR(255) NOT NULL UNIQUE,
    refresh_token_hash VARCHAR(255) UNIQUE,
    ip_address INET,
    user_agent TEXT,
    device_fingerprint VARCHAR(255),
    location JSONB, -- Geographic location data
    is_active BOOLEAN DEFAULT true,
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX idx_api_keys_active ON api_keys(is_active, is_revoked);

CREATE INDEX idx_api_key_usage_key_id ON api_key_usage(api_key_id);
CREATE INDEX idx_api_key_usage_created ON api_key_usage(created_at);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_status ON audit_logs(status);

CREATE INDEX idx_security_events_type ON security_events(event_type);
CREATE INDEX idx_security_events_severity ON security_events(severity);
CREATE INDEX idx_security_events_user_id ON security_events(user_id);
CREATE INDEX idx_security_events_ip ON security_events(ip_address);
CREATE INDEX idx_security_events_created ON security_events(created_at);
CREATE INDEX idx_security_events_resolved ON security_events(is_resolved);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role);
CREATE INDEX idx_user_roles_active ON user_roles(is_active);

CREATE INDEX idx_role_permissions_role ON role_permissions(role);
CREATE INDEX idx_role_permissions_permission ON role_permissions(permission_id);

CREATE INDEX idx_user_permissions_user_id ON user_permissions(user_id);
CREATE INDEX idx_user_permissions_permission ON user_permissions(permission_id);

CREATE INDEX idx_login_attempts_email ON login_attempts(email);
CREATE INDEX idx_login_attempts_ip ON login_attempts(ip_address);
CREATE INDEX idx_login_attempts_created ON login_attempts(created_at);

CREATE INDEX idx_password_reset_tokens_user ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_hash ON password_reset_tokens(token_hash);

CREATE INDEX idx_email_verification_tokens_user ON email_verification_tokens(user_id);
CREATE INDEX idx_email_verification_tokens_hash ON email_verification_tokens(token_hash);

CREATE INDEX idx_two_factor_auth_user ON two_factor_auth(user_id);

CREATE INDEX idx_active_sessions_user_id ON active_sessions(user_id);
CREATE INDEX idx_active_sessions_token ON active_sessions(session_token_hash);
CREATE INDEX idx_active_sessions_active ON active_sessions(is_active);

-- Partition audit_logs by month
CREATE TABLE IF NOT EXISTS audit_logs_partitioned (
    LIKE audit_logs INCLUDING ALL
) PARTITION BY RANGE (created_at);

-- Create triggers
CREATE TRIGGER update_api_keys_updated_at 
    BEFORE UPDATE ON api_keys 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_two_factor_auth_updated_at 
    BEFORE UPDATE ON two_factor_auth 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to check user permission
CREATE OR REPLACE FUNCTION user_has_permission(
    p_user_id VARCHAR,
    p_permission_name VARCHAR
) RETURNS BOOLEAN AS $$
DECLARE
    has_perm BOOLEAN;
BEGIN
    -- Check direct user permission override
    SELECT is_granted INTO has_perm
    FROM user_permissions up
    JOIN permissions p ON up.permission_id = p.id
    WHERE up.user_id = p_user_id 
    AND p.name = p_permission_name
    AND (up.expires_at IS NULL OR up.expires_at > NOW())
    LIMIT 1;
    
    IF has_perm IS NOT NULL THEN
        RETURN has_perm;
    END IF;
    
    -- Check role-based permissions
    SELECT EXISTS (
        SELECT 1
        FROM user_roles ur
        JOIN role_permissions rp ON ur.role = rp.role
        JOIN permissions p ON rp.permission_id = p.id
        WHERE ur.user_id = p_user_id
        AND p.name = p_permission_name
        AND ur.is_active = true
        AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
    ) INTO has_perm;
    
    RETURN COALESCE(has_perm, false);
END;
$$ LANGUAGE plpgsql;

