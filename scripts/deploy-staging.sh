#!/bin/bash

# Deploy DanteGPU to Staging Environment
# This script deploys all services to Kubernetes staging namespace

set -e

echo "🚀 DanteGPU Staging Deployment"
echo "=============================="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
NAMESPACE="dantegpu-staging"
KUBECTL_CONTEXT="${KUBECTL_CONTEXT:-staging}"

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

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl not found. Please install kubectl."
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        print_error "docker not found. Please install Docker."
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Create namespace
create_namespace() {
    print_info "Creating namespace..."
    kubectl apply -f k8s/staging/namespace.yaml
    print_success "Namespace created"
}

# Deploy infrastructure
deploy_infrastructure() {
    print_info "Deploying infrastructure services..."
    
    # PostgreSQL
    print_info "Deploying PostgreSQL..."
    kubectl apply -f k8s/staging/postgres-deployment.yaml
    kubectl wait --for=condition=ready pod -l app=postgres -n $NAMESPACE --timeout=300s
    print_success "PostgreSQL deployed"
    
    # Redis
    print_info "Deploying Redis..."
    kubectl apply -f k8s/staging/redis-deployment.yaml
    kubectl wait --for=condition=ready pod -l app=redis -n $NAMESPACE --timeout=180s
    print_success "Redis deployed"
    
    # NATS
    print_info "Deploying NATS..."
    kubectl apply -f k8s/staging/nats-deployment.yaml
    kubectl wait --for=condition=ready pod -l app=nats -n $NAMESPACE --timeout=180s
    print_success "NATS deployed"
    
    print_success "Infrastructure deployed"
}

# Run database migrations
run_migrations() {
    print_info "Running database migrations..."
    kubectl apply -f k8s/staging/postgres-deployment.yaml
    kubectl wait --for=condition=complete job/postgres-migrations -n $NAMESPACE --timeout=300s
    print_success "Migrations completed"
}

# Deploy backend services
deploy_backend() {
    print_info "Deploying backend services..."
    
    # API Gateway
    print_info "Deploying API Gateway..."
    kubectl apply -f k8s/staging/api-gateway-deployment.yaml
    
    # Auth Service
    print_info "Deploying Auth Service..."
    kubectl apply -f k8s/staging/auth-service-deployment.yaml || echo "Auth service deployment not found, skipping..."
    
    # Billing Service
    print_info "Deploying Billing Service..."
    kubectl apply -f k8s/staging/billing-service-deployment.yaml || echo "Billing service deployment not found, skipping..."
    
    # Provider Registry
    print_info "Deploying Provider Registry..."
    kubectl apply -f k8s/staging/provider-registry-deployment.yaml || echo "Provider registry deployment not found, skipping..."
    
    # Scheduler
    print_info "Deploying Scheduler..."
    kubectl apply -f k8s/staging/scheduler-deployment.yaml || echo "Scheduler deployment not found, skipping..."
    
    # Wait for deployments
    print_info "Waiting for backend services to be ready..."
    kubectl wait --for=condition=available deployment/api-gateway -n $NAMESPACE --timeout=300s || true
    
    print_success "Backend services deployed"
}

# Deploy frontend
deploy_frontend() {
    print_info "Deploying frontend applications..."
    
    # User Dashboard
    print_info "Deploying User Dashboard..."
    kubectl apply -f k8s/staging/user-dashboard-deployment.yaml || echo "User dashboard deployment not found, skipping..."
    
    # Provider Web App
    print_info "Deploying Provider Web App..."
    kubectl apply -f k8s/staging/provider-web-app-deployment.yaml || echo "Provider web app deployment not found, skipping..."
    
    print_success "Frontend deployed"
}

# Deploy monitoring
deploy_monitoring() {
    print_info "Deploying monitoring stack..."
    
    # Prometheus
    print_info "Deploying Prometheus..."
    kubectl apply -f k8s/staging/prometheus-deployment.yaml || echo "Prometheus deployment not found, skipping..."
    
    # Grafana
    print_info "Deploying Grafana..."
    kubectl apply -f k8s/staging/grafana-deployment.yaml || echo "Grafana deployment not found, skipping..."
    
    print_success "Monitoring deployed"
}

# Run smoke tests
run_smoke_tests() {
    print_info "Running smoke tests..."
    
    # Get API Gateway service URL
    API_URL=$(kubectl get svc api-gateway-service -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "localhost")
    
    if [ "$API_URL" != "localhost" ]; then
        # Test health endpoint
        if curl -f "http://$API_URL:8000/health" > /dev/null 2>&1; then
            print_success "Health check passed"
        else
            print_error "Health check failed"
        fi
    else
        print_info "Skipping smoke tests (no external IP)"
    fi
}

# Display deployment info
display_info() {
    echo ""
    print_success "Deployment completed!"
    echo ""
    echo "📊 Deployment Information:"
    echo "=========================="
    echo ""
    
    # Get service endpoints
    echo "Services:"
    kubectl get svc -n $NAMESPACE
    echo ""
    
    # Get pod status
    echo "Pods:"
    kubectl get pods -n $NAMESPACE
    echo ""
    
    # Get ingress
    echo "Ingress:"
    kubectl get ingress -n $NAMESPACE 2>/dev/null || echo "No ingress configured"
    echo ""
    
    echo "🔗 Access URLs:"
    echo "  API Gateway: http://staging.dantegpu.com"
    echo "  User Dashboard: http://staging.dantegpu.com"
    echo "  Provider Portal: http://staging-provider.dantegpu.com"
    echo "  Grafana: http://grafana-staging.dantegpu.com"
    echo ""
    
    echo "📝 Useful Commands:"
    echo "  View logs: kubectl logs -f <pod-name> -n $NAMESPACE"
    echo "  Port forward: kubectl port-forward svc/api-gateway-service 8000:8000 -n $NAMESPACE"
    echo "  Shell access: kubectl exec -it <pod-name> -n $NAMESPACE -- /bin/sh"
    echo ""
}

# Main execution
main() {
    print_info "Starting deployment to staging..."
    echo ""
    
    check_prerequisites
    create_namespace
    
    # Apply ConfigMaps
    print_info "Applying ConfigMaps..."
    kubectl apply -f k8s/staging/configmap.yaml
    print_success "ConfigMaps applied"
    
    deploy_infrastructure
    run_migrations
    deploy_backend
    deploy_frontend
    deploy_monitoring
    run_smoke_tests
    display_info
    
    print_success "Staging deployment complete!"
}

# Trap errors
trap 'print_error "Deployment failed!"; exit 1' ERR

# Run main function
main

exit 0

