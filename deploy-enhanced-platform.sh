#!/bin/bash

# Enhanced DanteGPU Platform Deployment Script
# This script deploys the complete GPU rental platform with all new components

set -e

echo "🚀 Starting Enhanced DanteGPU Platform Deployment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
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

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    # Check if Docker is running
    if ! docker info &> /dev/null; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    
    # Check available disk space (minimum 10GB)
    available_space=$(df . | tail -1 | awk '{print $4}')
    if [ "$available_space" -lt 10485760 ]; then
        print_warning "Less than 10GB disk space available. Deployment may fail."
    fi
    
    print_success "Prerequisites check completed"
}

# Setup environment variables
setup_environment() {
    print_status "Setting up environment variables..."
    
    if [ ! -f .env ]; then
        if [ -f env.production.example ]; then
            cp env.production.example .env
            print_success "Created .env file from example"
        else
            print_error ".env file not found and no example available"
            exit 1
        fi
    fi
    
    # Source environment variables
    source .env
    
    print_success "Environment variables loaded"
}

# Build Docker images for new services
build_new_services() {
    print_status "Building new service Docker images..."
    
    # Build Kubernetes Scheduler Extender
    if [ -d "scheduler-orchestrator-service" ]; then
        print_status "Building Kubernetes Scheduler Extender..."
        cd scheduler-orchestrator-service
        docker build -t dante/scheduler-extender:latest .
        cd ..
        print_success "Scheduler Extender built"
    fi
    
    # Build GPU Monitoring Service
    if [ -d "gpu-monitoring-service" ]; then
        print_status "Building GPU Monitoring Service..."
        cd gpu-monitoring-service
        cat > Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gpu-monitoring main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/gpu-monitoring .
EXPOSE 8095
CMD ["./gpu-monitoring"]
EOF
        docker build -t dante/gpu-monitoring:latest .
        cd ..
        print_success "GPU Monitoring Service built"
    fi
    
    # Build Redis Cache Service
    if [ -d "redis-cache-service" ]; then
        print_status "Building Redis Cache Service..."
        cd redis-cache-service
        cat > Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o redis-cache main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/redis-cache .
EXPOSE 8097
CMD ["./redis-cache"]
EOF
        docker build -t dante/redis-cache:latest .
        cd ..
        print_success "Redis Cache Service built"
    fi
    
    # Build Ory Kratos Integration
    if [ -d "ory-kratos-integration" ]; then
        print_status "Building Ory Kratos Integration..."
        cd ory-kratos-integration
        cat > Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o kratos-integration main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/kratos-integration .
EXPOSE 8098
CMD ["./kratos-integration"]
EOF
        docker build -t dante/kratos-integration:latest .
        cd ..
        print_success "Ory Kratos Integration built"
    fi
    
    # Build Stripe/PayPal Integration
    if [ -d "stripe-paypal-integration" ]; then
        print_status "Building Stripe/PayPal Integration..."
        cd stripe-paypal-integration
        cat > Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o payment-integration main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/payment-integration .
EXPOSE 8096
CMD ["./payment-integration"]
EOF
        docker build -t dante/payment-integration:latest .
        cd ..
        print_success "Stripe/PayPal Integration built"
    fi
    
    # Build Enhanced Frontend
    if [ -d "gpu-rental-frontend" ]; then
        print_status "Building Enhanced Frontend..."
        cd gpu-rental-frontend
        cat > Dockerfile << 'EOF'
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
EOF
        
        # Create nginx config
        cat > nginx.conf << 'EOF'
events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    
    server {
        listen 80;
        server_name localhost;
        root /usr/share/nginx/html;
        index index.html;
        
        location / {
            try_files $uri $uri/ /index.html;
        }
        
        location /api/ {
            proxy_pass http://api-gateway:8080;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }
    }
}
EOF
        
        docker build -t dante/gpu-rental-frontend:latest .
        cd ..
        print_success "Enhanced Frontend built"
    fi
}

# Create enhanced docker-compose file
create_enhanced_compose() {
    print_status "Creating enhanced docker-compose configuration..."
    
    cat > docker-compose.enhanced.yml << 'EOF'
version: '3.8'

services:
  # Infrastructure Services
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./scripts/db_setup:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 30s
      timeout: 10s
      retries: 3

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 30s
      timeout: 10s
      retries: 3

  nats:
    image: nats:2.9-alpine
    ports:
      - "4222:4222"
      - "8222:8222"
    command: ["--jetstream", "--store_dir=/data"]
    volumes:
      - nats_data:/data

  consul:
    image: hashicorp/consul:1.17
    ports:
      - "8500:8500"
    volumes:
      - consul_data:/consul/data
    command: ["consul", "agent", "-dev", "-client=0.0.0.0"]

  minio:
    image: minio/minio:latest
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - minio_data:/data
    command: server /data --console-address ":9001"

  # Enhanced Services
  gpu-monitoring:
    image: dante/gpu-monitoring:latest
    ports:
      - "8095:8095"
    depends_on:
      - redis
      - nats
    environment:
      - REDIS_URL=redis:6379
      - NATS_URL=nats://nats:4222

  redis-cache:
    image: dante/redis-cache:latest
    ports:
      - "8097:8097"
    depends_on:
      - redis
    environment:
      - REDIS_URL=redis:6379

  kratos-integration:
    image: dante/kratos-integration:latest
    ports:
      - "8098:8098"
    depends_on:
      - redis
    environment:
      - KRATOS_PUBLIC_URL=http://localhost:4433
      - KRATOS_ADMIN_URL=http://localhost:4434
      - REDIS_URL=redis:6379

  payment-integration:
    image: dante/payment-integration:latest
    ports:
      - "8096:8096"
    environment:
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
      - PAYPAL_CLIENT_ID=${PAYPAL_CLIENT_ID}
      - PAYPAL_CLIENT_SECRET=${PAYPAL_CLIENT_SECRET}

  # Original Services (Enhanced)
  api-gateway:
    build: ./api-gateway
    ports:
      - "8080:8080"
    depends_on:
      - consul
      - nats
      - redis-cache
      - kratos-integration
    environment:
      - CONSUL_URL=consul:8500
      - NATS_URL=nats://nats:4222
      - REDIS_URL=redis:6379

  auth-service:
    build: ./auth-service
    ports:
      - "8090:8090"
    depends_on:
      - postgres
      - redis
    environment:
      - DATABASE_URL=postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}
      - REDIS_URL=redis:6379

  billing-payment-service:
    build: ./billing-payment-service
    ports:
      - "8082:8082"
    depends_on:
      - postgres
      - payment-integration
    environment:
      - DATABASE_URL=postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}
      - PAYMENT_SERVICE_URL=http://payment-integration:8096

  provider-registry-service:
    build: ./provider-registry-service
    ports:
      - "8081:8081"
    depends_on:
      - postgres
      - gpu-monitoring
    environment:
      - DATABASE_URL=postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}
      - GPU_MONITORING_URL=http://gpu-monitoring:8095

  scheduler-orchestrator-service:
    image: dante/scheduler-extender:latest
    ports:
      - "8084:8084"
    depends_on:
      - postgres
      - nats
      - gpu-monitoring
    environment:
      - DATABASE_URL=postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}
      - NATS_URL=nats://nats:4222
      - GPU_MONITORING_URL=http://gpu-monitoring:8095

  storage-service:
    build: ./storage-service
    ports:
      - "8083:8083"
    depends_on:
      - minio
    environment:
      - MINIO_ENDPOINT=minio:9000
      - MINIO_ACCESS_KEY=${MINIO_ROOT_USER}
      - MINIO_SECRET_KEY=${MINIO_ROOT_PASSWORD}

  # Enhanced Frontend
  frontend:
    image: dante/gpu-rental-frontend:latest
    ports:
      - "3000:80"
    depends_on:
      - api-gateway

  # Monitoring Stack
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring-logging-service/prometheus:/etc/prometheus
      - prometheus_data:/prometheus

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring-logging-service/grafana:/etc/grafana/provisioning

volumes:
  postgres_data:
  redis_data:
  nats_data:
  consul_data:
  minio_data:
  prometheus_data:
  grafana_data:
EOF

    print_success "Enhanced docker-compose configuration created"
}

# Initialize database with extended schema
init_database() {
    print_status "Initializing database with extended schema..."
    
    # Wait for PostgreSQL to be ready
    print_status "Waiting for PostgreSQL to be ready..."
    sleep 30
    
    # Run extended schema
    if [ -f "scripts/db_setup/01_extended_schema.sql" ]; then
        docker-compose -f docker-compose.enhanced.yml exec -T postgres psql -U ${POSTGRES_USER} -f /docker-entrypoint-initdb.d/01_extended_schema.sql
        print_success "Extended database schema applied"
    fi
}

# Deploy the platform
deploy_platform() {
    print_status "Deploying enhanced platform..."
    
    # Stop any existing services
    docker-compose -f docker-compose.enhanced.yml down 2>/dev/null || true
    
    # Start infrastructure services first
    print_status "Starting infrastructure services..."
    docker-compose -f docker-compose.enhanced.yml up -d postgres redis nats consul minio
    
    # Wait for infrastructure to be ready
    sleep 60
    
    # Start enhanced services
    print_status "Starting enhanced services..."
    docker-compose -f docker-compose.enhanced.yml up -d gpu-monitoring redis-cache kratos-integration payment-integration
    
    # Wait for enhanced services
    sleep 30
    
    # Start core services
    print_status "Starting core services..."
    docker-compose -f docker-compose.enhanced.yml up -d auth-service billing-payment-service provider-registry-service scheduler-orchestrator-service storage-service
    
    # Wait for core services
    sleep 30
    
    # Start API gateway and frontend
    print_status "Starting API gateway and frontend..."
    docker-compose -f docker-compose.enhanced.yml up -d api-gateway frontend
    
    # Start monitoring
    print_status "Starting monitoring services..."
    docker-compose -f docker-compose.enhanced.yml up -d prometheus grafana
    
    print_success "Enhanced platform deployed successfully!"
}

# Health check
health_check() {
    print_status "Performing health checks..."
    
    services=(
        "http://localhost:8080/health:API Gateway"
        "http://localhost:8090/health:Auth Service"
        "http://localhost:8095/health:GPU Monitoring"
        "http://localhost:8096/health:Payment Integration"
        "http://localhost:8097/health:Redis Cache"
        "http://localhost:8098/health:Kratos Integration"
        "http://localhost:3000:Frontend"
        "http://localhost:9090:Prometheus"
        "http://localhost:3001:Grafana"
    )
    
    for service in "${services[@]}"; do
        url=$(echo $service | cut -d: -f1-2)
        name=$(echo $service | cut -d: -f3)
        
        if curl -s "$url" > /dev/null 2>&1; then
            print_success "$name is healthy"
        else
            print_warning "$name is not responding"
        fi
    done
}

# Display service URLs
display_urls() {
    print_success "🎉 Enhanced DanteGPU Platform is ready!"
    echo ""
    echo "📊 Service URLs:"
    echo "  • Frontend:              http://localhost:3000"
    echo "  • API Gateway:           http://localhost:8080"
    echo "  • GPU Monitoring:        http://localhost:8095"
    echo "  • Payment Integration:   http://localhost:8096"
    echo "  • Kratos Integration:    http://localhost:8098"
    echo "  • Grafana:              http://localhost:3001 (admin/admin)"
    echo "  • Prometheus:           http://localhost:9090"
    echo "  • MinIO Console:        http://localhost:9001"
    echo "  • Consul UI:            http://localhost:8500"
    echo ""
    echo "🔧 Enhanced Features:"
    echo "  ✅ Kubernetes Scheduler Extender"
    echo "  ✅ GPU Fractional Allocation"
    echo "  ✅ Real-time GPU Monitoring"
    echo "  ✅ Advanced Redis Caching"
    echo "  ✅ Ory Kratos Authentication"
    echo "  ✅ Stripe/PayPal Payments"
    echo "  ✅ Extended Database Schema"
    echo "  ✅ Enhanced React Frontend"
    echo ""
    echo "📚 Documentation:"
    echo "  • Architecture:         ./ARCHITECTURE.md"
    echo "  • Deployment Guide:     ./DEPLOYMENT_GUIDE.md"
    echo "  • Development Roadmap:  ./DEVELOPMENT_ROADMAP.md"
}

# Main execution
main() {
    echo "🚀 Enhanced DanteGPU Platform Deployment"
    echo "========================================"
    
    check_prerequisites
    setup_environment
    build_new_services
    create_enhanced_compose
    deploy_platform
    init_database
    
    # Wait a bit for services to fully start
    sleep 30
    
    health_check
    display_urls
    
    print_success "Deployment completed successfully! 🎉"
}

# Run main function
main "$@"
