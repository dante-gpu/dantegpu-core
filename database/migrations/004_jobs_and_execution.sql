-- DanteGPU Platform - Jobs and Execution Tables
-- Migration 004: Job management, container orchestration, and execution tracking

-- Jobs table for GPU workload management
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id VARCHAR(255) UNIQUE NOT NULL, -- Human-readable job ID
    user_id VARCHAR(255) NOT NULL,
    provider_id UUID NOT NULL,
    session_id UUID REFERENCES rental_sessions(id) ON DELETE SET NULL,
    
    -- Job configuration
    name VARCHAR(255) NOT NULL,
    description TEXT,
    job_type VARCHAR(50) NOT NULL CHECK (job_type IN (
        'training', 'inference', 'rendering', 'mining', 'custom'
    )),
    
    -- Container configuration
    docker_image VARCHAR(500) NOT NULL,
    docker_tag VARCHAR(100) DEFAULT 'latest',
    command TEXT[], -- Array of command arguments
    environment_vars JSONB DEFAULT '{}',
    working_directory VARCHAR(500),
    
    -- Resource requirements
    required_gpu_model VARCHAR(255),
    required_vram_gb INTEGER NOT NULL,
    required_cpu_cores INTEGER DEFAULT 1,
    required_ram_gb INTEGER DEFAULT 4,
    required_storage_gb INTEGER DEFAULT 10,
    
    -- Network configuration
    exposed_ports INTEGER[] DEFAULT '{}',
    network_mode VARCHAR(50) DEFAULT 'bridge',
    
    -- Storage mounts
    input_data_path VARCHAR(500),
    output_data_path VARCHAR(500),
    mount_points JSONB DEFAULT '[]',
    
    -- Execution status
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'queued', 'pulling', 'starting', 'running', 
        'paused', 'stopping', 'stopped', 'completed', 'failed', 'cancelled'
    )),
    
    -- Container details
    container_id VARCHAR(255),
    container_name VARCHAR(255),
    container_status VARCHAR(50),
    
    -- Execution tracking
    queued_at TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    
    -- Exit information
    exit_code INTEGER,
    exit_message TEXT,
    error_message TEXT,
    
    -- Resource usage
    max_gpu_utilization SMALLINT,
    max_vram_usage_mb BIGINT,
    max_cpu_utilization SMALLINT,
    max_ram_usage_mb BIGINT,
    
    -- Retry configuration
    max_retries INTEGER DEFAULT 0,
    retry_count INTEGER DEFAULT 0,
    retry_delay_seconds INTEGER DEFAULT 60,
    
    -- Timeout configuration
    timeout_seconds INTEGER,
    
    -- Priority and scheduling
    priority INTEGER DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    scheduled_start_time TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT positive_resources CHECK (
        required_vram_gb > 0 AND 
        required_cpu_cores > 0 AND 
        required_ram_gb > 0 AND 
        required_storage_gb > 0
    ),
    CONSTRAINT valid_retry CHECK (retry_count <= max_retries)
);

-- Job logs table for streaming logs
CREATE TABLE IF NOT EXISTS job_logs (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    stream VARCHAR(10) NOT NULL CHECK (stream IN ('stdout', 'stderr', 'system')),
    message TEXT NOT NULL,
    log_level VARCHAR(20) DEFAULT 'info' CHECK (log_level IN (
        'debug', 'info', 'warning', 'error', 'critical'
    )),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Job metrics for performance tracking
CREATE TABLE IF NOT EXISTS job_metrics (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- GPU metrics
    gpu_utilization SMALLINT CHECK (gpu_utilization >= 0 AND gpu_utilization <= 100),
    gpu_memory_used_mb BIGINT,
    gpu_memory_total_mb BIGINT,
    gpu_temperature_c SMALLINT,
    gpu_power_draw_w INTEGER,
    gpu_clock_mhz INTEGER,
    gpu_memory_clock_mhz INTEGER,
    
    -- CPU metrics
    cpu_utilization SMALLINT CHECK (cpu_utilization >= 0 AND cpu_utilization <= 100),
    cpu_cores_used DECIMAL(4,2),
    
    -- Memory metrics
    ram_used_mb BIGINT,
    ram_total_mb BIGINT,
    swap_used_mb BIGINT,
    
    -- Disk I/O
    disk_read_mb BIGINT,
    disk_write_mb BIGINT,
    disk_read_iops INTEGER,
    disk_write_iops INTEGER,
    
    -- Network I/O
    network_rx_mb BIGINT,
    network_tx_mb BIGINT,
    network_rx_packets BIGINT,
    network_tx_packets BIGINT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Job files for input/output management
CREATE TABLE IF NOT EXISTS job_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    file_type VARCHAR(20) NOT NULL CHECK (file_type IN ('input', 'output', 'checkpoint', 'log')),
    file_name VARCHAR(500) NOT NULL,
    file_path VARCHAR(1000) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(100),
    checksum VARCHAR(64), -- SHA-256 hash
    storage_backend VARCHAR(50) DEFAULT 'minio' CHECK (storage_backend IN ('minio', 's3', 'local')),
    storage_url TEXT,
    is_public BOOLEAN DEFAULT false,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT positive_file_size CHECK (file_size_bytes >= 0)
);

-- Job checkpoints for long-running jobs
CREATE TABLE IF NOT EXISTS job_checkpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    checkpoint_number INTEGER NOT NULL,
    checkpoint_name VARCHAR(255),
    file_id UUID REFERENCES job_files(id),
    progress_percentage SMALLINT CHECK (progress_percentage >= 0 AND progress_percentage <= 100),
    metrics JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(job_id, checkpoint_number)
);

-- Job dependencies for workflow management
CREATE TABLE IF NOT EXISTS job_dependencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    depends_on_job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    dependency_type VARCHAR(50) DEFAULT 'completion' CHECK (dependency_type IN (
        'completion', 'success', 'failure', 'output'
    )),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(job_id, depends_on_job_id),
    CHECK (job_id != depends_on_job_id)
);

-- Job events for audit trail
CREATE TABLE IF NOT EXISTS job_events (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB DEFAULT '{}',
    triggered_by VARCHAR(255), -- user_id or 'system'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_jobs_job_id ON jobs(job_id);
CREATE INDEX idx_jobs_user_id ON jobs(user_id);
CREATE INDEX idx_jobs_provider_id ON jobs(provider_id);
CREATE INDEX idx_jobs_session_id ON jobs(session_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_status_created ON jobs(status, created_at);
CREATE INDEX idx_jobs_user_status ON jobs(user_id, status);
CREATE INDEX idx_jobs_provider_status ON jobs(provider_id, status);
CREATE INDEX idx_jobs_priority_queued ON jobs(priority DESC, queued_at ASC) WHERE status = 'queued';

CREATE INDEX idx_job_logs_job_id ON job_logs(job_id);
CREATE INDEX idx_job_logs_timestamp ON job_logs(timestamp);
CREATE INDEX idx_job_logs_job_timestamp ON job_logs(job_id, timestamp);
CREATE INDEX idx_job_logs_stream ON job_logs(stream);

CREATE INDEX idx_job_metrics_job_id ON job_metrics(job_id);
CREATE INDEX idx_job_metrics_recorded ON job_metrics(recorded_at);
CREATE INDEX idx_job_metrics_job_recorded ON job_metrics(job_id, recorded_at);

CREATE INDEX idx_job_files_job_id ON job_files(job_id);
CREATE INDEX idx_job_files_type ON job_files(file_type);
CREATE INDEX idx_job_files_job_type ON job_files(job_id, file_type);

CREATE INDEX idx_job_checkpoints_job_id ON job_checkpoints(job_id);
CREATE INDEX idx_job_checkpoints_number ON job_checkpoints(checkpoint_number);

CREATE INDEX idx_job_dependencies_job ON job_dependencies(job_id);
CREATE INDEX idx_job_dependencies_depends ON job_dependencies(depends_on_job_id);

CREATE INDEX idx_job_events_job_id ON job_events(job_id);
CREATE INDEX idx_job_events_created ON job_events(created_at);
CREATE INDEX idx_job_events_type ON job_events(event_type);

-- Partition job_logs by month for better performance
CREATE TABLE IF NOT EXISTS job_logs_partitioned (
    LIKE job_logs INCLUDING ALL
) PARTITION BY RANGE (timestamp);

-- Partition job_metrics by month
CREATE TABLE IF NOT EXISTS job_metrics_partitioned (
    LIKE job_metrics INCLUDING ALL
) PARTITION BY RANGE (recorded_at);

-- Create updated_at trigger
CREATE TRIGGER update_jobs_updated_at 
    BEFORE UPDATE ON jobs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to get job status summary
CREATE OR REPLACE FUNCTION get_job_status_summary(p_user_id VARCHAR)
RETURNS TABLE (
    status VARCHAR,
    count BIGINT,
    total_cost DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        j.status,
        COUNT(*)::BIGINT,
        COALESCE(SUM(rs.total_cost), 0)::DECIMAL
    FROM jobs j
    LEFT JOIN rental_sessions rs ON j.session_id = rs.id
    WHERE j.user_id = p_user_id
    GROUP BY j.status;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate job duration
CREATE OR REPLACE FUNCTION calculate_job_duration(p_job_id UUID)
RETURNS INTERVAL AS $$
DECLARE
    duration INTERVAL;
BEGIN
    SELECT 
        CASE 
            WHEN completed_at IS NOT NULL THEN completed_at - started_at
            WHEN failed_at IS NOT NULL THEN failed_at - started_at
            WHEN started_at IS NOT NULL THEN NOW() - started_at
            ELSE INTERVAL '0'
        END INTO duration
    FROM jobs
    WHERE id = p_job_id;
    
    RETURN COALESCE(duration, INTERVAL '0');
END;
$$ LANGUAGE plpgsql;

-- Function to check if job can start (dependencies met)
CREATE OR REPLACE FUNCTION can_job_start(p_job_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    unmet_dependencies INTEGER;
BEGIN
    SELECT COUNT(*) INTO unmet_dependencies
    FROM job_dependencies jd
    JOIN jobs j ON jd.depends_on_job_id = j.id
    WHERE jd.job_id = p_job_id
    AND (
        (jd.dependency_type = 'completion' AND j.status NOT IN ('completed', 'failed'))
        OR (jd.dependency_type = 'success' AND j.status != 'completed')
        OR (jd.dependency_type = 'failure' AND j.status != 'failed')
    );
    
    RETURN unmet_dependencies = 0;
END;
$$ LANGUAGE plpgsql;

