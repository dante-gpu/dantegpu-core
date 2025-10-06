#!/bin/bash

# DanteGPU Platform - MinIO Bucket Setup
# Creates buckets and sets up access policies

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Setting up MinIO buckets and policies...${NC}\n"

# MinIO connection details
MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://localhost:9000}"
MINIO_ACCESS_KEY="${MINIO_ROOT_USER:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_ROOT_PASSWORD:-minioadmin}"

# Configure mc (MinIO Client)
mc alias set dantegpu "$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY"

# ============================================================================
# CREATE BUCKETS
# ============================================================================

echo -e "${GREEN}Creating buckets...${NC}"

# Job input data bucket
mc mb dantegpu/job-inputs --ignore-existing
mc versioning enable dantegpu/job-inputs

# Job output data bucket
mc mb dantegpu/job-outputs --ignore-existing
mc versioning enable dantegpu/job-outputs

# Job checkpoints bucket
mc mb dantegpu/job-checkpoints --ignore-existing
mc versioning enable dantegpu/job-checkpoints

# Job logs bucket
mc mb dantegpu/job-logs --ignore-existing

# User uploads bucket
mc mb dantegpu/user-uploads --ignore-existing
mc versioning enable dantegpu/user-uploads

# Provider data bucket
mc mb dantegpu/provider-data --ignore-existing

# System backups bucket
mc mb dantegpu/system-backups --ignore-existing
mc versioning enable dantegpu/system-backups

# Public assets bucket (for logos, avatars, etc.)
mc mb dantegpu/public-assets --ignore-existing

echo -e "${GREEN}Buckets created successfully!${NC}\n"

# ============================================================================
# SET BUCKET POLICIES
# ============================================================================

echo -e "${GREEN}Setting bucket policies...${NC}"

# Public read policy for public-assets
cat > /tmp/public-assets-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"AWS": ["*"]},
      "Action": ["s3:GetObject"],
      "Resource": ["arn:aws:s3:::public-assets/*"]
    }
  ]
}
EOF

mc anonymous set-json /tmp/public-assets-policy.json dantegpu/public-assets

# Private policy for job-inputs (authenticated users only)
cat > /tmp/job-inputs-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"AWS": ["arn:aws:iam::*:user/job-service"]},
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
      "Resource": ["arn:aws:s3:::job-inputs/*"]
    }
  ]
}
EOF

mc admin policy add dantegpu job-inputs-policy /tmp/job-inputs-policy.json

# Private policy for job-outputs
cat > /tmp/job-outputs-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"AWS": ["arn:aws:iam::*:user/job-service"]},
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
      "Resource": ["arn:aws:s3:::job-outputs/*"]
    }
  ]
}
EOF

mc admin policy add dantegpu job-outputs-policy /tmp/job-outputs-policy.json

echo -e "${GREEN}Bucket policies set successfully!${NC}\n"

# ============================================================================
# SET LIFECYCLE RULES
# ============================================================================

echo -e "${GREEN}Setting lifecycle rules...${NC}"

# Delete job logs older than 30 days
cat > /tmp/job-logs-lifecycle.json <<EOF
{
  "Rules": [
    {
      "ID": "delete-old-logs",
      "Status": "Enabled",
      "Expiration": {
        "Days": 30
      }
    }
  ]
}
EOF

mc ilm import dantegpu/job-logs < /tmp/job-logs-lifecycle.json

# Delete old checkpoints after 90 days
cat > /tmp/job-checkpoints-lifecycle.json <<EOF
{
  "Rules": [
    {
      "ID": "delete-old-checkpoints",
      "Status": "Enabled",
      "Expiration": {
        "Days": 90
      }
    }
  ]
}
EOF

mc ilm import dantegpu/job-checkpoints < /tmp/job-checkpoints-lifecycle.json

# Transition old backups to cheaper storage after 30 days, delete after 365 days
cat > /tmp/system-backups-lifecycle.json <<EOF
{
  "Rules": [
    {
      "ID": "archive-old-backups",
      "Status": "Enabled",
      "Expiration": {
        "Days": 365
      }
    }
  ]
}
EOF

mc ilm import dantegpu/system-backups < /tmp/system-backups-lifecycle.json

echo -e "${GREEN}Lifecycle rules set successfully!${NC}\n"

# ============================================================================
# CREATE SERVICE ACCOUNTS
# ============================================================================

echo -e "${GREEN}Creating service accounts...${NC}"

# Job service account
mc admin user add dantegpu job-service "${JOB_SERVICE_SECRET_KEY}"
mc admin policy set dantegpu readwrite user=job-service

# Storage service account
mc admin user add dantegpu storage-service "${STORAGE_SERVICE_SECRET_KEY}"
mc admin policy set dantegpu readwrite user=storage-service

# Backup service account
mc admin user add dantegpu backup-service "${BACKUP_SERVICE_SECRET_KEY}"
mc admin policy set dantegpu readwrite user=backup-service

echo -e "${GREEN}Service accounts created successfully!${NC}\n"

# ============================================================================
# ENABLE BUCKET NOTIFICATIONS (for event-driven processing)
# ============================================================================

echo -e "${GREEN}Setting up bucket notifications...${NC}"

# Configure NATS notification for job-outputs
mc event add dantegpu/job-outputs arn:minio:sqs::NATS:webhook \
  --event put \
  --suffix .tar.gz

# Configure NATS notification for job-inputs
mc event add dantegpu/job-inputs arn:minio:sqs::NATS:webhook \
  --event put

echo -e "${GREEN}Bucket notifications configured!${NC}\n"

# ============================================================================
# SUMMARY
# ============================================================================

echo -e "\n${BLUE}========================================${NC}"
echo -e "${GREEN}MinIO setup complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Buckets created:"
echo "  - job-inputs (versioned)"
echo "  - job-outputs (versioned)"
echo "  - job-checkpoints (versioned, 90d retention)"
echo "  - job-logs (30d retention)"
echo "  - user-uploads (versioned)"
echo "  - provider-data"
echo "  - system-backups (versioned, 365d retention)"
echo "  - public-assets (public read)"
echo ""
echo "Service accounts created:"
echo "  - job-service"
echo "  - storage-service"
echo "  - backup-service"
echo ""
echo "To view buckets:"
echo "  mc ls dantegpu"
echo ""
echo "To view bucket info:"
echo "  mc stat dantegpu/job-inputs"
echo ""

# Cleanup temp files
rm -f /tmp/*-policy.json /tmp/*-lifecycle.json

