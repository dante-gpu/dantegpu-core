# DanteGPU Platform - Database Documentation

## Overview

This directory contains all database schemas, migrations, and setup scripts for the DanteGPU decentralized GPU rental platform.

## Database Architecture

The platform uses **PostgreSQL 15+** with the following databases:

- `dante_core` - Main application database (default)
- `dante_auth` - Authentication and user management
- `dante_billing` - Payment and billing records
- `dante_registry` - Provider and GPU registry
- `dante_scheduler` - Job scheduling and orchestration

## Migration Files

### Core Schema Migrations

1. **001_initial_schema.sql** (168 lines)
   - Users table
   - GPU providers table
   - GPU models table
   - GPU instances table
   - GPU rentals table
   - Payment transactions table
   - GPU metrics table
   - User sessions table
   - Notifications table
   - Basic indexes and triggers

2. **002_sample_data.sql** (70 lines)
   - Sample GPU providers
   - Sample GPU models (RTX 4090, A100, RTX 3080, H100)
   - Sample GPU instances
   - Sample users
   - Sample rentals and transactions

### Blockchain Integration (Migration 003)

**003_blockchain_tables.sql** (300 lines)

Tables:
- `wallets` - dGPU token wallet management
- `blockchain_transactions` - Solana transaction tracking
- `rental_sessions` - Minute-based billing sessions
- `usage_records` - GPU usage metering
- `escrow_transactions` - Secure payment escrow
- `platform_fees` - Platform fee collection (5%)
- `provider_payouts` - Provider earnings distribution

Functions:
- `wallet_available_balance(wallet_id)` - Calculate available balance
- `lock_wallet_funds(wallet_id, amount)` - Lock funds for escrow
- `release_wallet_funds(wallet_id, amount)` - Release locked funds

### Job Execution (Migration 004)

**004_jobs_and_execution.sql** (300 lines)

Tables:
- `jobs` - GPU job management with Docker orchestration
- `job_logs` - Streaming logs (partitioned by month)
- `job_metrics` - Performance metrics (partitioned by month)
- `job_files` - Input/output file management
- `job_checkpoints` - Checkpoint management for long-running jobs
- `job_dependencies` - Workflow dependencies
- `job_events` - Audit trail for job events

Functions:
- `get_job_status_summary(user_id)` - Get job statistics
- `calculate_job_duration(job_id)` - Calculate job runtime
- `can_job_start(job_id)` - Check if dependencies are met

### Authentication & Security (Migration 005)

**005_auth_and_security.sql** (300 lines)

Tables:
- `api_keys` - API key management with scopes and rate limiting
- `api_key_usage` - API usage tracking
- `audit_logs` - System-wide audit logging (partitioned)
- `security_events` - Threat detection and security monitoring
- `user_roles` - Role assignments
- `permissions` - Permission definitions
- `role_permissions` - Role-permission mapping
- `user_permissions` - User-specific permission overrides
- `login_attempts` - Login attempt tracking
- `password_reset_tokens` - Password reset flow
- `email_verification_tokens` - Email verification
- `two_factor_auth` - 2FA (TOTP, SMS, email)
- `active_sessions` - Session management

Functions:
- `user_has_permission(user_id, permission_name)` - Check user permissions

### Provider & GPU Registry (Migration 006)

**006_providers_and_gpu_registry.sql** (300 lines)

Tables:
- `providers` - Enhanced provider management
- `gpu_capabilities` - Detailed GPU specifications
- `gpu_availability_schedule` - Availability scheduling
- `gpu_reservations` - GPU reservation system
- `provider_reviews` - Provider ratings and reviews
- `provider_payout_requests` - Payout request management

Functions:
- `update_provider_rating(provider_id)` - Recalculate provider rating
- `is_gpu_available(gpu_id, start_time, end_time)` - Check availability

### Performance Optimization (Migration 007)

**007_additional_indexes.sql** (300 lines)

Features:
- Composite indexes for common query patterns
- Partial indexes for filtered queries
- GIN indexes for JSONB columns
- Full-text search with tsvector
- Covering indexes for index-only scans
- Statistics optimization

## Running Migrations

### Prerequisites

- PostgreSQL 15 or higher
- `psql` command-line tool
- Database credentials configured in `.env` file

### Environment Variables

Create a `.env` file in the project root:

```bash
DB_HOST=localhost
DB_PORT=5432
DB_NAME=dante_core
DB_USER=dante_user
DB_PASSWORD=your_secure_password
```

### Run All Migrations

```bash
./database/run_migrations.sh run
```

### Check Migration Status

```bash
./database/run_migrations.sh status
```

### Migration Tracking

Migrations are tracked in the `schema_migrations` table:

```sql
SELECT * FROM schema_migrations ORDER BY id;
```

Each migration records:
- Version number
- Migration name
- Execution timestamp
- Execution time (ms)
- File checksum

## Database Schema Overview

### Key Tables and Relationships

```
users
  ├── wallets (1:N)
  ├── rental_sessions (1:N)
  ├── jobs (1:N)
  ├── api_keys (1:N)
  └── user_sessions (1:N)

providers
  ├── gpu_capabilities (1:N)
  ├── rental_sessions (1:N)
  └── provider_reviews (1:N)

gpu_capabilities
  ├── gpu_reservations (1:N)
  └── jobs (1:N)

rental_sessions
  ├── usage_records (1:N)
  ├── escrow_transactions (1:N)
  └── blockchain_transactions (1:N)

jobs
  ├── job_logs (1:N)
  ├── job_metrics (1:N)
  ├── job_files (1:N)
  └── job_checkpoints (1:N)
```

### Blockchain Integration

The platform integrates with Solana blockchain for payments:

- **dGPU Token**: `7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump`
- **Network**: Solana mainnet-beta
- **Escrow**: Smart contract-based escrow for secure payments
- **Platform Fee**: 5% on all transactions

### Billing Model

1. **Minute-based billing**: Usage tracked per minute
2. **Escrow system**: Funds locked before rental starts
3. **Automatic billing**: Deducted from escrow every minute
4. **Provider payout**: Automatic after rental completion
5. **Platform fee**: 5% collected on each transaction

## Performance Considerations

### Partitioning

Large tables are partitioned by time:
- `job_logs_partitioned` - Partitioned by month
- `job_metrics_partitioned` - Partitioned by month
- `audit_logs_partitioned` - Partitioned by month

### Indexing Strategy

1. **Composite indexes** for multi-column queries
2. **Partial indexes** for filtered queries (e.g., WHERE status = 'active')
3. **GIN indexes** for JSONB and full-text search
4. **Covering indexes** for index-only scans

### Query Optimization

- Statistics targets set to 1000 for frequently queried columns
- ANALYZE run on all tables after migration
- Proper foreign key constraints for referential integrity
- CHECK constraints for data validation

## Backup and Recovery

### Backup Script (Recommended)

```bash
#!/bin/bash
BACKUP_DIR="/var/backups/dantegpu"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

pg_dump -h localhost -U dante_user -d dante_core \
  -F c -b -v -f "$BACKUP_DIR/dante_core_$TIMESTAMP.backup"
```

### Restore from Backup

```bash
pg_restore -h localhost -U dante_user -d dante_core \
  -v "$BACKUP_DIR/dante_core_20240101_120000.backup"
```

## Security Best Practices

1. **Never commit `.env` files** with real credentials
2. **Use strong passwords** for database users
3. **Enable SSL/TLS** for database connections in production
4. **Restrict database access** to application servers only
5. **Regular backups** with point-in-time recovery
6. **Audit logging** enabled for all sensitive operations
7. **Encryption at rest** for sensitive data

## Monitoring

### Key Metrics to Monitor

- Connection pool utilization
- Query execution time
- Index usage statistics
- Table bloat
- Replication lag (if using replication)
- Disk space usage

### Useful Queries

```sql
-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- Check slow queries
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    max_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 20;
```

## Troubleshooting

### Migration Failed

1. Check PostgreSQL logs: `tail -f /var/log/postgresql/postgresql-15-main.log`
2. Verify database connection: `psql -h localhost -U dante_user -d dante_core`
3. Check migration file syntax
4. Ensure all dependencies are met

### Performance Issues

1. Run ANALYZE: `ANALYZE;`
2. Check for missing indexes
3. Review slow query log
4. Consider partitioning large tables
5. Optimize queries with EXPLAIN ANALYZE

## Development

### Adding New Migrations

1. Create new file: `database/migrations/008_your_migration.sql`
2. Follow naming convention: `{number}_{description}.sql`
3. Include rollback logic in comments
4. Test on development database first
5. Run migration: `./database/run_migrations.sh run`

### Testing Migrations

```bash
# Create test database
createdb dante_test

# Run migrations
DB_NAME=dante_test ./database/run_migrations.sh run

# Run tests
# ... your test commands ...

# Drop test database
dropdb dante_test
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/dante-gpu/dantegpu-core/issues
- Documentation: https://docs.dantegpu.com
- Discord: https://discord.gg/dantegpu

