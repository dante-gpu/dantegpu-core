# DanteGPU Platform - Detaylı Mimari Dokümantasyonu

## 🏗️ Sistem Mimarisi Genel Bakış

DanteGPU platformu, modern mikroservis mimarisi prensipleri kullanılarak tasarlanmış, ölçeklenebilir ve güvenilir bir GPU kiralama sistemidir.

## 📊 Mimari Diyagramı

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend Layer                           │
├─────────────────────────────────────────────────────────────────┤
│  Web Dashboard (React)  │  Provider GUI (Tauri)  │  Mobile App  │
└─────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                      API Gateway (Go)                          │
│              Load Balancing, Auth, Rate Limiting               │
└─────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│   Auth Service  │ │ Billing Service │ │Provider Registry│
│   (Python)      │ │     (Go)        │ │     (Go)        │
│   Port: 8090    │ │   Port: 8082    │ │   Port: 8081    │
└─────────────────┘ └─────────────────┘ └─────────────────┘
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│Scheduler Service│ │ Storage Service │ │Monitor Services │
│     (Go)        │ │     (Go)        │ │(Prometheus/etc) │
│   Port: 8084    │ │   Port: 8083    │ │  Various Ports  │
└─────────────────┘ └─────────────────┘ └─────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│   PostgreSQL    │ │      Redis      │ │  NATS JetStream │
│   (Database)    │ │    (Cache)      │ │ (Message Queue) │
└─────────────────┘ └─────────────────┘ └─────────────────┘
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│     Consul      │ │      MinIO      │ │ Solana Blockchain│
│(Service Discovery)│ │   (Storage)     │ │   (Payments)    │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

## 🔧 Mikroservis Detayları

### 1. API Gateway (Port: 8080)

**Teknoloji Stack:**
- **Dil**: Go 1.21+
- **Framework**: Chi Router
- **Özellikler**: Load Balancing, Authentication, Rate Limiting

**Sorumluluklar:**
- Tüm API isteklerinin merkezi giriş noktası
- JWT token doğrulama
- Rate limiting (60 req/min per user)
- Service discovery ile backend routing
- CORS policy enforcement
- Request/Response logging

**Konfigürasyon:**
```yaml
# api-gateway/configs/config.yaml
port: ":8080"
nats_address: "nats://nats:4222"
consul_address: "consul:8500"
auth_service_url: "http://auth-service:8090"
rate_limit: 60 # requests per minute
```

**Ana Endpoint'ler:**
```
GET  /health                    # Health check
POST /api/v1/auth/*            # Auth service proxy
GET  /api/v1/providers/*       # Provider registry proxy
POST /api/v1/jobs/*            # Scheduler service proxy
GET  /api/v1/billing/*         # Billing service proxy
POST /api/v1/storage/*         # Storage service proxy
```

### 2. Auth Service (Port: 8090)

**Teknoloji Stack:**
- **Dil**: Python 3.11+
- **Framework**: FastAPI
- **Database**: PostgreSQL
- **ORM**: SQLAlchemy + Alembic

**Sorumluluklar:**
- Kullanıcı kayıt ve giriş işlemleri
- JWT token oluşturma ve doğrulama
- Password hashing (bcrypt)
- User profile management
- Role-based access control (RBAC)

**Database Schema:**
```sql
-- users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,
    role VARCHAR(50) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- user_sessions table
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    refresh_token VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**API Endpoints:**
```python
POST /api/v1/auth/register      # User registration
POST /api/v1/auth/login         # User login
POST /api/v1/auth/refresh       # Token refresh
GET  /api/v1/auth/me           # Get user profile
PUT  /api/v1/auth/profile      # Update profile
POST /api/v1/auth/logout       # Logout
POST /api/v1/auth/verify-email # Email verification
```

### 3. Billing Service (Port: 8082)

**Teknoloji Stack:**
- **Dil**: Go 1.21+
- **Blockchain**: Solana
- **Token**: dGPU (SPL Token)
- **Database**: PostgreSQL

**Sorumluluklar:**
- Solana blockchain entegrasyonu
- dGPU token işlemleri
- Wallet oluşturma ve yönetimi
- Real-time billing (dakika bazında)
- Escrow sistemi
- Platform fee collection (%5)

**Database Schema:**
```sql
-- wallets table
CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    public_key VARCHAR(44) NOT NULL,
    encrypted_private_key TEXT NOT NULL,
    balance_dgpu DECIMAL(20,8) DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- transactions table
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_wallet_id UUID REFERENCES wallets(id),
    to_wallet_id UUID REFERENCES wallets(id),
    amount_dgpu DECIMAL(20,8) NOT NULL,
    transaction_hash VARCHAR(88),
    status VARCHAR(20) DEFAULT 'pending',
    job_id UUID,
    created_at TIMESTAMP DEFAULT NOW()
);

-- billing_sessions table
CREATE TABLE billing_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL,
    user_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    hourly_rate_dgpu DECIMAL(10,4) NOT NULL,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    total_cost_dgpu DECIMAL(20,8),
    status VARCHAR(20) DEFAULT 'active'
);
```

**API Endpoints:**
```go
POST /api/v1/billing/create-wallet     # Create new wallet
GET  /api/v1/billing/wallet/{id}       # Get wallet info
POST /api/v1/billing/transfer          # Transfer tokens
GET  /api/v1/billing/balance/{wallet}  # Get balance
POST /api/v1/billing/estimate-cost     # Cost estimation
GET  /api/v1/billing/transactions      # Transaction history
POST /api/v1/billing/start-session     # Start billing session
POST /api/v1/billing/end-session       # End billing session
```

### 4. Provider Registry (Port: 8081)

**Teknoloji Stack:**
- **Dil**: Go 1.21+
- **Database**: PostgreSQL
- **Message Queue**: NATS JetStream

**Sorumluluklar:**
- GPU provider kayıt ve yönetimi
- GPU specifications tracking
- Provider performance metrics
- Availability monitoring
- Geographic location management
- Rating and review system

**Database Schema:**
```sql
-- providers table
CREATE TABLE providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    location VARCHAR(100),
    status VARCHAR(20) DEFAULT 'offline',
    total_earnings_dgpu DECIMAL(20,8) DEFAULT 0,
    rating DECIMAL(3,2) DEFAULT 0,
    total_jobs_completed INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- gpus table
CREATE TABLE gpus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID REFERENCES providers(id),
    name VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL,
    vram_mb INTEGER NOT NULL,
    compute_capability VARCHAR(10),
    is_available BOOLEAN DEFAULT true,
    hourly_rate_dgpu DECIMAL(10,4),
    current_utilization DECIMAL(5,2) DEFAULT 0,
    temperature_c INTEGER,
    power_draw_w INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 5. Scheduler Orchestrator (Port: 8084)

**Teknoloji Stack:**
- **Dil**: Go 1.21+
- **Container Runtime**: Docker
- **Database**: PostgreSQL
- **Message Queue**: NATS JetStream

**Sorumluluklar:**
- Job scheduling ve queue management
- Docker container lifecycle management
- GPU resource allocation
- Job monitoring ve logging
- Auto-scaling decisions
- Failure recovery

**Database Schema:**
```sql
-- jobs table
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    provider_id UUID,
    gpu_id UUID,
    name VARCHAR(255) NOT NULL,
    docker_image VARCHAR(255),
    command TEXT,
    status VARCHAR(20) DEFAULT 'queued',
    progress_percent INTEGER DEFAULT 0,
    estimated_duration_minutes INTEGER,
    actual_duration_minutes INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

-- job_logs table
CREATE TABLE job_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID REFERENCES jobs(id),
    log_type VARCHAR(20) NOT NULL, -- stdout, stderr, system
    message TEXT NOT NULL,
    timestamp TIMESTAMP DEFAULT NOW()
);
```

## 🔄 Servis İletişimi

### Message Queue (NATS JetStream)

**Subjects:**
```
jobs.submit.{provider_id}      # Job submission
jobs.status.{job_id}          # Job status updates
billing.session.start        # Billing session events
billing.session.end          # Billing completion
providers.heartbeat          # Provider health checks
metrics.gpu.utilization      # GPU metrics
alerts.system.{severity}     # System alerts
```

### Service Discovery (Consul)

**Service Registration:**
```json
{
  "service": {
    "name": "auth-service",
    "port": 8090,
    "check": {
      "http": "http://localhost:8090/health",
      "interval": "10s"
    }
  }
}
```

## 📊 Monitoring ve Observability

### Metrics Collection (Prometheus)

**Custom Metrics:**
```
# Business Metrics
dante_active_jobs_total
dante_gpu_utilization_percent
dante_revenue_dgpu_total
dante_providers_online_total

# Technical Metrics
dante_api_requests_total
dante_api_request_duration_seconds
dante_database_connections_active
dante_queue_messages_pending
```

### Logging (Loki)

**Log Levels:**
- **ERROR**: System errors, failed transactions
- **WARN**: Performance issues, rate limits
- **INFO**: Business events, user actions
- **DEBUG**: Detailed technical information

### Alerting Rules

```yaml
# prometheus/alerts.yml
groups:
  - name: dante.rules
    rules:
      - alert: HighAPILatency
        expr: dante_api_request_duration_seconds > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High API latency detected"
      
      - alert: GPUProviderOffline
        expr: dante_providers_online_total < 1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "No GPU providers online"
```

## 🔒 Güvenlik Mimarisi

### Authentication Flow

```
1. User → API Gateway: Login request
2. API Gateway → Auth Service: Validate credentials
3. Auth Service → Database: Check user
4. Auth Service → API Gateway: JWT token
5. API Gateway → User: Token response
6. User → API Gateway: Authenticated requests (with JWT)
7. API Gateway: Validate JWT locally
```

### Authorization Matrix

| Role | Auth | Billing | Providers | Jobs | Storage |
|------|------|---------|-----------|------|---------|
| User | ✓ | Read Own | Read | CRUD Own | CRUD Own |
| Provider | ✓ | Read Own | CRUD Own | Read Assigned | Read Own |
| Admin | ✓ | Full | Full | Full | Full |

## 🚀 Deployment Stratejisi

### Container Orchestration

**Docker Compose (Development):**
```yaml
version: '3.8'
services:
  api-gateway:
    build: ./api-gateway
    ports: ["8080:8080"]
    depends_on: [consul, nats]
    
  auth-service:
    build: ./auth-service
    ports: ["8090:8090"]
    depends_on: [postgres]
```

**Kubernetes (Production):**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-gateway
  template:
    spec:
      containers:
      - name: api-gateway
        image: dante/api-gateway:latest
        ports:
        - containerPort: 8080
```

### Scaling Considerations

**Horizontal Scaling:**
- API Gateway: 3+ replicas behind load balancer
- Auth Service: 2+ replicas with session affinity
- Billing Service: 2+ replicas with distributed locks
- Provider Registry: 2+ replicas with eventual consistency

**Vertical Scaling:**
- Database: CPU/Memory optimization
- Redis: Memory scaling for cache
- NATS: Message throughput optimization

## 📈 Performance Optimizations

### Database Optimizations

```sql
-- Indexes for performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_user_id ON jobs(user_id);
CREATE INDEX idx_transactions_wallet ON transactions(from_wallet_id, to_wallet_id);
CREATE INDEX idx_billing_sessions_active ON billing_sessions(status) WHERE status = 'active';
```

### Caching Strategy

**Redis Cache Keys:**
```
user:profile:{user_id}        # TTL: 1 hour
gpu:availability:{provider_id} # TTL: 5 minutes
pricing:rates:{gpu_model}     # TTL: 15 minutes
job:status:{job_id}          # TTL: 30 seconds
```

### Connection Pooling

```go
// Database connection pool
db, err := sql.Open("postgres", dsn)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

Bu mimari dokümantasyonu, DanteGPU platformunun teknik detaylarını ve servis etkileşimlerini kapsamlı bir şekilde açıklamaktadır.
