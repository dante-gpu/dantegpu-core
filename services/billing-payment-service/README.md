# Billing & Payment Service

A comprehensive financial management service for the Dante GPU Platform that handles dGPU token transactions, real-time billing, dynamic pricing, and Solana blockchain integration.

## Overview

The Billing & Payment Service is the financial backbone of the Dante GPU Platform, providing complete payment processing, wallet management, and billing operations using dGPU tokens on the Solana blockchain.

## Core Responsibilities

### Blockchain Integration
- dGPU token transaction processing (7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump)
- Solana blockchain interaction and monitoring
- Wallet creation and management
- Transaction verification and confirmation
- Multi-signature support for security

### Real-time Billing System
- Session-based billing with automatic monitoring
- Usage tracking with 1-minute precision
- Insufficient funds protection and grace periods
- Automatic session termination on balance depletion
- Real-time cost calculation and updates

### Dynamic Pricing Engine
- GPU model-specific base rates
- VRAM allocation-based pricing (per GB per hour)
- Power consumption multipliers
- Dynamic demand and supply adjustments
- Platform fee calculation and collection

### Wallet Management
- User and provider dGPU token wallets
- Balance tracking and transaction history
- Deposit and withdrawal operations
- Security features and fraud protection

## Technology Stack

- Go 1.21+ - Primary service language
- PostgreSQL - Transaction and billing data storage
- Solana Blockchain - dGPU token transactions
- NATS JetStream - Real-time event streaming
- Consul - Service discovery and configuration
- Prometheus - Metrics and monitoring

## Key Features

### dGPU Token Integration
- Real-time token balance checking
- Secure transaction processing
- Multi-signature wallet support
- Transaction fee optimization

### Dynamic Pricing Engine
- Base rates by GPU model and VRAM
- Power consumption multipliers
- Market demand adjustments
- Provider-set minimum rates

### VRAM Allocation Management
- Fractional VRAM rental (e.g., 50% of 24GB = 12GB)
- Per-GB pricing calculations
- Real-time allocation tracking
- Automatic billing adjustments

### Usage Monitoring
- Real-time GPU utilization tracking
- Precise billing down to the minute
- Power consumption monitoring
- Automatic session termination on insufficient funds

## API Endpoints

### User Wallet Management
- `GET /api/v1/wallet/balance` - Get user's dGPU token balance
- `POST /api/v1/wallet/deposit` - Initiate token deposit
- `POST /api/v1/wallet/withdraw` - Request token withdrawal
- `GET /api/v1/wallet/transactions` - Get transaction history

### Billing & Usage
- `POST /api/v1/billing/start-session` - Start GPU rental session
- `POST /api/v1/billing/end-session` - End GPU rental session
- `GET /api/v1/billing/current-usage` - Get current session costs
- `GET /api/v1/billing/history` - Get billing history

### Provider Payouts
- `GET /api/v1/provider/earnings` - Get provider earnings
- `POST /api/v1/provider/payout` - Request payout
- `GET /api/v1/provider/rates` - Get/set provider rates

### Pricing
- `GET /api/v1/pricing/rates` - Get current GPU rental rates
- `POST /api/v1/pricing/calculate` - Calculate cost for specific requirements

## Configuration

```yaml
# Solana Configuration
solana:
  rpc_url: "https://api.mainnet-beta.solana.com"
  dgpu_token_address: "7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump"
  platform_wallet: "YOUR_PLATFORM_WALLET_ADDRESS"
  private_key_path: "/secrets/solana_private_key"

# Database Configuration
database:
  url: "postgres://user:password@localhost:5432/dante_billing"
  max_connections: 25
  connection_timeout: "30s"

# Pricing Configuration
pricing:
  base_rates:
    nvidia_rtx_4090: 0.50  # dGPU tokens per hour
    nvidia_a100: 2.00
    nvidia_h100: 4.00
  power_multiplier: 0.001  # Additional cost per watt
  platform_fee_percent: 5.0
  minimum_session_minutes: 1

# NATS Configuration
nats:
  address: "nats://localhost:4222"
  usage_updates_subject: "dante.billing.usage"
  payment_events_subject: "dante.billing.payments"

# Service Configuration
server:
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "60s"

# Consul Configuration
consul:
  address: "localhost:8500"
  service_name: "billing-payment-service"
  health_check_interval: "10s"
```

## Database Schema

### Tables
- `wallets` - User and provider dGPU token wallets
- `rental_sessions` - Active and completed GPU rental sessions
- `transactions` - All dGPU token transactions
- `usage_records` - Detailed usage tracking
- `provider_rates` - Custom provider pricing
- `billing_history` - Aggregated billing records

## Security Considerations

- Private keys stored in secure key management system
- Multi-signature transactions for large amounts
- Rate limiting on API endpoints
- Audit logging for all financial operations
- Real-time fraud detection

## Integration Points

- **Provider Registry Service**: Get GPU specifications and availability
- **Scheduler Service**: Receive job start/end notifications
- **Auth Service**: Validate user permissions
- **Monitoring Service**: Track service health and performance
- **Solana Blockchain**: Execute dGPU token transactions

## Installation and Setup

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14+
- Solana CLI tools
- NATS Server 2.9+
- Consul 1.15+

### Environment Configuration

```bash
# Solana Configuration
export SOLANA_RPC_URL="https://api.mainnet-beta.solana.com"
export DGPU_TOKEN_ADDRESS="7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump"
export SOLANA_PRIVATE_KEY="your_base58_private_key"

# Database Configuration
export DATABASE_URL="postgres://user:pass@localhost:5432/dante_billing"

# Service Configuration
export PORT="8080"
export NATS_URL="nats://localhost:4222"
export CONSUL_URL="http://localhost:8500"
```

### Build and Run

```bash
# Clone and build
git clone <repository-url>
cd billing-payment-service
go mod download
go build -o bin/billing-service cmd/main.go

# Run database migrations
migrate -path migrations -database $DATABASE_URL up

# Start the service
./bin/billing-service
```

### Docker Deployment

```bash
# Build Docker image
docker build -t billing-payment-service .

# Run container
docker run -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@db:5432/dante_billing" \
  -e SOLANA_RPC_URL="https://api.mainnet-beta.solana.com" \
  -e DGPU_TOKEN_ADDRESS="7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump" \
  billing-payment-service
```

## Development

### Code Structure

```
billing-payment-service/
├── cmd/main.go                    # Application entry point
├── internal/
│   ├── config/                    # Configuration management
│   ├── handlers/                  # HTTP request handlers
│   ├── models/                    # Data models and schemas
│   ├── pricing/                   # Pricing engine
│   ├── service/                   # Business logic
│   ├── solana/                    # Blockchain integration
│   └── store/                     # Database operations
├── migrations/                    # Database migrations
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

### Database Migrations

```bash
# Create new migration
migrate create -ext sql -dir migrations -seq add_new_table

# Run migrations
migrate -path migrations -database $DATABASE_URL up

# Rollback migrations
migrate -path migrations -database $DATABASE_URL down 1
```

## Production Deployment

### Security Considerations

- Store Solana private keys in secure key management systems
- Use multi-signature wallets for large transactions
- Enable database encryption at rest
- Configure rate limiting and DDoS protection
- Regular security audits and penetration testing

### Monitoring and Alerting

- Set up Prometheus metrics collection
- Configure Grafana dashboards for financial metrics
- Enable alerts for transaction failures
- Monitor blockchain network status
- Track wallet balance thresholds

### Backup and Recovery

- Regular database backups with point-in-time recovery
- Secure backup of wallet private keys
- Disaster recovery procedures
- Transaction replay capabilities

## Contributing

1. Fork the repository
2. Create a feature branch
3. Implement changes with comprehensive tests
4. Update documentation
5. Submit a pull request

### Development Guidelines

- Follow Go coding standards and best practices
- Add unit and integration tests for all features
- Update API documentation for endpoint changes
- Ensure backward compatibility
- Test blockchain integration thoroughly

## License

This project is licensed under the MIT License.
