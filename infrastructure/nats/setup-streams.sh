#!/bin/bash

# DanteGPU Platform - NATS JetStream Setup
# This script creates all required streams and consumers

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Setting up NATS JetStream streams and consumers...${NC}\n"

# NATS connection details
NATS_URL="${NATS_URL:-nats://localhost:4222}"
NATS_USER="${NATS_SYSTEM_USER:-admin}"
NATS_PASS="${NATS_SYSTEM_PASSWORD:-password}"

# ============================================================================
# JOBS STREAM - For job lifecycle events
# ============================================================================

echo -e "${GREEN}Creating JOBS stream...${NC}"
nats stream add JOBS \
    --subjects "jobs.>" \
    --storage file \
    --retention limits \
    --max-msgs=-1 \
    --max-bytes=10GB \
    --max-age=30d \
    --max-msg-size=1MB \
    --discard old \
    --dupe-window=2m \
    --replicas=3 \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# Create consumers for jobs stream
echo "Creating job consumers..."

# Scheduler consumer - processes new job submissions
nats consumer add JOBS scheduler \
    --filter "jobs.submitted" \
    --deliver all \
    --ack explicit \
    --max-deliver=3 \
    --max-pending=100 \
    --replay instant \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# Provider consumer - receives job assignments
nats consumer add JOBS provider \
    --filter "jobs.assigned.>" \
    --deliver all \
    --ack explicit \
    --max-deliver=3 \
    --max-pending=50 \
    --replay instant \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# ============================================================================
# EVENTS STREAM - For platform-wide events
# ============================================================================

echo -e "\n${GREEN}Creating EVENTS stream...${NC}"
nats stream add EVENTS \
    --subjects "events.>" \
    --storage file \
    --retention limits \
    --max-msgs=-1 \
    --max-bytes=5GB \
    --max-age=90d \
    --max-msg-size=512KB \
    --discard old \
    --dupe-window=5m \
    --replicas=3 \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# Create consumers for events stream
echo "Creating event consumers..."

# Notification consumer
nats consumer add EVENTS notifications \
    --filter "events.user.>" \
    --deliver all \
    --ack explicit \
    --max-deliver=5 \
    --max-pending=1000 \
    --replay instant \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# Analytics consumer
nats consumer add EVENTS analytics \
    --filter "events.>" \
    --deliver all \
    --ack explicit \
    --max-deliver=3 \
    --max-pending=500 \
    --replay instant \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# ============================================================================
# METRICS STREAM - For GPU and system metrics
# ============================================================================

echo -e "\n${GREEN}Creating METRICS stream...${NC}"
nats stream add METRICS \
    --subjects "metrics.>" \
    --storage file \
    --retention limits \
    --max-msgs=-1 \
    --max-bytes=20GB \
    --max-age=7d \
    --max-msg-size=256KB \
    --discard old \
    --dupe-window=1m \
    --replicas=3 \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# Create consumers for metrics stream
echo "Creating metrics consumers..."

# Monitoring consumer
nats consumer add METRICS monitoring \
    --filter "metrics.>" \
    --deliver all \
    --ack explicit \
    --max-deliver=2 \
    --max-pending=2000 \
    --replay instant \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# ============================================================================
# BILLING STREAM - For billing and payment events
# ============================================================================

echo -e "\n${GREEN}Creating BILLING stream...${NC}"
nats stream add BILLING \
    --subjects "billing.>,payments.>" \
    --storage file \
    --retention limits \
    --max-msgs=-1 \
    --max-bytes=5GB \
    --max-age=365d \
    --max-msg-size=512KB \
    --discard old \
    --dupe-window=10m \
    --replicas=3 \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# Create consumers for billing stream
echo "Creating billing consumers..."

# Billing processor consumer
nats consumer add BILLING processor \
    --filter "billing.>" \
    --deliver all \
    --ack explicit \
    --max-deliver=5 \
    --max-pending=100 \
    --replay instant \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# Payment processor consumer
nats consumer add BILLING payments \
    --filter "payments.>" \
    --deliver all \
    --ack explicit \
    --max-deliver=5 \
    --max-pending=100 \
    --replay instant \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# ============================================================================
# LOGS STREAM - For application logs
# ============================================================================

echo -e "\n${GREEN}Creating LOGS stream...${NC}"
nats stream add LOGS \
    --subjects "logs.>" \
    --storage file \
    --retention limits \
    --max-msgs=-1 \
    --max-bytes=50GB \
    --max-age=30d \
    --max-msg-size=1MB \
    --discard old \
    --dupe-window=30s \
    --replicas=3 \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# Create consumers for logs stream
echo "Creating log consumers..."

# Log aggregator consumer
nats consumer add LOGS aggregator \
    --filter "logs.>" \
    --deliver all \
    --ack explicit \
    --max-deliver=2 \
    --max-pending=5000 \
    --replay instant \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# ============================================================================
# PROVIDER STREAM - For provider heartbeats and status
# ============================================================================

echo -e "\n${GREEN}Creating PROVIDER stream...${NC}"
nats stream add PROVIDER \
    --subjects "provider.>" \
    --storage file \
    --retention limits \
    --max-msgs=-1 \
    --max-bytes=2GB \
    --max-age=7d \
    --max-msg-size=256KB \
    --discard old \
    --dupe-window=1m \
    --replicas=3 \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# Create consumers for provider stream
echo "Creating provider consumers..."

# Registry consumer
nats consumer add PROVIDER registry \
    --filter "provider.>" \
    --deliver all \
    --ack explicit \
    --max-deliver=3 \
    --max-pending=500 \
    --replay instant \
    --server="$NATS_URL" \
    --user="$NATS_USER" \
    --password="$NATS_PASS"

# ============================================================================
# Summary
# ============================================================================

echo -e "\n${BLUE}========================================${NC}"
echo -e "${GREEN}NATS JetStream setup complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Streams created:"
echo "  - JOBS (job lifecycle)"
echo "  - EVENTS (platform events)"
echo "  - METRICS (GPU metrics)"
echo "  - BILLING (payments)"
echo "  - LOGS (application logs)"
echo "  - PROVIDER (provider status)"
echo ""
echo "To view stream status:"
echo "  nats stream ls"
echo "  nats stream info JOBS"
echo ""
echo "To view consumers:"
echo "  nats consumer ls JOBS"
echo ""

