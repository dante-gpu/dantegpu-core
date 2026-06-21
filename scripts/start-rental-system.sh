#!/bin/bash

# DanteGPU Rental System - Complete Startup Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                                                            ║${NC}"
echo -e "${BLUE}║         🚀 DanteGPU Rental System Startup 🚀              ║${NC}"
echo -e "${BLUE}║                                                            ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Function to print section headers
print_section() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to wait for service
wait_for_service() {
    local service=$1
    local host=$2
    local port=$3
    local max_attempts=30
    local attempt=0

    echo -e "${YELLOW}⏳ Waiting for $service to be ready...${NC}"
    
    while [ $attempt -lt $max_attempts ]; do
        if nc -z "$host" "$port" 2>/dev/null; then
            echo -e "${GREEN}✅ $service is ready!${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    
    echo -e "${RED}❌ $service failed to start${NC}"
    return 1
}

# Step 1: Check prerequisites
print_section "1️⃣  Checking Prerequisites"

if ! command_exists docker; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker is installed${NC}"

if ! command_exists docker-compose; then
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker Compose is installed${NC}"

if ! command_exists go; then
    echo -e "${YELLOW}⚠️  Go is not installed (needed for backend services)${NC}"
else
    echo -e "${GREEN}✅ Go is installed ($(go version))${NC}"
fi

if ! command_exists node; then
    echo -e "${YELLOW}⚠️  Node.js is not installed (needed for frontend)${NC}"
else
    echo -e "${GREEN}✅ Node.js is installed ($(node --version))${NC}"
fi

# Step 2: Setup environment
print_section "2️⃣  Setting Up Environment"

cd "$PROJECT_ROOT"

if [ ! -f ".env" ]; then
    echo -e "${YELLOW}📝 Creating .env file from .env.example...${NC}"
    cp .env.example .env
    
    # Update with development defaults
    sed -i.bak 's/your_secure_password_here/dante_password/g' .env
    sed -i.bak 's/your_redis_password_here//g' .env
    sed -i.bak 's/your_jwt_secret_here/dev_jwt_secret_change_in_production/g' .env
    sed -i.bak 's/your_solana_private_key_here/DEVELOPMENT_KEY_ONLY/g' .env
    rm -f .env.bak
    
    echo -e "${GREEN}✅ .env file created${NC}"
else
    echo -e "${GREEN}✅ .env file already exists${NC}"
fi

# Step 3: Start infrastructure services
print_section "3️⃣  Starting Infrastructure Services"

echo -e "${BLUE}🐳 Starting Docker Compose services...${NC}"
docker-compose up -d postgres redis nats consul

# Wait for services
wait_for_service "PostgreSQL" "localhost" "5432"
wait_for_service "Redis" "localhost" "6379"
wait_for_service "NATS" "localhost" "4222"
wait_for_service "Consul" "localhost" "8500"

# Step 4: Initialize databases
print_section "4️⃣  Initializing Databases"

echo -e "${BLUE}📊 Creating databases...${NC}"
sleep 5  # Give PostgreSQL time to fully initialize

# Check if databases exist
PGPASSWORD=dante_password psql -h localhost -U dante_user -d postgres -c "SELECT 1" >/dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Database connection successful${NC}"
    
    # Create databases if they don't exist
    for db in dante_auth dante_billing dante_registry dante_scheduler dante_core; do
        PGPASSWORD=dante_password psql -h localhost -U dante_user -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$db'" | grep -q 1
        if [ $? -ne 0 ]; then
            echo -e "${YELLOW}📝 Creating database: $db${NC}"
            PGPASSWORD=dante_password psql -h localhost -U dante_user -d postgres -c "CREATE DATABASE $db;"
        else
            echo -e "${GREEN}✅ Database $db already exists${NC}"
        fi
    done
else
    echo -e "${RED}❌ Cannot connect to PostgreSQL${NC}"
    exit 1
fi

# Run migrations
if [ -d "database/migrations" ]; then
    echo -e "${BLUE}🔄 Running database migrations...${NC}"
    # This would run migrations - for now we'll skip if no migration tool
    echo -e "${YELLOW}⚠️  Migration tool not configured - skipping${NC}"
fi

# Step 5: Setup NATS streams
print_section "5️⃣  Setting Up NATS Streams"

if [ -f "setup-nats-streams.sh" ]; then
    echo -e "${BLUE}📡 Creating NATS JetStream streams...${NC}"
    bash setup-nats-streams.sh || echo -e "${YELLOW}⚠️  NATS streams setup skipped${NC}"
else
    echo -e "${YELLOW}⚠️  NATS setup script not found - skipping${NC}"
fi

# Step 6: Start monitoring services
print_section "6️⃣  Starting Monitoring Services"

echo -e "${BLUE}📊 Starting Prometheus, Grafana, Loki...${NC}"
docker-compose up -d prometheus grafana loki

sleep 3
echo -e "${GREEN}✅ Monitoring services started${NC}"

# Step 7: Display service status
print_section "7️⃣  Service Status"

echo -e "${BLUE}📋 Running services:${NC}"
docker-compose ps

# Step 8: Display access information
print_section "8️⃣  Access Information"

echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                    🎉 SYSTEM READY! 🎉                     ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}📍 Infrastructure Services:${NC}"
echo -e "   PostgreSQL:    ${GREEN}localhost:5432${NC}"
echo -e "   Redis:         ${GREEN}localhost:6379${NC}"
echo -e "   NATS:          ${GREEN}localhost:4222${NC}"
echo -e "   Consul:        ${GREEN}http://localhost:8500${NC}"
echo ""
echo -e "${BLUE}📊 Monitoring:${NC}"
echo -e "   Prometheus:    ${GREEN}http://localhost:9090${NC}"
echo -e "   Grafana:       ${GREEN}http://localhost:3000${NC} (admin/admin)"
echo -e "   Loki:          ${GREEN}http://localhost:3100${NC}"
echo ""
echo -e "${BLUE}🔧 Next Steps:${NC}"
echo -e "   1. Start backend services: ${YELLOW}cd api-gateway && go run cmd/main.go${NC}"
echo -e "   2. Start auth service:     ${YELLOW}cd auth-service && python main.py${NC}"
echo -e "   3. Start frontend:         ${YELLOW}cd user-dashboard && npm run dev${NC}"
echo ""
echo -e "${BLUE}📝 Useful Commands:${NC}"
echo -e "   Stop all:       ${YELLOW}docker-compose down${NC}"
echo -e "   View logs:      ${YELLOW}docker-compose logs -f [service]${NC}"
echo -e "   Restart:        ${YELLOW}docker-compose restart [service]${NC}"
echo ""
echo -e "${GREEN}✨ Infrastructure is ready for development!${NC}"
echo ""

