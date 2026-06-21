# DanteGPU Core - API Documentation


## Overview

DanteGPU Core provides a comprehensive REST API for managing GPU rentals, blockchain transactions, and job execution on a decentralized platform.

**Base URL**: `https://api.dantegpu.com/api/v1`  
**Staging URL**: `https://staging.dantegpu.com/api/v1`

## Authentication

All API requests require authentication using JWT tokens.

### Login

```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

### Using Tokens

Include the access token in the Authorization header:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

### Refresh Token

```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

---

## Authentication Endpoints

### Register User

```http
POST /auth/register
Content-Type: application/json

{
  "email": "newuser@example.com",
  "password": "SecurePassword123!",
  "first_name": "John",
  "last_name": "Doe"
}
```

**Response:** `201 Created`

### Verify Email

```http
POST /auth/verify-email
Content-Type: application/json

{
  "token": "verification-token-from-email"
}
```

### Password Reset Request

```http
POST /auth/password-reset
Content-Type: application/json

{
  "email": "user@example.com"
}
```

### Password Reset Confirm

```http
POST /auth/password-reset/confirm
Content-Type: application/json

{
  "token": "reset-token-from-email",
  "new_password": "NewSecurePassword123!"
}
```

### Enable 2FA

```http
POST /auth/2fa/enable
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "method": "totp"
}
```

**Response:**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code": "data:image/png;base64,...",
  "backup_codes": ["12345678", "87654321", ...]
}
```

---

## Wallet Endpoints

### Create Wallet

```http
POST /wallet/create
Authorization: Bearer {access_token}
```

**Response:**
```json
{
  "wallet_address": "7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump",
  "balance": 0,
  "created_at": "2025-10-06T12:00:00Z"
}
```

### Get Wallet

```http
GET /wallet
Authorization: Bearer {access_token}
```

**Response:**
```json
{
  "wallet_address": "7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump",
  "balance": 100.50,
  "available_balance": 95.25,
  "locked_balance": 5.25
}
```

### Get Transaction History

```http
GET /wallet/transactions?limit=50&offset=0
Authorization: Bearer {access_token}
```

**Response:**
```json
{
  "transactions": [
    {
      "id": "tx-123",
      "type": "deposit",
      "amount": 100.00,
      "status": "confirmed",
      "signature": "5VERv8NMvzbJMEkV8xnrLkEaWRtSz9CosKDYjCJjBRnb...",
      "created_at": "2025-10-06T12:00:00Z"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

---

## GPU Marketplace Endpoints

### List GPUs

```http
GET /gpus?available=true&gpu_model=RTX 4090&min_memory=16&max_price=0.10
Authorization: Bearer {access_token}
```

**Query Parameters:**
- `available` (boolean): Filter by availability
- `gpu_model` (string): Filter by GPU model
- `min_memory` (number): Minimum GPU memory in GB
- `max_price` (number): Maximum price per minute
- `limit` (number): Results per page (default: 20)
- `offset` (number): Pagination offset

**Response:**
```json
{
  "gpus": [
    {
      "id": "gpu-123",
      "provider_id": "provider-456",
      "gpu_model": "NVIDIA RTX 4090",
      "gpu_memory_gb": 24,
      "compute_capability": "8.9",
      "price_per_minute": 0.05,
      "is_available": true,
      "status": "online",
      "cuda_cores": 16384,
      "tensor_cores": 512
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

### Get GPU Details

```http
GET /gpus/{gpu_id}
Authorization: Bearer {access_token}
```

---

## Rental Endpoints

### Start Rental

```http
POST /billing/start-rental
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "gpu_id": "gpu-123",
  "estimated_hours": 2
}
```

**Response:**
```json
{
  "session_id": "session-789",
  "gpu_id": "gpu-123",
  "escrow_amount": 6.00,
  "status": "active",
  "started_at": "2025-10-06T12:00:00Z"
}
```

### End Rental

```http
POST /billing/end-rental/{session_id}
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "reason": "completed"
}
```

### Get Billing History

```http
GET /billing/history?limit=50&offset=0
Authorization: Bearer {access_token}
```

---

## Job Endpoints

### Submit Job

```http
POST /jobs
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "gpu_id": "gpu-123",
  "docker_image": "tensorflow/tensorflow:latest-gpu",
  "command": "python train.py",
  "gpu_count": 1,
  "estimated_duration_hours": 2,
  "environment_variables": {
    "CUDA_VISIBLE_DEVICES": "0"
  }
}
```

**Response:**
```json
{
  "job_id": "job-456",
  "status": "pending",
  "created_at": "2025-10-06T12:00:00Z"
}
```

### Get Job Status

```http
GET /jobs/{job_id}
Authorization: Bearer {access_token}
```

### List Jobs

```http
GET /jobs?status=running&limit=20&offset=0
Authorization: Bearer {access_token}
```

### Cancel Job

```http
POST /jobs/{job_id}/cancel
Authorization: Bearer {access_token}
```

### Get Job Logs

```http
GET /jobs/{job_id}/logs?tail=100
Authorization: Bearer {access_token}
```

**Response:**
```json
{
  "logs": [
    {
      "timestamp": "2025-10-06T12:00:00Z",
      "level": "INFO",
      "message": "Starting job execution"
    }
  ]
}
```

---

## Provider Endpoints

### Register as Provider

```http
POST /providers/register
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "company_name": "GPU Cloud Inc",
  "contact_email": "contact@gpucloud.com",
  "description": "Professional GPU provider"
}
```

### Add GPU Capability

```http
POST /providers/{provider_id}/gpus
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "gpu_model": "NVIDIA RTX 4090",
  "gpu_memory_gb": 24,
  "compute_capability": "8.9",
  "price_per_minute": 0.05,
  "cuda_cores": 16384,
  "tensor_cores": 512
}
```

### Get Provider Statistics

```http
GET /providers/{provider_id}/statistics
Authorization: Bearer {access_token}
```

**Response:**
```json
{
  "total_gpus": 5,
  "available_gpus": 3,
  "total_earnings": 1250.50,
  "total_rentals": 42,
  "average_rating": 4.8
}
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "Error message",
  "error_code": "ERROR_CODE",
  "details": {
    "field": "Additional error details"
  }
}
```

### Common Error Codes

- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource conflict (e.g., duplicate email)
- `422 Unprocessable Entity`: Validation error
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

---

## Rate Limiting

API requests are rate-limited to:
- **60 requests per minute** for authenticated users
- **10 requests per minute** for unauthenticated requests

Rate limit headers:
```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1696598400
```

---

## WebSocket API

Connect to real-time updates:

```javascript
const ws = new WebSocket('wss://api.dantegpu.com/ws');

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'auth',
    token: 'your-access-token'
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
};
```

### WebSocket Events

- `job.status`: Job status updates
- `job.logs`: Real-time job logs
- `gpu.metrics`: GPU metrics updates
- `billing.update`: Billing updates
- `notification`: User notifications

---

## SDKs

Official SDKs available:
- **JavaScript/TypeScript**: `npm install @dantegpu/sdk`
- **Python**: `pip install dantegpu`
- **Go**: `go get github.com/dante-gpu/go-sdk`

---

For more information, visit [docs.dantegpu.com](https://docs.dantegpu.com)

