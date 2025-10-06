#!/bin/bash

# DanteGPU Platform - Database Migration Runner
# This script runs all database migrations in order

set -e  # Exit on error
set -u  # Exit on undefined variable

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="${SCRIPT_DIR}/migrations"
ENV_FILE="${SCRIPT_DIR}/../.env"

# Default database connection parameters
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-dante_core}"
DB_USER="${DB_USER:-dante_user}"
DB_PASSWORD="${DB_PASSWORD:-}"

# Load environment variables if .env file exists
if [ -f "$ENV_FILE" ]; then
    echo -e "${BLUE}Loading environment variables from .env file...${NC}"
    export $(grep -v '^#' "$ENV_FILE" | xargs)
fi

# Function to print colored messages
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if PostgreSQL is available
check_postgres() {
    print_info "Checking PostgreSQL connection..."
    
    if ! command -v psql &> /dev/null; then
        print_error "psql command not found. Please install PostgreSQL client."
        exit 1
    fi
    
    if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c '\q' 2>/dev/null; then
        print_error "Cannot connect to PostgreSQL at $DB_HOST:$DB_PORT"
        print_error "Please check your database connection settings."
        exit 1
    fi
    
    print_success "PostgreSQL connection successful"
}

# Function to create database if it doesn't exist
create_database() {
    print_info "Checking if database '$DB_NAME' exists..."
    
    DB_EXISTS=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'")
    
    if [ "$DB_EXISTS" != "1" ]; then
        print_info "Creating database '$DB_NAME'..."
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME;"
        print_success "Database '$DB_NAME' created"
    else
        print_info "Database '$DB_NAME' already exists"
    fi
}

# Function to create migrations tracking table
create_migrations_table() {
    print_info "Creating migrations tracking table..."
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    version VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    execution_time_ms INTEGER,
    checksum VARCHAR(64)
);

CREATE INDEX IF NOT EXISTS idx_schema_migrations_version ON schema_migrations(version);
EOF
    
    print_success "Migrations tracking table ready"
}

# Function to calculate file checksum
calculate_checksum() {
    local file="$1"
    if command -v sha256sum &> /dev/null; then
        sha256sum "$file" | awk '{print $1}'
    elif command -v shasum &> /dev/null; then
        shasum -a 256 "$file" | awk '{print $1}'
    else
        print_warning "No checksum utility found, skipping checksum verification"
        echo ""
    fi
}

# Function to check if migration has been applied
is_migration_applied() {
    local version="$1"
    local count=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM schema_migrations WHERE version='$version'")
    [ "$count" -gt 0 ]
}

# Function to run a single migration
run_migration() {
    local migration_file="$1"
    local version=$(basename "$migration_file" .sql)
    local name=$(echo "$version" | sed 's/^[0-9]*_//')
    
    print_info "Running migration: $version"
    
    # Check if already applied
    if is_migration_applied "$version"; then
        print_warning "Migration $version already applied, skipping..."
        return 0
    fi
    
    # Calculate checksum
    local checksum=$(calculate_checksum "$migration_file")
    
    # Record start time
    local start_time=$(date +%s%3N)
    
    # Run migration in a transaction
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -f "$migration_file"; then
        # Record end time
        local end_time=$(date +%s%3N)
        local execution_time=$((end_time - start_time))
        
        # Record migration in tracking table
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
INSERT INTO schema_migrations (version, name, execution_time_ms, checksum)
VALUES ('$version', '$name', $execution_time, '$checksum');
EOF
        
        print_success "Migration $version completed in ${execution_time}ms"
        return 0
    else
        print_error "Migration $version failed!"
        return 1
    fi
}

# Function to run all migrations
run_all_migrations() {
    print_info "Starting database migrations..."
    echo ""
    
    local migration_count=0
    local failed_count=0
    
    # Get all migration files sorted by version
    for migration_file in $(ls -1 "$MIGRATIONS_DIR"/*.sql 2>/dev/null | sort -V); do
        if [ -f "$migration_file" ]; then
            migration_count=$((migration_count + 1))
            
            if ! run_migration "$migration_file"; then
                failed_count=$((failed_count + 1))
                print_error "Migration failed, stopping..."
                break
            fi
            echo ""
        fi
    done
    
    echo ""
    print_info "Migration Summary:"
    print_info "  Total migrations found: $migration_count"
    print_info "  Failed migrations: $failed_count"
    
    if [ $failed_count -eq 0 ]; then
        print_success "All migrations completed successfully!"
        return 0
    else
        print_error "Some migrations failed!"
        return 1
    fi
}

# Function to show migration status
show_status() {
    print_info "Migration Status:"
    echo ""
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
SELECT 
    version,
    name,
    executed_at,
    execution_time_ms || 'ms' as execution_time
FROM schema_migrations
ORDER BY id;
EOF
}

# Function to rollback last migration
rollback_last() {
    print_warning "Rollback functionality not implemented yet."
    print_warning "Please manually rollback using SQL scripts."
}

# Main script
main() {
    echo ""
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║         DanteGPU Platform - Database Migrations           ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""
    
    # Parse command line arguments
    case "${1:-run}" in
        run)
            check_postgres
            create_database
            create_migrations_table
            run_all_migrations
            ;;
        status)
            check_postgres
            show_status
            ;;
        rollback)
            check_postgres
            rollback_last
            ;;
        *)
            echo "Usage: $0 {run|status|rollback}"
            echo ""
            echo "Commands:"
            echo "  run      - Run all pending migrations (default)"
            echo "  status   - Show migration status"
            echo "  rollback - Rollback last migration (not implemented)"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"

