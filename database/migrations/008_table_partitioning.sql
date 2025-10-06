-- DanteGPU Platform - Table Partitioning
-- Migration 008: Implement table partitioning for large tables

-- ============================================================================
-- PARTITION SETUP FOR JOB LOGS
-- ============================================================================

-- Create partitioned job_logs table
CREATE TABLE IF NOT EXISTS job_logs_new (
    id BIGSERIAL,
    job_id UUID NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    stream VARCHAR(10) NOT NULL CHECK (stream IN ('stdout', 'stderr', 'system')),
    message TEXT NOT NULL,
    log_level VARCHAR(20) DEFAULT 'info' CHECK (log_level IN (
        'debug', 'info', 'warning', 'error', 'critical'
    )),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Create partitions for the next 12 months
CREATE TABLE job_logs_2024_01 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE job_logs_2024_02 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

CREATE TABLE job_logs_2024_03 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');

CREATE TABLE job_logs_2024_04 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');

CREATE TABLE job_logs_2024_05 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');

CREATE TABLE job_logs_2024_06 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');

CREATE TABLE job_logs_2024_07 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');

CREATE TABLE job_logs_2024_08 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');

CREATE TABLE job_logs_2024_09 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');

CREATE TABLE job_logs_2024_10 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');

CREATE TABLE job_logs_2024_11 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');

CREATE TABLE job_logs_2024_12 PARTITION OF job_logs_new
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

CREATE TABLE job_logs_2025_01 PARTITION OF job_logs_new
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Create indexes on partitioned table
CREATE INDEX idx_job_logs_new_job_id ON job_logs_new(job_id);
CREATE INDEX idx_job_logs_new_timestamp ON job_logs_new(timestamp);
CREATE INDEX idx_job_logs_new_stream ON job_logs_new(stream);
CREATE INDEX idx_job_logs_new_log_level ON job_logs_new(log_level);

-- ============================================================================
-- PARTITION SETUP FOR JOB METRICS
-- ============================================================================

CREATE TABLE IF NOT EXISTS job_metrics_new (
    id BIGSERIAL,
    job_id UUID NOT NULL,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    gpu_utilization SMALLINT CHECK (gpu_utilization >= 0 AND gpu_utilization <= 100),
    gpu_memory_used_mb BIGINT,
    gpu_memory_total_mb BIGINT,
    gpu_temperature_c SMALLINT,
    gpu_power_draw_w INTEGER,
    gpu_clock_mhz INTEGER,
    gpu_memory_clock_mhz INTEGER,
    cpu_utilization SMALLINT CHECK (cpu_utilization >= 0 AND cpu_utilization <= 100),
    cpu_cores_used DECIMAL(4,2),
    ram_used_mb BIGINT,
    ram_total_mb BIGINT,
    swap_used_mb BIGINT,
    disk_read_mb BIGINT,
    disk_write_mb BIGINT,
    disk_read_iops INTEGER,
    disk_write_iops INTEGER,
    network_rx_mb BIGINT,
    network_tx_mb BIGINT,
    network_rx_packets BIGINT,
    network_tx_packets BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, recorded_at)
) PARTITION BY RANGE (recorded_at);

-- Create partitions for job_metrics
CREATE TABLE job_metrics_2024_01 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE job_metrics_2024_02 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

CREATE TABLE job_metrics_2024_03 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');

CREATE TABLE job_metrics_2024_04 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');

CREATE TABLE job_metrics_2024_05 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');

CREATE TABLE job_metrics_2024_06 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');

CREATE TABLE job_metrics_2024_07 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');

CREATE TABLE job_metrics_2024_08 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');

CREATE TABLE job_metrics_2024_09 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');

CREATE TABLE job_metrics_2024_10 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');

CREATE TABLE job_metrics_2024_11 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');

CREATE TABLE job_metrics_2024_12 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

CREATE TABLE job_metrics_2025_01 PARTITION OF job_metrics_new
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Create indexes on partitioned table
CREATE INDEX idx_job_metrics_new_job_id ON job_metrics_new(job_id);
CREATE INDEX idx_job_metrics_new_recorded ON job_metrics_new(recorded_at);

-- ============================================================================
-- PARTITION SETUP FOR AUDIT LOGS
-- ============================================================================

CREATE TABLE IF NOT EXISTS audit_logs_new (
    id BIGSERIAL,
    user_id VARCHAR(255),
    api_key_id UUID,
    actor_type VARCHAR(50) NOT NULL CHECK (actor_type IN ('user', 'api_key', 'system', 'admin')),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    method VARCHAR(10),
    endpoint VARCHAR(500),
    ip_address INET,
    user_agent TEXT,
    old_values JSONB,
    new_values JSONB,
    status VARCHAR(20) NOT NULL CHECK (status IN ('success', 'failure', 'error')),
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create partitions for audit_logs
CREATE TABLE audit_logs_2024_01 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE audit_logs_2024_02 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

CREATE TABLE audit_logs_2024_03 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');

CREATE TABLE audit_logs_2024_04 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');

CREATE TABLE audit_logs_2024_05 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');

CREATE TABLE audit_logs_2024_06 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');

CREATE TABLE audit_logs_2024_07 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');

CREATE TABLE audit_logs_2024_08 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');

CREATE TABLE audit_logs_2024_09 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');

CREATE TABLE audit_logs_2024_10 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');

CREATE TABLE audit_logs_2024_11 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');

CREATE TABLE audit_logs_2024_12 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

CREATE TABLE audit_logs_2025_01 PARTITION OF audit_logs_new
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Create indexes on partitioned table
CREATE INDEX idx_audit_logs_new_user_id ON audit_logs_new(user_id);
CREATE INDEX idx_audit_logs_new_action ON audit_logs_new(action);
CREATE INDEX idx_audit_logs_new_resource ON audit_logs_new(resource_type, resource_id);
CREATE INDEX idx_audit_logs_new_created ON audit_logs_new(created_at);

-- ============================================================================
-- PARTITION SETUP FOR USAGE RECORDS
-- ============================================================================

CREATE TABLE IF NOT EXISTS usage_records_new (
    id BIGSERIAL,
    session_id UUID NOT NULL,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    gpu_utilization_percent SMALLINT CHECK (gpu_utilization_percent >= 0 AND gpu_utilization_percent <= 100),
    vram_utilization_percent SMALLINT CHECK (vram_utilization_percent >= 0 AND vram_utilization_percent <= 100),
    power_draw_w INTEGER NOT NULL CHECK (power_draw_w >= 0),
    temperature_c SMALLINT,
    period_minutes INTEGER NOT NULL CHECK (period_minutes > 0),
    period_cost DECIMAL(20,9) NOT NULL CHECK (period_cost >= 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, recorded_at)
) PARTITION BY RANGE (recorded_at);

-- Create partitions for usage_records
CREATE TABLE usage_records_2024_01 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE usage_records_2024_02 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

CREATE TABLE usage_records_2024_03 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');

CREATE TABLE usage_records_2024_04 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');

CREATE TABLE usage_records_2024_05 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');

CREATE TABLE usage_records_2024_06 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');

CREATE TABLE usage_records_2024_07 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');

CREATE TABLE usage_records_2024_08 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');

CREATE TABLE usage_records_2024_09 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');

CREATE TABLE usage_records_2024_10 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');

CREATE TABLE usage_records_2024_11 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');

CREATE TABLE usage_records_2024_12 PARTITION OF usage_records_new
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

CREATE TABLE usage_records_2025_01 PARTITION OF usage_records_new
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Create indexes on partitioned table
CREATE INDEX idx_usage_records_new_session ON usage_records_new(session_id);
CREATE INDEX idx_usage_records_new_recorded ON usage_records_new(recorded_at);

-- ============================================================================
-- AUTOMATIC PARTITION MANAGEMENT FUNCTION
-- ============================================================================

-- Function to create next month's partition
CREATE OR REPLACE FUNCTION create_next_month_partitions()
RETURNS void AS $$
DECLARE
    next_month_start DATE;
    next_month_end DATE;
    partition_name TEXT;
BEGIN
    -- Calculate next month
    next_month_start := DATE_TRUNC('month', CURRENT_DATE + INTERVAL '1 month');
    next_month_end := next_month_start + INTERVAL '1 month';
    
    -- Create partition name
    partition_name := TO_CHAR(next_month_start, 'YYYY_MM');
    
    -- Create job_logs partition
    EXECUTE format('CREATE TABLE IF NOT EXISTS job_logs_%s PARTITION OF job_logs_new FOR VALUES FROM (%L) TO (%L)',
        partition_name, next_month_start, next_month_end);
    
    -- Create job_metrics partition
    EXECUTE format('CREATE TABLE IF NOT EXISTS job_metrics_%s PARTITION OF job_metrics_new FOR VALUES FROM (%L) TO (%L)',
        partition_name, next_month_start, next_month_end);
    
    -- Create audit_logs partition
    EXECUTE format('CREATE TABLE IF NOT EXISTS audit_logs_%s PARTITION OF audit_logs_new FOR VALUES FROM (%L) TO (%L)',
        partition_name, next_month_start, next_month_end);
    
    -- Create usage_records partition
    EXECUTE format('CREATE TABLE IF NOT EXISTS usage_records_%s PARTITION OF usage_records_new FOR VALUES FROM (%L) TO (%L)',
        partition_name, next_month_start, next_month_end);
    
    RAISE NOTICE 'Created partitions for %', partition_name;
END;
$$ LANGUAGE plpgsql;

-- Function to drop old partitions (older than 6 months)
CREATE OR REPLACE FUNCTION drop_old_partitions(months_to_keep INTEGER DEFAULT 6)
RETURNS void AS $$
DECLARE
    cutoff_date DATE;
    partition_record RECORD;
BEGIN
    cutoff_date := DATE_TRUNC('month', CURRENT_DATE - (months_to_keep || ' months')::INTERVAL);
    
    -- Drop old job_logs partitions
    FOR partition_record IN
        SELECT tablename FROM pg_tables
        WHERE schemaname = 'public'
        AND tablename LIKE 'job_logs_20%'
        AND tablename < 'job_logs_' || TO_CHAR(cutoff_date, 'YYYY_MM')
    LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || partition_record.tablename;
        RAISE NOTICE 'Dropped partition %', partition_record.tablename;
    END LOOP;
    
    -- Similar for other partitioned tables
    FOR partition_record IN
        SELECT tablename FROM pg_tables
        WHERE schemaname = 'public'
        AND (tablename LIKE 'job_metrics_20%' OR tablename LIKE 'audit_logs_20%' OR tablename LIKE 'usage_records_20%')
        AND tablename < SUBSTRING(tablename FROM '^[a-z_]+') || TO_CHAR(cutoff_date, 'YYYY_MM')
    LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || partition_record.tablename;
        RAISE NOTICE 'Dropped partition %', partition_record.tablename;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Note: Set up a cron job or pg_cron to run create_next_month_partitions() monthly
-- Example: SELECT cron.schedule('create-partitions', '0 0 1 * *', 'SELECT create_next_month_partitions()');

