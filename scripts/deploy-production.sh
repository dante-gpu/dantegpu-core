#!/bin/bash

# Deploy DanteGPU to Production Environment
# Blue-Green Deployment Strategy

set -e

echo "🚀 DanteGPU Production Deployment (Blue-Green)"
echo "=============================================="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
NAMESPACE="dantegpu-production"
CURRENT_COLOR="${1:-blue}"
NEW_COLOR="green"
if [ "$CURRENT_COLOR" = "green" ]; then
    NEW_COLOR="blue"
fi

# Functions
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

print_step() {
    echo -e "${BLUE}▶️  $1${NC}"
}

# Confirmation
echo ""
print_info "Current environment: $CURRENT_COLOR"
print_info "Deploying to: $NEW_COLOR"
echo ""
read -p "Continue with deployment? (yes/no): " -r
if [[ ! $REPLY =~ ^[Yy]es$ ]]; then
    print_error "Deployment cancelled"
    exit 1
fi

# Step 1: Backup database
print_step "Step 1: Backing up database..."
./scripts/backup-database.sh
print_success "Database backup completed"

# Step 2: Deploy to new color
print_step "Step 2: Deploying to $NEW_COLOR environment..."
kubectl apply -f k8s/production/$NEW_COLOR/
print_success "$NEW_COLOR environment deployed"

# Step 3: Wait for deployments to be ready
print_step "Step 3: Waiting for deployments to be ready..."
kubectl wait --for=condition=available --timeout=600s \
    deployment/api-gateway-$NEW_COLOR \
    deployment/auth-service-$NEW_COLOR \
    deployment/billing-service-$NEW_COLOR \
    -n $NAMESPACE
print_success "All deployments ready"

# Step 4: Run smoke tests on new environment
print_step "Step 4: Running smoke tests on $NEW_COLOR..."
NEW_URL="https://$NEW_COLOR.dantegpu.com"
if ./scripts/smoke-tests.sh "$NEW_URL"; then
    print_success "Smoke tests passed"
else
    print_error "Smoke tests failed!"
    print_info "Rolling back..."
    kubectl delete -f k8s/production/$NEW_COLOR/
    exit 1
fi

# Step 5: Switch traffic
print_step "Step 5: Switching traffic to $NEW_COLOR..."
kubectl patch service api-gateway -n $NAMESPACE \
    -p "{\"spec\":{\"selector\":{\"version\":\"$NEW_COLOR\"}}}"
kubectl patch service user-dashboard -n $NAMESPACE \
    -p "{\"spec\":{\"selector\":{\"version\":\"$NEW_COLOR\"}}}"
kubectl patch service provider-web-app -n $NAMESPACE \
    -p "{\"spec\":{\"selector\":{\"version\":\"$NEW_COLOR\"}}}"
print_success "Traffic switched to $NEW_COLOR"

# Step 6: Monitor for 5 minutes
print_step "Step 6: Monitoring $NEW_COLOR environment..."
print_info "Monitoring for 5 minutes. Press Ctrl+C to abort and rollback."

for i in {1..30}; do
    sleep 10
    
    # Check pod status
    READY_PODS=$(kubectl get pods -n $NAMESPACE -l version=$NEW_COLOR --no-headers | grep "Running" | wc -l)
    TOTAL_PODS=$(kubectl get pods -n $NAMESPACE -l version=$NEW_COLOR --no-headers | wc -l)
    
    # Check error rate
    ERROR_RATE=$(kubectl logs -n $NAMESPACE -l app=api-gateway,version=$NEW_COLOR --tail=100 | grep -c "ERROR" || echo "0")
    
    echo -ne "\r⏱️  ${i}0s - Pods: $READY_PODS/$TOTAL_PODS - Errors: $ERROR_RATE"
    
    if [ "$READY_PODS" -lt "$TOTAL_PODS" ]; then
        echo ""
        print_error "Some pods are not ready!"
        print_info "Rolling back to $CURRENT_COLOR..."
        kubectl patch service api-gateway -n $NAMESPACE \
            -p "{\"spec\":{\"selector\":{\"version\":\"$CURRENT_COLOR\"}}}"
        exit 1
    fi
done

echo ""
print_success "Monitoring completed - No issues detected"

# Step 7: Delete old environment
print_step "Step 7: Cleaning up $CURRENT_COLOR environment..."
read -p "Delete $CURRENT_COLOR environment? (yes/no): " -r
if [[ $REPLY =~ ^[Yy]es$ ]]; then
    kubectl delete -f k8s/production/$CURRENT_COLOR/ || true
    print_success "$CURRENT_COLOR environment deleted"
else
    print_info "Keeping $CURRENT_COLOR environment for manual cleanup"
fi

# Summary
echo ""
print_success "🎉 Production deployment completed successfully!"
echo ""
echo "📊 Deployment Summary:"
echo "  Previous: $CURRENT_COLOR"
echo "  Current: $NEW_COLOR"
echo "  URL: https://dantegpu.com"
echo ""
echo "📝 Next Steps:"
echo "  1. Monitor Grafana: https://grafana.dantegpu.com"
echo "  2. Check logs: kubectl logs -f -l version=$NEW_COLOR -n $NAMESPACE"
echo "  3. Monitor alerts: https://alertmanager.dantegpu.com"
echo ""
echo "🔄 Rollback Command (if needed):"
echo "  kubectl patch service api-gateway -n $NAMESPACE -p '{\"spec\":{\"selector\":{\"version\":\"$CURRENT_COLOR\"}}}'"
echo ""

exit 0

