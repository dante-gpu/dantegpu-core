# API Gateway Service

The API Gateway serves as the central entry point for all client requests to the Dante GPU Platform. It handles authentication, request routing, billing endpoints, and service discovery with comprehensive dGPU token integration.

## Features

### Core Functionality
- Request routing to appropriate backend services
- JWT authentication and authorization
- Rate limiting and abuse prevention
- CORS support for web applications
- Service discovery via Consul
- Load balancing across service instances
- Health monitoring and checks
- Comprehensive request/response logging

### Billing Integration
- dGPU wallet management endpoints
- Token deposit and withdrawal operations
- GPU marketplace browsing
- Real-time pricing and cost estimation
- Transaction history and balance checking
- Integration with Solana blockchain

## Architecture

The API Gateway integrates with:
- Chi Router - Fast HTTP router with middleware support
- Consul - Service discovery and health checking
- NATS - Message publishing for job submissions
- Billing Service - dGPU token operations
- Zap - Structured logging with correlation IDs

## Configuration

### Environment Variables

```bash
# Server Configuration
PORT=8080
HOST=0.0.0.0

# JWT Configuration
JWT_SECRET=your-secret-key

# Consul Configuration
CONSUL_ADDRESS=localhost:8500

# NATS Configuration
NATS_URL=nats://localhost:4222

# Billing Service
BILLING_SERVICE_URL=http://localhost:8080

# Service URLs (fallback if Consul unavailable)
AUTH_SERVICE_URL=http://localhost:8000
PROVIDER_REGISTRY_URL=http://localhost:8001
STORAGE_SERVICE_URL=http://localhost:8002
SCHEDULER_SERVICE_URL=http://localhost:8003

# Rate Limiting
RATE_LIMIT_REQUESTS_PER_MINUTE=100
RATE_LIMIT_BURST=20

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001
```

## API Endpoints

### Public Endpoints

#### Health Check
```
GET /health
GET /health/ready
GET /health/live
```

#### Authentication
```
POST /api/v1/auth/login      # User login
POST /api/v1/auth/register   # User registration
POST /api/v1/auth/refresh    # Token refresh
```

### Protected Endpoints (Require JWT)

#### Job Management
```
POST /api/v1/jobs            # Submit GPU rental job
GET /api/v1/jobs/{jobID}     # Get job status
DELETE /api/v1/jobs/{jobID}  # Cancel job
```

#### Billing & Payments
```
POST /api/v1/billing/wallet                    # Create dGPU wallet
GET /api/v1/billing/wallet/{walletID}/balance  # Check wallet balance
POST /api/v1/billing/wallet/{walletID}/deposit # Deposit dGPU tokens
POST /api/v1/billing/wallet/{walletID}/withdraw # Withdraw dGPU tokens
GET /api/v1/billing/wallet/{walletID}/transactions # Transaction history

GET /api/v1/billing/user/{userID}/wallet       # Get user wallet
GET /api/v1/billing/user/{userID}/balance      # Get user balance

GET /api/v1/billing/pricing/rates              # Get current pricing rates
POST /api/v1/billing/pricing/calculate         # Calculate pricing
POST /api/v1/billing/pricing/estimate          # Estimate job cost
GET /api/v1/billing/marketplace                # Browse available GPUs
```

#### Provider Management
```
GET /api/v1/providers                    # List available providers
POST /api/v1/providers                   # Register GPU provider
PUT /api/v1/providers/{providerID}       # Update provider
DELETE /api/v1/providers/{providerID}    # Remove provider
```

#### Storage Operations
```
PUT /api/v1/storage/{bucket}/{key}       # Upload file
GET /api/v1/storage/{bucket}/{key}       # Download file
DELETE /api/v1/storage/{bucket}/{key}    # Delete file
GET /api/v1/storage/{bucket}             # List bucket contents
```

## Installation

### Prerequisites

- Go 1.21 or higher
- Consul for service discovery
- NATS for message publishing
- Access to billing service

### Build and Run

```bash
# Clone and build
git clone <repository-url>
cd api-gateway
go mod download
go build -o bin/api-gateway cmd/main.go

# Set environment variables
export JWT_SECRET="your-secret-key"
export CONSUL_ADDRESS="localhost:8500"
export NATS_URL="nats://localhost:4222"
export BILLING_SERVICE_URL="http://localhost:8080"

# Run the service
./bin/api-gateway
```

### Docker Deployment

```bash
# Build Docker image
docker build -t api-gateway .

# Run container
docker run -p 8080:8080 \
  -e JWT_SECRET="your-secret-key" \
  -e CONSUL_ADDRESS="consul:8500" \
  -e NATS_URL="nats://nats:4222" \
  -e BILLING_SERVICE_URL="http://billing:8080" \
  api-gateway
```

## Development

### Code Structure

```
api-gateway/
├── cmd/main.go                    # Application entry point
├── internal/
│   ├── billing/                   # Billing service client
│   ├── config/                    # Configuration management
│   ├── consul/                    # Service discovery
│   ├── handlers/                  # HTTP handlers
│   │   ├── auth.go               # Authentication handlers
│   │   ├── billing.go            # Billing endpoints
│   │   ├── job.go                # Job management
│   │   └── proxy.go              # Service proxying
│   ├── loadbalancer/             # Load balancing
│   ├── middleware/               # Custom middleware
│   └── nats/                     # Message publishing
├── configs/config.yaml           # Configuration file
└── Dockerfile                    # Container configuration
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

## Security

### JWT Authentication
- RS256/HS256 token signing
- Configurable token expiration
- Automatic token validation
- Role-based access control

### Rate Limiting
- Per-IP request limiting
- Configurable burst sizes
- Protection against abuse
- Graceful degradation

### CORS Configuration
- Configurable allowed origins
- Method and header restrictions
- Preflight request handling
- Security header injection

## Monitoring and Observability

### Health Checks
- Basic health endpoint
- Readiness checks with dependencies
- Liveness checks for container orchestration

### Metrics
- Request count and latency
- Error rates by endpoint
- Service discovery status
- Rate limiting statistics
- Billing operation metrics

### Logging
- Structured JSON logging
- Request correlation IDs
- User context tracking
- Error stack traces
- Performance metrics

## Billing Integration

### Wallet Operations
- Create and manage dGPU wallets
- Real-time balance checking
- Secure token transfers
- Transaction history tracking

### Marketplace Features
- Browse available GPUs with pricing
- Filter by specifications and location
- Real-time availability updates
- Cost estimation tools

### Pricing Engine
- Dynamic pricing based on demand
- GPU model-specific rates
- VRAM and power consumption factors
- Platform fee calculation

## Troubleshooting

### Common Issues

1. **Service Discovery Problems**
   - Verify Consul connectivity
   - Check service registration
   - Validate health check endpoints

2. **Authentication Failures**
   - Confirm JWT secret configuration
   - Check token expiration settings
   - Validate token format and claims

3. **Billing Service Errors**
   - Verify billing service connectivity
   - Check dGPU token configuration
   - Validate Solana network settings

4. **Rate Limiting Issues**
   - Review rate limit configuration
   - Monitor request patterns
   - Adjust limits for legitimate traffic

### Debug Configuration

Enable detailed logging:

```bash
export LOG_LEVEL=debug
export LOG_FORMAT=json
./bin/api-gateway
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Implement changes with tests
4. Update documentation
5. Submit a pull request

### Development Guidelines
- Follow Go coding standards
- Add comprehensive tests
- Update API documentation
- Ensure backward compatibility

## License

This project is licensed under the MIT License.
