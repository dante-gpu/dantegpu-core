#!/bin/bash

# Database Backup Script for DanteGPU Core
# Performs automated backups with retention policy

set -euo pipefail

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/dantegpu}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
S3_BUCKET="${S3_BUCKET:-s3://dantegpu-backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Database credentials from environment
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD}"

# Databases to backup
DATABASES=(
    "dante_auth"
    "dante_billing"
    "dante_registry"
    "dante_scheduler"
    "dante_core"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1"
}

# Create backup directory
mkdir -p "${BACKUP_DIR}"

# Backup each database
for db in "${DATABASES[@]}"; do
    log "Starting backup of database: ${db}"
    
    BACKUP_FILE="${BACKUP_DIR}/${db}_${TIMESTAMP}.sql.gz"
    
    # Perform backup with pg_dump
    PGPASSWORD="${DB_PASSWORD}" pg_dump \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${db}" \
        --format=custom \
        --compress=9 \
        --verbose \
        --file="${BACKUP_FILE%.gz}" \
        2>&1 | tee -a "${BACKUP_DIR}/backup_${TIMESTAMP}.log"
    
    # Compress backup
    gzip -9 "${BACKUP_FILE%.gz}"
    
    if [ -f "${BACKUP_FILE}" ]; then
        SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
        log "Backup completed: ${BACKUP_FILE} (${SIZE})"
        
        # Upload to S3
        if command -v aws &> /dev/null; then
            log "Uploading to S3: ${S3_BUCKET}/${db}/"
            aws s3 cp "${BACKUP_FILE}" "${S3_BUCKET}/${db}/${db}_${TIMESTAMP}.sql.gz" \
                --storage-class STANDARD_IA \
                --metadata "backup-date=${TIMESTAMP},database=${db}"
            
            if [ $? -eq 0 ]; then
                log "S3 upload successful"
            else
                error "S3 upload failed"
            fi
        else
            warn "AWS CLI not found, skipping S3 upload"
        fi
    else
        error "Backup file not created: ${BACKUP_FILE}"
        exit 1
    fi
done

# Backup database schemas
log "Backing up database schemas"
SCHEMA_FILE="${BACKUP_DIR}/schemas_${TIMESTAMP}.sql"
for db in "${DATABASES[@]}"; do
    PGPASSWORD="${DB_PASSWORD}" pg_dump \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${db}" \
        --schema-only \
        >> "${SCHEMA_FILE}"
done
gzip -9 "${SCHEMA_FILE}"
log "Schema backup completed: ${SCHEMA_FILE}.gz"

# Backup database roles and permissions
log "Backing up database roles"
ROLES_FILE="${BACKUP_DIR}/roles_${TIMESTAMP}.sql"
PGPASSWORD="${DB_PASSWORD}" pg_dumpall \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    --roles-only \
    > "${ROLES_FILE}"
gzip -9 "${ROLES_FILE}"
log "Roles backup completed: ${ROLES_FILE}.gz"

# Create backup manifest
MANIFEST_FILE="${BACKUP_DIR}/manifest_${TIMESTAMP}.json"
cat > "${MANIFEST_FILE}" <<EOF
{
  "timestamp": "${TIMESTAMP}",
  "date": "$(date -Iseconds)",
  "databases": [
$(for db in "${DATABASES[@]}"; do
    echo "    {\"name\": \"${db}\", \"file\": \"${db}_${TIMESTAMP}.sql.gz\"},"
done | sed '$ s/,$//')
  ],
  "schema_file": "schemas_${TIMESTAMP}.sql.gz",
  "roles_file": "roles_${TIMESTAMP}.sql.gz",
  "host": "${DB_HOST}",
  "retention_days": ${RETENTION_DAYS}
}
EOF
log "Manifest created: ${MANIFEST_FILE}"

# Upload manifest to S3
if command -v aws &> /dev/null; then
    aws s3 cp "${MANIFEST_FILE}" "${S3_BUCKET}/manifests/manifest_${TIMESTAMP}.json"
fi

# Clean up old backups
log "Cleaning up backups older than ${RETENTION_DAYS} days"
find "${BACKUP_DIR}" -name "*.sql.gz" -type f -mtime +${RETENTION_DAYS} -delete
find "${BACKUP_DIR}" -name "*.json" -type f -mtime +${RETENTION_DAYS} -delete
find "${BACKUP_DIR}" -name "*.log" -type f -mtime +${RETENTION_DAYS} -delete

# Clean up old S3 backups
if command -v aws &> /dev/null; then
    CUTOFF_DATE=$(date -d "${RETENTION_DAYS} days ago" +%Y%m%d)
    for db in "${DATABASES[@]}"; do
        aws s3 ls "${S3_BUCKET}/${db}/" | while read -r line; do
            FILE_DATE=$(echo "$line" | awk '{print $4}' | grep -oP '\d{8}' | head -1)
            if [ "${FILE_DATE}" -lt "${CUTOFF_DATE}" ]; then
                FILE_NAME=$(echo "$line" | awk '{print $4}')
                log "Deleting old S3 backup: ${FILE_NAME}"
                aws s3 rm "${S3_BUCKET}/${db}/${FILE_NAME}"
            fi
        done
    done
fi

# Verify backups
log "Verifying backups"
for db in "${DATABASES[@]}"; do
    BACKUP_FILE="${BACKUP_DIR}/${db}_${TIMESTAMP}.sql.gz"
    if [ -f "${BACKUP_FILE}" ]; then
        # Test gzip integrity
        if gzip -t "${BACKUP_FILE}" 2>/dev/null; then
            log "Backup verified: ${BACKUP_FILE}"
        else
            error "Backup verification failed: ${BACKUP_FILE}"
            exit 1
        fi
    fi
done

# Send notification
if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
    TOTAL_SIZE=$(du -sh "${BACKUP_DIR}" | cut -f1)
    curl -X POST "${SLACK_WEBHOOK_URL}" \
        -H 'Content-Type: application/json' \
        -d "{
            \"text\": \"✅ Database backup completed successfully\",
            \"attachments\": [{
                \"color\": \"good\",
                \"fields\": [
                    {\"title\": \"Timestamp\", \"value\": \"${TIMESTAMP}\", \"short\": true},
                    {\"title\": \"Databases\", \"value\": \"${#DATABASES[@]}\", \"short\": true},
                    {\"title\": \"Total Size\", \"value\": \"${TOTAL_SIZE}\", \"short\": true},
                    {\"title\": \"Retention\", \"value\": \"${RETENTION_DAYS} days\", \"short\": true}
                ]
            }]
        }"
fi

log "Backup process completed successfully"
exit 0

