-- DanteGPU Platform - Database Roles and Seed Data
-- Migration 011: Database roles, permissions, seed data, and rollback procedures

-- ============================================================================
-- DATABASE ROLES AND PERMISSIONS
-- ============================================================================

-- Create read-only role for analytics
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'dante_readonly') THEN
        CREATE ROLE dante_readonly;
    END IF;
END
$$;

GRANT CONNECT ON DATABASE dante_core TO dante_readonly;
GRANT USAGE ON SCHEMA public TO dante_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO dante_readonly;
GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO dante_readonly;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO dante_readonly;

-- Create application role with full access
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'dante_app') THEN
        CREATE ROLE dante_app;
    END IF;
END
$$;

GRANT CONNECT ON DATABASE dante_core TO dante_app;
GRANT USAGE, CREATE ON SCHEMA public TO dante_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO dante_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO dante_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO dante_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO dante_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO dante_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO dante_app;

-- Create migration role
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'dante_migration') THEN
        CREATE ROLE dante_migration;
    END IF;
END
$$;

GRANT ALL PRIVILEGES ON DATABASE dante_core TO dante_migration;
GRANT ALL PRIVILEGES ON SCHEMA public TO dante_migration;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO dante_migration;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO dante_migration;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO dante_migration;

-- ============================================================================
-- SEED DATA FOR PERMISSIONS
-- ============================================================================

-- Insert default permissions
INSERT INTO permissions (name, description, resource_type, action) VALUES
-- User permissions
('user.read', 'Read user information', 'user', 'read'),
('user.update', 'Update user information', 'user', 'update'),
('user.delete', 'Delete user account', 'user', 'delete'),

-- Job permissions
('job.create', 'Create new jobs', 'job', 'create'),
('job.read', 'Read job information', 'job', 'read'),
('job.update', 'Update job configuration', 'job', 'update'),
('job.delete', 'Delete jobs', 'job', 'delete'),
('job.execute', 'Execute jobs', 'job', 'execute'),

-- Wallet permissions
('wallet.read', 'Read wallet balance', 'wallet', 'read'),
('wallet.deposit', 'Deposit funds', 'wallet', 'deposit'),
('wallet.withdraw', 'Withdraw funds', 'wallet', 'withdraw'),
('wallet.transfer', 'Transfer funds', 'wallet', 'transfer'),

-- Provider permissions
('provider.register', 'Register as provider', 'provider', 'create'),
('provider.read', 'Read provider information', 'provider', 'read'),
('provider.update', 'Update provider settings', 'provider', 'update'),
('provider.gpu.register', 'Register GPU capabilities', 'gpu', 'create'),
('provider.gpu.update', 'Update GPU settings', 'gpu', 'update'),
('provider.payout.request', 'Request payout', 'payout', 'create'),

-- Admin permissions
('admin.user.manage', 'Manage all users', 'user', 'admin'),
('admin.provider.verify', 'Verify providers', 'provider', 'verify'),
('admin.provider.manage', 'Manage all providers', 'provider', 'admin'),
('admin.platform.manage', 'Manage platform settings', 'platform', 'admin'),
('admin.analytics.view', 'View platform analytics', 'analytics', 'read'),

-- Auditor permissions
('auditor.logs.read', 'Read audit logs', 'audit', 'read'),
('auditor.transactions.read', 'Read all transactions', 'transaction', 'read'),
('auditor.reports.generate', 'Generate audit reports', 'report', 'create')
ON CONFLICT (name) DO NOTHING;

-- Map permissions to roles
INSERT INTO role_permissions (role, permission_id)
SELECT 'user', id FROM permissions WHERE name IN (
    'user.read', 'user.update',
    'job.create', 'job.read', 'job.update', 'job.delete',
    'wallet.read', 'wallet.deposit', 'wallet.withdraw'
)
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'provider', id FROM permissions WHERE name IN (
    'user.read', 'user.update',
    'provider.read', 'provider.update',
    'provider.gpu.register', 'provider.gpu.update',
    'provider.payout.request',
    'wallet.read', 'wallet.deposit', 'wallet.withdraw'
)
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'admin', id FROM permissions
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'auditor', id FROM permissions WHERE name LIKE 'auditor.%'
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SEED DATA FOR DEVELOPMENT
-- ============================================================================

-- Insert platform wallet
INSERT INTO wallets (user_id, wallet_type, solana_address, balance, is_active)
VALUES ('platform', 'platform', '7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump', 0, true)
ON CONFLICT DO NOTHING;

-- Insert sample GPU models for reference
INSERT INTO gpu_models (name, manufacturer, architecture, memory_gb, memory_type, cuda_cores, tensor_cores, features, benchmarks) VALUES
('RTX 4090', 'NVIDIA', 'Ada Lovelace', 24, 'GDDR6X', 16384, 512,
 '{"dlss": "3.0", "ray_tracing": true, "nvenc": true}',
 '{"fp32_tflops": 82.6, "fp16_tflops": 165.2}'),
 
('RTX 4080', 'NVIDIA', 'Ada Lovelace', 16, 'GDDR6X', 9728, 304,
 '{"dlss": "3.0", "ray_tracing": true, "nvenc": true}',
 '{"fp32_tflops": 48.7, "fp16_tflops": 97.4}'),
 
('A100 80GB', 'NVIDIA', 'Ampere', 80, 'HBM2e', 6912, 432,
 '{"multi_instance": true, "nvlink": true, "tensor_cores": "3rd Gen"}',
 '{"fp32_tflops": 19.5, "fp16_tflops": 312}'),
 
('H100 80GB', 'NVIDIA', 'Hopper', 80, 'HBM3', 14592, 456,
 '{"transformer_engine": true, "nvlink": "4th Gen", "tensor_cores": "4th Gen"}',
 '{"fp32_tflops": 51, "fp16_tflops": 1979}'),
 
('A6000', 'NVIDIA', 'Ampere', 48, 'GDDR6', 10752, 336,
 '{"ray_tracing": true, "nvenc": true}',
 '{"fp32_tflops": 38.7, "fp16_tflops": 77.4}'),
 
('RTX 3090', 'NVIDIA', 'Ampere', 24, 'GDDR6X', 10496, 328,
 '{"dlss": "2.0", "ray_tracing": true, "nvenc": true}',
 '{"fp32_tflops": 35.6, "fp16_tflops": 71.2}'),
 
('MI250X', 'AMD', 'CDNA 2', 128, 'HBM2e', 14080, 0,
 '{"infinity_fabric": true, "rocm": true}',
 '{"fp32_tflops": 47.9, "fp16_tflops": 383}'),
 
('MI210', 'AMD', 'CDNA 2', 64, 'HBM2e', 6656, 0,
 '{"infinity_fabric": true, "rocm": true}',
 '{"fp32_tflops": 45.3, "fp16_tflops": 181}')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- ROLLBACK PROCEDURES
-- ============================================================================

-- Function to rollback a migration
CREATE OR REPLACE FUNCTION rollback_migration(p_version VARCHAR)
RETURNS void AS $$
DECLARE
    v_migration RECORD;
BEGIN
    -- Get migration details
    SELECT * INTO v_migration
    FROM schema_migrations
    WHERE version = p_version;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Migration % not found', p_version;
    END IF;
    
    -- Log rollback attempt
    RAISE NOTICE 'Rolling back migration: %', p_version;
    
    -- Delete migration record
    DELETE FROM schema_migrations WHERE version = p_version;
    
    RAISE NOTICE 'Migration % rolled back. Manual cleanup may be required.', p_version;
END;
$$ LANGUAGE plpgsql;

-- Function to get migration status
CREATE OR REPLACE FUNCTION get_migration_status()
RETURNS TABLE (
    version VARCHAR,
    name VARCHAR,
    executed_at TIMESTAMP WITH TIME ZONE,
    execution_time_ms INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        sm.version,
        sm.name,
        sm.executed_at,
        sm.execution_time_ms
    FROM schema_migrations sm
    ORDER BY sm.id;
END;
$$ LANGUAGE plpgsql;

-- Function to validate database schema
CREATE OR REPLACE FUNCTION validate_database_schema()
RETURNS TABLE (
    check_name VARCHAR,
    status VARCHAR,
    details TEXT
) AS $$
BEGIN
    -- Check for missing indexes
    RETURN QUERY
    SELECT 
        'Missing Indexes'::VARCHAR,
        CASE WHEN COUNT(*) = 0 THEN 'OK' ELSE 'WARNING' END::VARCHAR,
        'Found ' || COUNT(*)::TEXT || ' foreign keys without indexes'::TEXT
    FROM (
        SELECT 
            tc.table_name,
            kcu.column_name
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu 
            ON tc.constraint_name = kcu.constraint_name
        WHERE tc.constraint_type = 'FOREIGN KEY'
        AND NOT EXISTS (
            SELECT 1 FROM pg_indexes
            WHERE tablename = tc.table_name
            AND indexdef LIKE '%' || kcu.column_name || '%'
        )
    ) missing_indexes;
    
    -- Check for tables without primary keys
    RETURN QUERY
    SELECT 
        'Tables Without Primary Keys'::VARCHAR,
        CASE WHEN COUNT(*) = 0 THEN 'OK' ELSE 'ERROR' END::VARCHAR,
        'Found ' || COUNT(*)::TEXT || ' tables without primary keys'::TEXT
    FROM information_schema.tables t
    WHERE t.table_schema = 'public'
    AND t.table_type = 'BASE TABLE'
    AND NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        WHERE tc.table_name = t.table_name
        AND tc.constraint_type = 'PRIMARY KEY'
    );
    
    -- Check for unused indexes
    RETURN QUERY
    SELECT 
        'Unused Indexes'::VARCHAR,
        'INFO'::VARCHAR,
        'Found ' || COUNT(*)::TEXT || ' potentially unused indexes'::TEXT
    FROM pg_stat_user_indexes
    WHERE idx_scan = 0
    AND indexrelname NOT LIKE '%_pkey';
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- MAINTENANCE FUNCTIONS
-- ============================================================================

-- Function to vacuum and analyze all tables
CREATE OR REPLACE FUNCTION maintenance_vacuum_analyze()
RETURNS void AS $$
DECLARE
    table_record RECORD;
BEGIN
    FOR table_record IN
        SELECT tablename FROM pg_tables WHERE schemaname = 'public'
    LOOP
        EXECUTE 'VACUUM ANALYZE ' || table_record.tablename;
        RAISE NOTICE 'Vacuumed and analyzed table: %', table_record.tablename;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Function to reindex all tables
CREATE OR REPLACE FUNCTION maintenance_reindex_all()
RETURNS void AS $$
DECLARE
    table_record RECORD;
BEGIN
    FOR table_record IN
        SELECT tablename FROM pg_tables WHERE schemaname = 'public'
    LOOP
        EXECUTE 'REINDEX TABLE ' || table_record.tablename;
        RAISE NOTICE 'Reindexed table: %', table_record.tablename;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Function to get database size statistics
CREATE OR REPLACE FUNCTION get_database_size_stats()
RETURNS TABLE (
    table_name VARCHAR,
    row_count BIGINT,
    total_size TEXT,
    table_size TEXT,
    indexes_size TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.tablename::VARCHAR,
        (SELECT COUNT(*) FROM pg_class WHERE relname = t.tablename)::BIGINT,
        pg_size_pretty(pg_total_relation_size(t.tablename::regclass))::TEXT,
        pg_size_pretty(pg_relation_size(t.tablename::regclass))::TEXT,
        pg_size_pretty(pg_total_relation_size(t.tablename::regclass) - pg_relation_size(t.tablename::regclass))::TEXT
    FROM pg_tables t
    WHERE t.schemaname = 'public'
    ORDER BY pg_total_relation_size(t.tablename::regclass) DESC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- INITIAL SETUP COMPLETE
-- ============================================================================

-- Log completion
DO $$
BEGIN
    RAISE NOTICE '=================================================================';
    RAISE NOTICE 'DanteGPU Platform Database Setup Complete';
    RAISE NOTICE '=================================================================';
    RAISE NOTICE 'Database roles created: dante_readonly, dante_app, dante_migration';
    RAISE NOTICE 'Permissions configured for: user, provider, admin, auditor';
    RAISE NOTICE 'Seed data inserted for GPU models and platform wallet';
    RAISE NOTICE 'Maintenance functions available';
    RAISE NOTICE '=================================================================';
END $$;

