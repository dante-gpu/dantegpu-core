# DanteGPU - Decentralized GPU Rental Platform

## Overview

DanteGPU is a blockchain-based decentralized GPU rental platform that enables GPU owners to monetize their idle resources while providing users with flexible access to GPU computing power for machine learning, AI workloads, and high-performance computing tasks.

## Table of Contents

- [Core Features](#core-features)
- [Architecture Overview](#architecture-overview)
- [Quick Start](#quick-start)
- [Service Details](#service-details)
- [Development](#development)
- [Monitoring and Logging](#monitoring-and-logging)
- [Security](#security)
- [Deployment](#deployment)
- [Performance](#performance)
- [API Documentation](#api-documentation)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)

## Core Features

### Blockchain Integration
- **Solana Network**: Native integration with Solana blockchain for payments
- **dGPU Token**: Custom token for platform transactions and rewards
- **Smart Contracts**: Automated escrow and payment distribution
- **Wallet Integration**: Support for major Solana wallets

### Real-time Billing System
- **Minute-based Tracking**: Precise usage monitoring and billing
- **Dynamic Pricing**: Automatic price adjustment based on demand and GPU specifications
- **Cost Estimation**: Real-time cost calculation for users
- **Payment Processing**: Automated blockchain-based payments

### Multi-GPU Support
- **NVIDIA GPUs**: Full support for CUDA-enabled graphics cards
- **AMD GPUs**: ROCm-compatible AMD graphics processing units
- **Apple Silicon**: M1, M2, and M3 chip support for macOS users
- **Hardware Detection**: Automatic GPU discovery and capability assessment

### Comprehensive Monitoring
- **Prometheus**: Metrics collection and storage
- **Grafana**: Real-time dashboards and visualization
- **Loki**: Centralized logging and log aggregation
- **AlertManager**: Automated alerting and notification system

### Security Framework
- **JWT Authentication**: Secure token-based authentication system
- **Role-based Access Control**: Granular permission management
- **API Security**: Rate limiting, request validation, and CORS policies
- **Blockchain Security**: Private key encryption and transaction verification

### Modern User Interface
- **React Frontend**: Responsive web application for users
- **Tauri Desktop App**: Cross-platform desktop application for providers
- **RESTful APIs**: Well-documented API endpoints for integration
- **Real-time Updates**: WebSocket connections for live status updates

## Architecture Overview

The platform is built using a microservices architecture consisting of the following layers:

### Infrastructure Layer

#### Database and Storage
- **PostgreSQL**: Primary relational database for user data, transactions, and metadata
- **Redis**: High-performance caching layer and rate limiting store
- **MinIO**: S3-compatible object storage for files and artifacts

#### Message Queue and Service Discovery
- **NATS JetStream**: Event streaming and message queue for inter-service communication
- **Consul**: Service discovery, configuration management, and health checking

### Microservices Layer

#### Core Services
- **API Gateway**: Central entry point for all client requests with routing and authentication
- **Auth Service**: User authentication, authorization, and JWT token management
- **Billing Service**: Blockchain payment processing and usage-based billing
- **Provider Registry**: GPU provider registration, status tracking, and capability management
- **Scheduler Orchestrator**: Job scheduling, container orchestration, and resource allocation
- **Storage Service**: File upload, download, and management with S3-compatible interface

#### Supporting Services
- **Monitoring Services**: System health monitoring, metrics collection, and alerting
- **Notification Service**: Email, SMS, and push notification delivery
- **Analytics Service**: Usage analytics, reporting, and business intelligence

### Client Applications

#### Web Applications
- **User Dashboard**: React-based web interface for GPU rental and management
- **Provider Portal**: Web interface for GPU providers to manage their resources
- **Admin Console**: Administrative interface for platform management

#### Desktop Applications
- **Provider GUI**: Tauri-based cross-platform desktop application for GPU providers
- **Provider Daemon**: Background service for GPU monitoring and job execution

## Quick Start

### System Requirements

#### Minimum Requirements
- **Operating System**: Linux (Ubuntu 20.04+), macOS (10.15+), or Windows 10+
- **Docker**: Version 20.10+ with Docker Compose
- **Memory**: 8GB RAM minimum (16GB recommended)
- **Storage**: 20GB free disk space
- **Network**: Stable internet connection for blockchain operations

#### Recommended Specifications
- **CPU**: 4+ cores
- **Memory**: 16GB+ RAM
- **Storage**: SSD with 50GB+ free space
- **Network**: High-speed internet connection

### Installation

#### 1. Repository Setup
```bash
# Clone the repository
git clone https://github.com/dante-gpu/dantegpu-core.git
cd dantegpu-core

# Verify system requirements
./scripts/check-requirements.sh
```

#### 2. Environment Configuration
```bash
# Copy environment template
cp env.production.example .env

# Configure environment variables
nano .env
```

**Required Environment Variables:**
```bash
# Database Configuration
POSTGRES_DB=dantegpu
POSTGRES_USER=dantegpu_user
POSTGRES_PASSWORD=secure_password

# Redis Configuration
REDIS_PASSWORD=redis_secure_password

# Blockchain Configuration
SOLANA_NETWORK=mainnet-beta
SOLANA_RPC_URL=https://api.mainnet-beta.solana.com

# JWT Configuration
JWT_SECRET=your_jwt_secret_key
JWT_EXPIRY=24h

# MinIO Configuration
MINIO_ROOT_USER=admin
MINIO_ROOT_PASSWORD=secure_minio_password
```

#### 3. Platform Deployment
```bash
# Deploy all services
./deploy-production.sh

# Verify deployment
./scripts/health-check.sh
```

### Service Endpoints

| Service | URL | Credentials |
|---------|-----|-------------|
| **User Dashboard** | http://localhost:3000 | N/A |
| **API Gateway** | http://localhost:8080 | N/A |
| **Grafana** | http://localhost:3001 | admin/admin |
| **Prometheus** | http://localhost:9090 | N/A |
| **Consul UI** | http://localhost:8500 | N/A |
| **MinIO Console** | http://localhost:9001 | admin/secure_minio_password |

## Service Details

### API Gateway (Port: 8080)

**Technology Stack**: Go, Chi Router, Redis
**Primary Role**: Central API routing, authentication middleware, rate limiting, and request validation

**Core Endpoints:**
```
GET    /health                    - System health check and status
POST   /api/v1/auth/*            - Authentication operations routing
GET    /api/v1/providers/*       - GPU provider listing and filtering
POST   /api/v1/jobs/*            - Job management and orchestration
GET    /api/v1/billing/*         - Billing and payment queries
GET    /api/v1/storage/*         - File storage operations
```

**Features:**
- Request routing and load balancing
- JWT token validation and refresh
- Rate limiting (60 requests/minute per user)
- Request/response logging and metrics
- CORS policy enforcement

### Auth Service (Port: 8090)

**Technology Stack**: Python, FastAPI, PostgreSQL, Redis
**Primary Role**: User authentication, authorization, and session management

**Core Endpoints:**
```
POST   /api/v1/auth/register     - User registration with email verification
POST   /api/v1/auth/login        - User authentication and token generation
POST   /api/v1/auth/refresh      - JWT token refresh and validation
GET    /api/v1/auth/me           - Current user profile and permissions
PUT    /api/v1/auth/profile      - User profile updates
POST   /api/v1/auth/logout       - Session termination and token invalidation
```

**Features:**
- Secure password hashing with bcrypt
- JWT token generation and validation
- Role-based access control (RBAC)
- Email verification and password reset
- Session management and security logging

### Billing Service (Port: 8082)

**Technology Stack**: Go, Solana SDK, PostgreSQL
**Primary Role**: Blockchain payment processing, dGPU token management, and usage billing

**Core Endpoints:**
```
POST   /api/v1/billing/create-wallet    - Solana wallet creation and management
POST   /api/v1/billing/transfer         - dGPU token transfers and payments
GET    /api/v1/billing/balance          - Account balance and transaction history
POST   /api/v1/billing/estimate-cost    - Real-time cost estimation
GET    /api/v1/billing/transactions     - Transaction history and status
POST   /api/v1/billing/withdraw         - Withdrawal requests and processing
```

**Features:**
- Solana blockchain integration
- Automated escrow and payment distribution
- Real-time usage tracking and billing
- Multi-signature wallet support
- Transaction verification and audit trails

### Provider Registry (Port: 8081)

**Technology Stack**: Go, PostgreSQL, NATS
**Primary Role**: GPU provider registration, capability assessment, and performance monitoring

**Core Endpoints:**
```
POST   /api/v1/providers/register       - Provider registration and verification
GET    /api/v1/providers/list           - Available providers with filtering
PUT    /api/v1/providers/{id}/status    - Provider status updates
GET    /api/v1/providers/{id}/metrics   - Real-time performance metrics
POST   /api/v1/providers/{id}/verify    - Provider verification process
DELETE /api/v1/providers/{id}           - Provider deregistration
```

**Features:**
- Automated GPU detection and benchmarking
- Real-time availability tracking
- Performance metrics collection
- Provider reputation scoring
- Geographic distribution management

### Scheduler Orchestrator (Port: 8084)

**Technology Stack**: Go, Docker API, Kubernetes (optional)
**Primary Role**: Job scheduling, container orchestration, and resource allocation

**Core Endpoints:**
```
POST   /api/v1/jobs/submit              - Job submission and validation
GET    /api/v1/jobs/{id}/status         - Job status and progress tracking
DELETE /api/v1/jobs/{id}               - Job cancellation and cleanup
GET    /api/v1/jobs/{id}/logs           - Real-time job logs and output
POST   /api/v1/jobs/{id}/scale          - Job scaling and resource adjustment
GET    /api/v1/jobs/queue               - Job queue status and statistics
```

**Features:**
- Docker container orchestration
- Resource allocation and optimization
- Job queue management and prioritization
- Real-time monitoring and logging
- Automatic failover and recovery

### Storage Service (Port: 8083)

**Technology Stack**: Go, MinIO SDK, S3 API
**Primary Role**: File management, artifact storage, and data persistence

**Core Endpoints:**
```
POST   /api/v1/storage/upload           - File upload with metadata
GET    /api/v1/storage/download/{id}    - Secure file download
DELETE /api/v1/storage/{id}             - File deletion and cleanup
GET    /api/v1/storage/list             - File listing with pagination
POST   /api/v1/storage/share            - File sharing and access control
GET    /api/v1/storage/metadata/{id}    - File metadata and properties
```

**Features:**
- S3-compatible API interface
- Secure file upload and download
- Metadata management and indexing
- Access control and permissions
- Automatic backup and versioning

## Development

### Local Development Environment

#### Prerequisites for Development
- **Go**: Version 1.21+ for backend services
- **Python**: Version 3.11+ for auth service
- **Node.js**: Version 18+ for frontend applications
- **Docker**: For infrastructure services
- **Git**: For version control

#### Infrastructure Services Setup
```bash
# Start core infrastructure services
docker-compose up -d postgres redis nats consul minio

# Verify services are running
docker-compose ps

# Check service logs
docker-compose logs -f postgres redis
```

#### Backend Services Development

**Auth Service (Python/FastAPI):**
```bash
cd auth-service

# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt
pip install -r requirements-dev.txt

# Run development server
uvicorn app.main:app --reload --port 8090 --host 0.0.0.0
```

**API Gateway (Go):**
```bash
cd api-gateway

# Install dependencies
go mod download

# Run development server
go run cmd/main.go

# Or with hot reload using air
air
```

**Other Go Services:**
```bash
# Billing Service
cd billing-service && go run cmd/main.go

# Provider Registry
cd provider-registry && go run cmd/main.go

# Scheduler Orchestrator
cd scheduler-orchestrator && go run cmd/main.go

# Storage Service
cd storage-service && go run cmd/main.go
```

#### Frontend Development

**User Dashboard (React):**
```bash
cd gpu-rental-frontend

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

**Provider GUI (Tauri):**
```bash
cd provider-gui

# Install dependencies
npm install

# Start development server
npm run tauri dev

# Build desktop application
npm run tauri build
```

### Testing and Quality Assurance

#### System Health Verification
```bash
# Check all services health
./scripts/health-check.sh

# Individual service health checks
curl http://localhost:8080/health
curl http://localhost:8090/health
curl http://localhost:8081/health
```

#### Test Data Setup
```bash
# Create test users and providers
cd auth-service
python scripts/create_test_data.py

# Generate sample GPU providers
cd provider-registry
go run scripts/generate_test_providers.go

# Create test blockchain wallets
cd billing-service
go run scripts/setup_test_wallets.go
```

#### Integration Testing
```bash
# Run complete GPU rental workflow test
./scripts/integration-test.sh

# Test blockchain payment flow
./scripts/test-payment-flow.sh

# Load testing
./scripts/load-test.sh
```

#### Unit Testing
```bash
# Go services
cd api-gateway && go test ./...
cd billing-service && go test ./...

# Python services
cd auth-service && python -m pytest

# Frontend
cd gpu-rental-frontend && npm test
```

## Monitoring and Logging

### Observability Stack

#### Metrics Collection and Visualization
**Prometheus Configuration:**
- Service discovery via Consul
- Custom metrics for GPU utilization, job performance, and billing
- Retention period: 30 days for detailed metrics, 1 year for aggregated data

**Grafana Dashboards:**
- **System Overview**: Infrastructure health, service status, and resource utilization
- **GPU Utilization**: Real-time GPU usage, temperature, and performance metrics
- **Transaction Monitoring**: Blockchain transaction status, payment flows, and escrow management
- **Service Health**: Microservice health checks, response times, and error rates
- **Business Metrics**: Revenue tracking, user engagement, and provider performance
- **Security Dashboard**: Authentication attempts, rate limiting, and security events

#### Log Aggregation and Analysis
**Loki Configuration:**
- Centralized log collection from all services
- Log retention: 90 days for application logs, 1 year for audit logs
- Structured logging with JSON format for better searchability

**Promtail Setup:**
- Automatic log shipping from Docker containers
- Log parsing and labeling for efficient querying
- Multi-tenant log isolation for security

**Log Categories:**
- **Application Logs**: Service-specific operational logs
- **Audit Logs**: Security events, authentication, and authorization
- **Transaction Logs**: Blockchain operations and payment processing
- **Performance Logs**: Response times, resource usage, and bottlenecks

#### Alerting and Notification System
**Prometheus AlertManager:**
- Service downtime alerts with escalation policies
- Resource utilization thresholds (CPU > 80%, Memory > 85%)
- GPU provider availability monitoring
- Blockchain transaction failure alerts
- Security incident notifications

**Notification Channels:**
- **Slack Integration**: Real-time alerts to development and operations teams
- **Email Notifications**: Critical alerts and daily/weekly reports
- **PagerDuty**: On-call escalation for production incidents
- **Webhook Integration**: Custom notification endpoints for third-party systems

### Performance Monitoring

#### Application Performance Monitoring (APM)
- Request tracing across microservices
- Database query performance analysis
- API endpoint response time monitoring
- Error rate tracking and analysis

#### Infrastructure Monitoring
- Server resource utilization (CPU, memory, disk, network)
- Container health and resource consumption
- Database performance and connection pooling
- Message queue throughput and latency

## Security

### Authentication and Authorization

#### JWT Token Management
- **Token Structure**: Secure JWT tokens with user claims and permissions
- **Token Lifecycle**: 15-minute access tokens with 7-day refresh tokens
- **Token Rotation**: Automatic token refresh with secure rotation mechanism
- **Token Revocation**: Immediate token invalidation on logout or security events

#### Role-Based Access Control (RBAC)
**User Roles:**
- **Admin**: Full platform access and management capabilities
- **Provider**: GPU resource management and earnings tracking
- **User**: GPU rental and job management
- **Auditor**: Read-only access for compliance and monitoring

**Permission Matrix:**
```
Resource          | Admin | Provider | User | Auditor
------------------|-------|----------|------|--------
User Management   |   ✓   |    ✗     |  ✗   |   ✓
Provider Registry |   ✓   |    ✓     |  ✗   |   ✓
Job Management    |   ✓   |    ✓     |  ✓   |   ✓
Billing System    |   ✓   |    ✓     |  ✓   |   ✓
System Config     |   ✓   |    ✗     |  ✗   |   ✗
```

### API Security Framework

#### Request Validation and Rate Limiting
- **Input Validation**: Comprehensive request payload validation using JSON schemas
- **Rate Limiting**: Tiered rate limiting based on user roles and subscription levels
- **DDoS Protection**: Automatic IP blocking and traffic shaping
- **Request Sanitization**: SQL injection and XSS prevention

#### Network Security
- **HTTPS Enforcement**: TLS 1.3 encryption for all client communications
- **CORS Policy**: Strict cross-origin resource sharing configuration
- **API Versioning**: Backward-compatible API versioning with deprecation notices
- **Request Logging**: Comprehensive audit trail for all API requests

### Blockchain Security

#### Wallet and Key Management
- **Private Key Encryption**: AES-256 encryption for stored private keys
- **Hardware Security Module (HSM)**: Optional HSM integration for enterprise deployments
- **Key Rotation**: Regular private key rotation for enhanced security
- **Multi-Signature Support**: Multi-sig wallets for high-value transactions

#### Transaction Security
- **Transaction Verification**: Comprehensive blockchain transaction validation
- **Escrow System**: Automated escrow for secure payment processing
- **Fraud Detection**: Machine learning-based fraud detection and prevention
- **Audit Trail**: Immutable transaction history on the blockchain

#### Smart Contract Security
- **Contract Auditing**: Regular security audits of smart contracts
- **Upgrade Mechanisms**: Secure contract upgrade procedures
- **Emergency Stops**: Circuit breakers for emergency contract suspension
- **Access Controls**: Multi-signature governance for contract modifications

## Deployment

### Production Deployment Strategy

#### Container Orchestration
**Docker Compose Deployment:**
```bash
# Production environment deployment
./scripts/deploy-production.sh

# Manual Docker Compose deployment
docker-compose -f docker-compose.prod.yml up -d

# Verify deployment status
./scripts/verify-deployment.sh
```

**Kubernetes Deployment (Recommended for Scale):**
```bash
# Deploy to Kubernetes cluster
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmaps/
kubectl apply -f k8s/secrets/
kubectl apply -f k8s/services/
kubectl apply -f k8s/deployments/

# Verify deployment
kubectl get pods -n dantegpu
kubectl get services -n dantegpu
```

#### Environment-Specific Configurations

**Staging Environment:**
```bash
# Deploy to staging
./scripts/deploy-staging.sh

# Run integration tests
./scripts/run-staging-tests.sh
```

**Production Environment:**
```bash
# Pre-deployment checks
./scripts/pre-deployment-checks.sh

# Deploy with zero downtime
./scripts/deploy-production-rolling.sh

# Post-deployment verification
./scripts/post-deployment-verification.sh
```

### Scaling and Load Balancing

#### Horizontal Scaling
```bash
# Scale API Gateway instances
docker-compose up -d --scale api-gateway=3

# Scale billing service for high transaction volume
docker-compose up -d --scale billing-service=2

# Scale provider registry for large provider networks
docker-compose up -d --scale provider-registry=2
```

#### Load Balancer Configuration
**NGINX Configuration:**
```nginx
upstream api_gateway {
    server api-gateway-1:8080;
    server api-gateway-2:8080;
    server api-gateway-3:8080;
}

upstream auth_service {
    server auth-service-1:8090;
    server auth-service-2:8090;
}
```

#### Auto-scaling with Kubernetes
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-gateway-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Backup and Disaster Recovery

#### Database Backup Strategy
```bash
# Automated daily backups
./scripts/backup-database.sh

# Point-in-time recovery setup
./scripts/setup-pitr.sh

# Backup verification
./scripts/verify-backups.sh
```

#### Disaster Recovery Procedures
- **RTO (Recovery Time Objective)**: 4 hours
- **RPO (Recovery Point Objective)**: 1 hour
- **Multi-region deployment** for high availability
- **Automated failover** mechanisms

## Performance

### Benchmark Results and Capacity Planning

#### Load Testing Results
**Concurrent User Capacity:**
- **1,000 concurrent users**: Sustained with 99.9% uptime
- **5,000 concurrent users**: Peak capacity with auto-scaling
- **10,000+ concurrent users**: Achievable with multi-region deployment

**Transaction Throughput:**
- **API Requests**: 10,000+ requests/second (with load balancing)
- **Job Submissions**: 500+ jobs/second
- **Blockchain Transactions**: Limited by Solana network (65,000 TPS theoretical)
- **File Uploads**: 1GB/second aggregate throughput

**Response Time Metrics:**
- **API Gateway**: <50ms average, <200ms 99th percentile
- **Authentication**: <100ms average
- **Job Submission**: <500ms average
- **Blockchain Operations**: 1-3 seconds (network dependent)

#### Performance Optimization Strategies

**Application Level:**
- **Connection Pooling**: Database connection pools with optimal sizing
- **Caching Strategy**: Multi-layer caching with Redis and in-memory caches
- **Query Optimization**: Database query optimization and indexing
- **Async Processing**: Non-blocking I/O and async job processing

**Infrastructure Level:**
- **Container Optimization**: Minimal container images and resource limits
- **Load Balancing**: Intelligent load distribution across service instances
- **CDN Integration**: Content delivery network for static assets
- **Database Sharding**: Horizontal database partitioning for scale

**Network Optimization:**
- **HTTP/2 Support**: Multiplexed connections and server push
- **Compression**: Gzip/Brotli compression for API responses
- **Keep-Alive Connections**: Persistent connections to reduce overhead
- **Regional Deployment**: Geographic distribution for reduced latency

### Monitoring and Performance Tuning

#### Key Performance Indicators (KPIs)
- **System Availability**: 99.9% uptime SLA
- **Response Time**: <100ms average API response time
- **Error Rate**: <0.1% error rate across all services
- **Resource Utilization**: <80% CPU, <85% memory usage
- **User Satisfaction**: >95% user satisfaction score

#### Performance Monitoring Tools
- **Application Performance Monitoring**: Real-time performance tracking
- **Database Performance**: Query performance and optimization recommendations
- **Infrastructure Monitoring**: Resource utilization and capacity planning
- **User Experience Monitoring**: Real user monitoring and synthetic testing

## API Documentation

### Authentication Endpoints

#### User Registration
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure_password",
  "name": "John Doe",
  "role": "user"
}
```

#### User Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure_password"
}
```

### Provider Management Endpoints

#### Register GPU Provider
```http
POST /api/v1/providers/register
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "name": "High-Performance GPU Cluster",
  "description": "NVIDIA RTX 4090 cluster for AI workloads",
  "gpu_specs": {
    "model": "RTX 4090",
    "memory": "24GB",
    "cuda_cores": 16384
  },
  "pricing": {
    "per_hour": 2.50,
    "currency": "dGPU"
  }
}
```

#### List Available Providers
```http
GET /api/v1/providers/list?gpu_type=nvidia&min_memory=16&available=true
Authorization: Bearer <jwt_token>
```

### Job Management Endpoints

#### Submit GPU Job
```http
POST /api/v1/jobs/submit
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "provider_id": "provider_123",
  "job_config": {
    "image": "tensorflow/tensorflow:latest-gpu",
    "command": ["python", "train_model.py"],
    "resources": {
      "gpu_count": 1,
      "memory": "16GB",
      "cpu_cores": 4
    }
  },
  "duration_hours": 2
}
```

### Billing Endpoints

#### Get Account Balance
```http
GET /api/v1/billing/balance
Authorization: Bearer <jwt_token>
```

#### Estimate Job Cost
```http
POST /api/v1/billing/estimate-cost
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "provider_id": "provider_123",
  "duration_hours": 2,
  "gpu_count": 1
}
```

## Contributing

### Development Workflow

#### 1. Fork and Clone
```bash
# Fork the repository on GitHub
# Clone your fork locally
git clone https://github.com/your-username/dantegpu-core.git
cd dantegpu-core

# Add upstream remote
git remote add upstream https://github.com/dante-gpu/dantegpu-core.git
```

#### 2. Create Feature Branch
```bash
# Create and switch to feature branch
git checkout -b feature/amazing-new-feature

# Keep your fork updated
git fetch upstream
git rebase upstream/main
```

#### 3. Development Guidelines

**Code Standards:**
- **Go**: Follow Go best practices and use `gofmt`, `golint`, and `go vet`
- **Python**: Follow PEP 8 style guide and use `black` for formatting
- **TypeScript/React**: Use ESLint and Prettier for consistent formatting
- **Documentation**: Update relevant documentation for new features

**Commit Message Format:**
```
type(scope): brief description

Detailed explanation of the change, including:
- What was changed and why
- Any breaking changes
- References to issues or tickets

Closes #123
```

**Commit Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

#### 4. Testing Requirements
```bash
# Run all tests before submitting
./scripts/run-all-tests.sh

# Ensure code coverage meets requirements
./scripts/check-coverage.sh

# Run linting and formatting checks
./scripts/lint-all.sh
```

#### 5. Pull Request Process

**Before Submitting:**
- Ensure all tests pass
- Update documentation if necessary
- Add tests for new functionality
- Verify no breaking changes (or document them)

**Pull Request Template:**
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added/updated
```

### Code Review Process

#### Review Criteria
- **Functionality**: Code works as intended and meets requirements
- **Performance**: No significant performance regressions
- **Security**: No security vulnerabilities introduced
- **Maintainability**: Code is readable and well-documented
- **Testing**: Adequate test coverage for new functionality

#### Review Timeline
- **Initial Review**: Within 2 business days
- **Follow-up Reviews**: Within 1 business day
- **Merge Timeline**: After approval and CI checks pass

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for complete details.

### MIT License Summary
- **Commercial Use**: Permitted
- **Modification**: Permitted
- **Distribution**: Permitted
- **Private Use**: Permitted
- **Liability**: Limited
- **Warranty**: None

## Support

### Documentation and Resources
- **Technical Documentation**: [GitHub Wiki](https://github.com/dante-gpu/dantegpu-core/wiki)
- **API Reference**: [API Documentation](https://docs.dantegpu.com/api)
- **Tutorials**: [Getting Started Guide](https://docs.dantegpu.com/tutorials)
- **FAQ**: [Frequently Asked Questions](https://docs.dantegpu.com/faq)

### Community Support
- **GitHub Issues**: [Report bugs and request features](https://github.com/dante-gpu/dantegpu-core/issues)
- **GitHub Discussions**: [Community discussions and Q&A](https://github.com/dante-gpu/dantegpu-core/discussions)
- **Discord Server**: [Real-time community chat](https://discord.gg/dantegpu)
- **Reddit Community**: [r/DanteGPU](https://reddit.com/r/dantegpu)

### Professional Support
- **Email Support**: support@dantegpu.com
- **Enterprise Support**: enterprise@dantegpu.com
- **Security Issues**: security@dantegpu.com
- **Partnership Inquiries**: partnerships@dantegpu.com

### Response Times
- **Community Support**: Best effort, typically 24-48 hours
- **Bug Reports**: 1-3 business days
- **Security Issues**: Within 24 hours
- **Enterprise Support**: Based on SLA agreement

---

**DanteGPU** - Decentralized GPU Computing Platform

*Empowering the future of distributed computing through blockchain technology and community-driven GPU sharing.*
