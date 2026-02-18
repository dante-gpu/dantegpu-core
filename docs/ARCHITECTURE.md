# DanteGPU Core - System Architecture

{🙏 Don't worry about what the f😳ck I be doing, I'm Mock King AKINCI}

## Overview

DanteGPU Core is a decentralized GPU rental platform built on Solana blockchain, enabling users to rent GPU compute power for AI/ML workloads while providers earn cryptocurrency.

## Architecture Principles

1. **Microservices Architecture**: Independent, scalable services
2. **Event-Driven**: NATS JetStream for async communication
3. **Blockchain-First**: Solana for payments and escrow
4. **Cloud-Native**: Kubernetes-ready, containerized
5. **Security-First**: Zero-trust, encrypted, audited
6. **Real-Time**: WebSocket for live updates
7. **Observable**: Comprehensive monitoring and logging

---

## System Components

### Frontend Applications

#### User Dashboard (React + TypeScript)
- **Purpose**: User interface for renting GPUs and managing jobs
- **Tech Stack**: React 18, TypeScript, Vite, TailStack Query, Tailwind CSS
- **Features**:
  - User authentication and profile management
  - GPU marketplace browsing and filtering
  - Wallet integration (Phantom, Solflare)
  - Job submission and monitoring
  - Real-time job logs via WebSocket
  - Billing history and analytics

#### Provider Web App (React + TypeScript)
- **Purpose**: Provider portal for managing GPU offerings
- **Tech Stack**: React 18, TypeScript, Vite, TanStack Query
- **Features**:
  - Provider registration and verification
  - GPU capability management
  - Earnings dashboard
  - Rental history
  - Performance metrics

#### Provider GUI (Tauri + Rust + React)
- **Purpose**: Desktop application for GPU providers
- **Tech Stack**: Tauri, Rust, React, TypeScript
- **Features**:
  - GPU detection and monitoring
  - Automatic job execution
  - System resource monitoring
  - Offline capability
  - Secure key management

---

### Backend Services

#### API Gateway (Go)
- **Port**: 8000
- **Purpose**: Single entry point for all client requests
- **Responsibilities**:
  - Request routing to microservices
  - Authentication and authorization
  - Rate limiting (60 req/min per user)
  - Request/response transformation
  - CORS handling
  - WebSocket hub for real-time updates
- **Dependencies**: Redis (rate limiting), Consul (service discovery)

#### Auth Service (Python + FastAPI)
- **Port**: 8001
- **Purpose**: User authentication and authorization
- **Responsibilities**:
  - User registration and email verification
  - Login with JWT tokens (15min access, 7d refresh)
  - Password reset flow
  - Two-factor authentication (TOTP, SMS, Email)
  - OAuth2 integration (Google, GitHub)
  - API key management
  - Role-based access control (RBAC)
  - Session management (PostgreSQL + Redis)
- **Database**: `dante_auth`
- **Security**: Bcrypt (cost 12), account lockout after 5 failed attempts

#### Billing Service (Go)
- **Port**: 8002
- **Purpose**: Blockchain transactions and billing
- **Responsibilities**:
  - Solana wallet management
  - dGPU token transfers
  - Escrow creation and release
  - Minute-based billing calculation
  - Platform fee collection (5%)
  - Provider payouts
  - Transaction history
- **Blockchain**: Solana mainnet-beta
- **Token**: dGPU (7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump)
- **Database**: `dante_billing`

#### Provider Registry (Go)
- **Port**: 8003
- **Purpose**: GPU provider and capability management
- **Responsibilities**:
  - Provider registration and verification
  - GPU capability registration
  - GPU availability tracking
  - Marketplace listing
  - Provider statistics
  - Rating and review system
- **Database**: `dante_registry`

#### Scheduler (Go)
- **Port**: 8004
- **Purpose**: Job scheduling and execution
- **Responsibilities**:
  - Job submission and validation
  - GPU allocation
  - Job queue management
  - Docker container orchestration
  - Job lifecycle management
  - Log collection and streaming
  - Job cancellation
- **Database**: `dante_scheduler`
- **Message Queue**: NATS JetStream (JOBS stream)

#### Storage Service (Go)
- **Port**: 8005
- **Purpose**: File storage and management
- **Responsibilities**:
  - Dataset upload/download
  - Model storage
  - Job artifacts
  - Temporary file management
- **Storage**: MinIO (S3-compatible)
- **Buckets**: datasets, models, job-outputs, logs, backups, temp, user-uploads, provider-data

#### GPU Monitoring (Go)
- **Port**: 8006
- **Purpose**: Real-time GPU metrics collection
- **Responsibilities**:
  - GPU utilization tracking
  - Temperature monitoring
  - Memory usage
  - Power consumption
  - Performance metrics
- **Database**: `dante_core` (partitioned metrics table)
- **Message Queue**: NATS JetStream (METRICS stream)

#### Provider Daemon (Go)
- **Purpose**: Agent running on provider machines
- **Responsibilities**:
  - GPU detection (NVIDIA, AMD, Intel)
  - Job execution in Docker containers
  - Resource monitoring
  - Log streaming
  - Heartbeat to registry
- **Deployment**: Runs on provider hardware

#### Terminal Streaming (Go)
- **Port**: 8007
- **Purpose**: Real-time terminal access to jobs
- **Responsibilities**:
  - WebSocket terminal sessions
  - PTY allocation
  - Input/output streaming
  - Session management
- **Protocol**: WebSocket with xterm.js

---

### Infrastructure Services

#### PostgreSQL 15
- **Purpose**: Primary data store
- **Databases**:
  - `dante_auth`: Users, sessions, permissions
  - `dante_billing`: Transactions, wallets, billing
  - `dante_registry`: Providers, GPUs, capabilities
  - `dante_scheduler`: Jobs, logs, queue
  - `dante_core`: Metrics, audit logs, analytics
- **Features**:
  - Table partitioning by month (logs, metrics, audit logs)
  - Materialized views for analytics
  - GIN indexes for JSONB
  - Full-text search
  - 150+ optimized indexes
  - Stored procedures for business logic

#### Redis 7
- **Purpose**: Caching and session storage
- **Use Cases**:
  - Session storage (dual with PostgreSQL)
  - Rate limiting counters
  - Token blacklist
  - Cache for frequently accessed data
  - Pub/Sub for real-time events
- **Configuration**: 2GB max memory, allkeys-lru eviction

#### NATS JetStream
- **Purpose**: Message queue and event streaming
- **Streams**:
  - `JOBS`: Job submission and updates
  - `EVENTS`: System events
  - `METRICS`: GPU and system metrics
  - `BILLING`: Billing events
  - `LOGS`: Application logs
  - `PROVIDER`: Provider events
- **Configuration**: 10GB file store, 1GB memory store

#### Consul
- **Purpose**: Service discovery and configuration
- **Features**:
  - Service registration
  - Health checking
  - KV store for configuration
  - DNS interface

#### MinIO
- **Purpose**: S3-compatible object storage
- **Buckets**: 8 buckets for different data types
- **Features**: Versioning, lifecycle policies, encryption

---

### Monitoring Stack

#### Prometheus
- **Purpose**: Metrics collection and alerting
- **Scrape Targets**:
  - All microservices (9090 port)
  - PostgreSQL exporter
  - Redis exporter
  - NATS metrics
  - Kubernetes metrics
- **Retention**: 30 days
- **Alerts**: Service down, high error rate, resource usage

#### Grafana
- **Purpose**: Metrics visualization
- **Dashboards**:
  - System overview
  - Service health
  - GPU utilization
  - Billing analytics
  - User activity
  - Provider performance

#### Loki
- **Purpose**: Log aggregation
- **Sources**: All microservices, Kubernetes pods
- **Retention**: 7 days

#### AlertManager
- **Purpose**: Alert routing and notification
- **Channels**: Slack, Email, PagerDuty

---

## Data Flow

### User Rental Flow

```
1. User browses GPUs (User Dashboard → API Gateway → Provider Registry)
2. User creates wallet (User Dashboard → API Gateway → Billing Service → Solana)
3. User starts rental (User Dashboard → API Gateway → Billing Service)
   - Billing Service creates escrow on Solana
   - Billing Service marks GPU as unavailable
   - Billing Service starts billing timer
4. User submits job (User Dashboard → API Gateway → Scheduler)
   - Scheduler validates job
   - Scheduler publishes to NATS JOBS stream
   - Provider Daemon picks up job
   - Provider Daemon executes in Docker
5. Job runs (Provider Daemon → GPU Monitoring → NATS METRICS stream)
6. User views logs (User Dashboard → WebSocket → Terminal Streaming)
7. Job completes (Provider Daemon → Scheduler → NATS JOBS stream)
8. User ends rental (User Dashboard → API Gateway → Billing Service)
   - Billing Service calculates final cost
   - Billing Service releases escrow
   - Billing Service pays provider (95%)
   - Billing Service collects platform fee (5%)
   - Billing Service marks GPU as available
```

### Provider Registration Flow

```
1. Provider registers (Provider Web App → API Gateway → Provider Registry)
2. Provider adds GPU (Provider Web App → API Gateway → Provider Registry)
3. Provider installs daemon (Provider GUI → Provider Daemon)
4. Daemon detects GPUs (Provider Daemon → GPU Monitoring)
5. Daemon registers with registry (Provider Daemon → Provider Registry)
6. Daemon sends heartbeat (Provider Daemon → Provider Registry every 30s)
7. GPU appears in marketplace (User Dashboard → API Gateway → Provider Registry)
```

---

## Security Architecture

### Authentication Flow
1. User logs in with email/password
2. Auth Service validates credentials (bcrypt)
3. Auth Service generates JWT tokens (access + refresh)
4. Access token expires in 15 minutes
5. Refresh token expires in 7 days
6. Tokens stored in PostgreSQL + Redis
7. API Gateway validates tokens on every request

### Authorization
- **RBAC**: Roles (admin, user, provider) with permissions
- **Middleware**: API Gateway checks permissions
- **Database**: Row-level security in PostgreSQL

### Blockchain Security
- **Escrow**: Funds locked in Solana smart contract
- **Multi-sig**: Platform wallet requires multiple signatures
- **Audit**: All transactions logged and auditable

### Network Security
- **TLS**: All external communication encrypted
- **VPN**: Internal service communication
- **Firewall**: Kubernetes network policies
- **DDoS**: CloudFlare protection

---

## Scalability

### Horizontal Scaling
- All services are stateless (except databases)
- Kubernetes HPA for auto-scaling
- Load balancing via Kubernetes services

### Database Scaling
- Read replicas for PostgreSQL
- Table partitioning for large tables
- Connection pooling (pgBouncer)
- Materialized views for analytics

### Caching Strategy
- Redis for frequently accessed data
- CDN for static assets
- Browser caching for frontend

### Message Queue
- NATS JetStream for async processing
- Decouples services
- Handles traffic spikes

---

## Disaster Recovery

### Backup Strategy
- **PostgreSQL**: Daily full backup, hourly incremental
- **MinIO**: Versioning enabled, cross-region replication
- **Configuration**: Git-based, versioned

### High Availability
- **Services**: 3+ replicas in production
- **Databases**: Primary + 2 replicas
- **Load Balancers**: Multi-AZ
- **Monitoring**: 24/7 alerting

### Recovery Procedures
- **RTO**: 1 hour (Recovery Time Objective)
- **RPO**: 15 minutes (Recovery Point Objective)
- **Runbooks**: Documented procedures for all scenarios

---

## Deployment Architecture

### Environments
- **Development**: Local Docker Compose
- **Staging**: Kubernetes cluster (3 nodes)
- **Production**: Kubernetes cluster (10+ nodes, multi-AZ)

### CI/CD Pipeline
1. Code push to GitHub
2. GitHub Actions runs tests
3. Build Docker images
4. Push to GitHub Container Registry
5. Deploy to staging (auto)
6. Run smoke tests
7. Deploy to production (manual approval)
8. Blue-green deployment
9. Monitor and rollback if needed

---

For implementation details, see [IMPLEMENTATION_COMPLETE.md](./IMPLEMENTATION_COMPLETE.md)

