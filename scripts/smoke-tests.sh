#!/bin/bash

# Smoke Tests for DanteGPU Platform
# Tests basic functionality after deployment

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
BASE_URL="${1:-http://localhost:8000}"
TIMEOUT=10

# Counters
PASSED=0
FAILED=0

# Functions
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
    ((PASSED++))
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
    ((FAILED++))
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# Test function
test_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="${3:-200}"
    local method="${4:-GET}"
    
    print_info "Testing: $name"
    
    response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" --max-time $TIMEOUT 2>/dev/null || echo "000")
    status_code=$(echo "$response" | tail -n1)
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "$name - Status: $status_code"
        return 0
    else
        print_error "$name - Expected: $expected_status, Got: $status_code"
        return 1
    fi
}

# Main tests
echo "🧪 DanteGPU Smoke Tests"
echo "======================="
echo "Base URL: $BASE_URL"
echo ""

# Health checks
print_info "=== Health Checks ==="
test_endpoint "API Gateway Health" "$BASE_URL/health" 200
test_endpoint "API Gateway Ready" "$BASE_URL/ready" 200

# Authentication endpoints
print_info "=== Authentication Endpoints ==="
test_endpoint "Login Endpoint" "$BASE_URL/api/v1/auth/login" 400 POST
test_endpoint "Register Endpoint" "$BASE_URL/api/v1/auth/register" 400 POST

# GPU Marketplace (requires auth, should return 401)
print_info "=== GPU Marketplace ==="
test_endpoint "List GPUs (Unauthorized)" "$BASE_URL/api/v1/gpus" 401

# Wallet endpoints (requires auth)
print_info "=== Wallet Endpoints ==="
test_endpoint "Get Wallet (Unauthorized)" "$BASE_URL/api/v1/wallet" 401

# Job endpoints (requires auth)
print_info "=== Job Endpoints ==="
test_endpoint "List Jobs (Unauthorized)" "$BASE_URL/api/v1/jobs" 401

# Provider endpoints (requires auth)
print_info "=== Provider Endpoints ==="
test_endpoint "List Providers" "$BASE_URL/api/v1/providers" 200

# Test with valid registration
print_info "=== Full Auth Flow Test ==="
TEST_EMAIL="smoketest_$(date +%s)@example.com"
TEST_PASSWORD="SmokeTest123!"

# Register user
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\",\"first_name\":\"Smoke\",\"last_name\":\"Test\"}" \
    --max-time $TIMEOUT 2>/dev/null || echo "")

if echo "$REGISTER_RESPONSE" | grep -q "success\|created\|registered"; then
    print_success "User Registration"
else
    print_error "User Registration"
fi

# Try login (will fail without email verification, but endpoint should work)
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}" \
    --max-time $TIMEOUT 2>/dev/null || echo "")

if echo "$LOGIN_RESPONSE" | grep -q "token\|verify\|email"; then
    print_success "Login Endpoint Response"
else
    print_error "Login Endpoint Response"
fi

# WebSocket test
print_info "=== WebSocket Test ==="
if command -v wscat &> /dev/null; then
    WS_URL="${BASE_URL/http/ws}/ws"
    timeout 5 wscat -c "$WS_URL" -x '{"type":"ping"}' &>/dev/null && \
        print_success "WebSocket Connection" || \
        print_error "WebSocket Connection"
else
    print_info "wscat not installed, skipping WebSocket test"
fi

# Database connectivity (indirect test via API)
print_info "=== Database Connectivity ==="
if curl -s "$BASE_URL/api/v1/providers" --max-time $TIMEOUT | grep -q "\[\]"; then
    print_success "Database Connection (via API)"
else
    print_error "Database Connection (via API)"
fi

# Summary
echo ""
echo "======================="
echo "📊 Test Summary"
echo "======================="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo "Total: $((PASSED + FAILED))"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All smoke tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed!${NC}"
    exit 1
fi

