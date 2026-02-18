# DanteGPU Core - Operations Runbook

{🙏 Don't worry about what the f😳ck I be doing, I'm Mock King AKINCI}

## Table of Contents

1. [Common Operations](#common-operations)
2. [Incident Response](#incident-response)
3. [Deployment Procedures](#deployment-procedures)
4. [Monitoring and Alerts](#monitoring-and-alerts)
5. [Database Operations](#database-operations)
6. [Troubleshooting](#troubleshooting)

---

## Common Operations

### Checking System Health

```bash
# Check all pods status
kubectl get pods -n dantegpu-production

# Check service endpoints
kubectl get svc -n dantegpu-production

# Check recent events
kubectl get events -n dantegpu-production --sort-by='.lastTimestamp'

# Check resource usage
kubectl top pods -n dantegpu-production
kubectl top nodes
```

### Viewing Logs

```bash
# View logs for a specific service
kubectl logs -f deployment/api-gateway -n dantegpu-production

# View logs for all replicas
kubectl logs -f -l app=api-gateway -n dantegpu-production

# View logs from last hour
kubectl logs --since=1h deployment/api-gateway -n dantegpu-production

# Search logs for errors
kubectl logs deployment/api-gateway -n dantegpu-production | grep ERROR
```

### Scaling Services

```bash
# Manual scaling
kubectl scale deployment api-gateway --replicas=5 -n dantegpu-production

# Check HPA status
kubectl get hpa -n dantegpu-production

# Describe HPA for details
kubectl describe hpa api-gateway-hpa -n dantegpu-production
```

### Restarting Services

```bash
# Rolling restart
kubectl rollout restart deployment/api-gateway -n dantegpu-production

# Check rollout status
kubectl rollout status deployment/api-gateway -n dantegpu-production

# Rollback if needed
kubectl rollout undo deployment/api-gateway -n dantegpu-production
```

---

## Incident Response

### Service Down Alert

**Symptoms**: Prometheus alert "ServiceDown", 5xx errors, no response

**Investigation**:
```bash
# 1. Check pod status
kubectl get pods -n dantegpu-production -l app=<service-name>

# 2. Check pod logs
kubectl logs <pod-name> -n dantegpu-production --tail=100

# 3. Describe pod for events
kubectl describe pod <pod-name> -n dantegpu-production

# 4. Check resource usage
kubectl top pod <pod-name> -n dantegpu-production
```

**Resolution**:
```bash
# If pod is CrashLoopBackOff
kubectl logs <pod-name> -n dantegpu-production --previous

# If OOMKilled
kubectl edit deployment <service-name> -n dantegpu-production
# Increase memory limits

# If ImagePullBackOff
kubectl describe pod <pod-name> -n dantegpu-production
# Check image name and registry credentials

# Force restart
kubectl delete pod <pod-name> -n dantegpu-production
```

### High Error Rate Alert

**Symptoms**: Prometheus alert "HighErrorRate", increased 5xx responses

**Investigation**:
```bash
# 1. Check error logs
kubectl logs -f deployment/<service-name> -n dantegpu-production | grep ERROR

# 2. Check Grafana dashboard
# Open: https://grafana.dantegpu.com/d/service-health

# 3. Check database connections
kubectl exec -it deployment/api-gateway -n dantegpu-production -- \
  psql -h postgres-service -U dante_user -d dante_core -c \
  "SELECT count(*) FROM pg_stat_activity;"

# 4. Check Redis
kubectl exec -it deployment/redis -n dantegpu-production -- redis-cli ping
```

**Resolution**:
```bash
# If database connection pool exhausted
# Increase pool size in ConfigMap
kubectl edit configmap dantegpu-config -n dantegpu-production

# If Redis down
kubectl rollout restart deployment/redis -n dantegpu-production

# If code bug
# Deploy hotfix following deployment procedure
```

### Database Connection Issues

**Symptoms**: "connection refused", "too many connections"

**Investigation**:
```bash
# 1. Check PostgreSQL pod
kubectl get pods -n dantegpu-production -l app=postgres

# 2. Check active connections
kubectl exec -it deployment/postgres -n dantegpu-production -- \
  psql -U dante_user -d postgres -c \
  "SELECT count(*), state FROM pg_stat_activity GROUP BY state;"

# 3. Check connection limits
kubectl exec -it deployment/postgres -n dantegpu-production -- \
  psql -U dante_user -d postgres -c "SHOW max_connections;"
```

**Resolution**:
```bash
# Kill idle connections
kubectl exec -it deployment/postgres -n dantegpu-production -- \
  psql -U dante_user -d postgres -c \
  "SELECT pg_terminate_backend(pid) FROM pg_stat_activity 
   WHERE state = 'idle' AND state_change < NOW() - INTERVAL '10 minutes';"

# Increase max_connections (requires restart)
kubectl edit statefulset postgres -n dantegpu-production
# Add: -c max_connections=200

# Deploy pgBouncer for connection pooling
kubectl apply -f k8s/production/pgbouncer-deployment.yaml
```

### Blockchain Transaction Failures

**Symptoms**: Failed rentals, escrow errors, transaction timeouts

**Investigation**:
```bash
# 1. Check billing service logs
kubectl logs -f deployment/billing-service -n dantegpu-production | grep "solana"

# 2. Check Solana RPC status
curl https://api.mainnet-beta.solana.com -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}'

# 3. Check wallet balance
# Use Solana Explorer: https://explorer.solana.com/address/<platform-wallet>

# 4. Check recent transactions
kubectl exec -it deployment/billing-service -n dantegpu-production -- \
  /app/billing-service check-transactions --last=10
```

**Resolution**:
```bash
# If RPC rate limited
# Switch to backup RPC in ConfigMap
kubectl edit configmap dantegpu-config -n dantegpu-production
# Update SOLANA_RPC_URL

# If insufficient balance
# Fund platform wallet from cold storage

# If transaction stuck
# Check on Solana Explorer and potentially resubmit
```

---

## Deployment Procedures

### Staging Deployment

```bash
# 1. Ensure you're on the right context
kubectl config use-context staging

# 2. Run deployment script
./scripts/deploy-staging.sh

# 3. Verify deployment
kubectl get pods -n dantegpu-staging
kubectl get svc -n dantegpu-staging

# 4. Run smoke tests
./scripts/smoke-tests.sh https://staging.dantegpu.com

# 5. Check logs for errors
kubectl logs -f deployment/api-gateway -n dantegpu-staging --tail=50
```

### Production Deployment (Blue-Green)

```bash
# 1. Ensure tests pass
./scripts/run-all-tests.sh

# 2. Build and push images
docker build -t ghcr.io/dante-gpu/dantegpu-core/api-gateway:v1.2.0 ./api-gateway
docker push ghcr.io/dante-gpu/dantegpu-core/api-gateway:v1.2.0

# 3. Deploy to green environment
kubectl apply -f k8s/production/green/

# 4. Wait for green to be ready
kubectl wait --for=condition=available --timeout=600s \
  deployment/api-gateway-green -n dantegpu-production

# 5. Run smoke tests on green
./scripts/smoke-tests.sh https://green.dantegpu.com

# 6. Switch traffic to green
kubectl patch service api-gateway -n dantegpu-production \
  -p '{"spec":{"selector":{"version":"green"}}}'

# 7. Monitor for 10 minutes
watch -n 10 'kubectl get pods -n dantegpu-production'

# 8. If successful, delete blue
kubectl delete -f k8s/production/blue/

# 9. If issues, rollback
kubectl patch service api-gateway -n dantegpu-production \
  -p '{"spec":{"selector":{"version":"blue"}}}'
```

### Database Migration

```bash
# 1. Backup database
./scripts/backup-database.sh

# 2. Test migration on staging
kubectl exec -it deployment/postgres -n dantegpu-staging -- \
  psql -U dante_user -d dante_core -f /migrations/001_new_migration.sql

# 3. Verify migration
kubectl exec -it deployment/postgres -n dantegpu-staging -- \
  psql -U dante_user -d dante_core -c "\dt"

# 4. Run on production (during maintenance window)
kubectl exec -it deployment/postgres -n dantegpu-production -- \
  psql -U dante_user -d dante_core -f /migrations/001_new_migration.sql

# 5. Verify
kubectl exec -it deployment/postgres -n dantegpu-production -- \
  psql -U dante_user -d dante_core -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"
```

---

## Monitoring and Alerts

### Accessing Monitoring Tools

- **Prometheus**: https://prometheus.dantegpu.com
- **Grafana**: https://grafana.dantegpu.com
- **AlertManager**: https://alertmanager.dantegpu.com

### Key Metrics to Monitor

1. **Service Health**
   - Request rate (requests/sec)
   - Error rate (%)
   - Response time (p50, p95, p99)
   - Availability (uptime %)

2. **Resource Usage**
   - CPU usage (%)
   - Memory usage (%)
   - Disk usage (%)
   - Network I/O

3. **Database**
   - Active connections
   - Query duration
   - Cache hit rate
   - Replication lag

4. **Blockchain**
   - Transaction success rate
   - Transaction duration
   - Wallet balance
   - Gas fees

### Alert Thresholds

| Alert | Threshold | Severity | Action |
|-------|-----------|----------|--------|
| Service Down | up == 0 for 5min | Critical | Immediate investigation |
| High Error Rate | >5% for 5min | Warning | Check logs |
| High CPU | >80% for 5min | Warning | Scale up |
| High Memory | >90% for 5min | Warning | Scale up or investigate leak |
| Database Connections | >80% of max | Warning | Investigate connection pool |
| Disk Space | >85% used | Warning | Clean up or expand |
| Replication Lag | >60s | Critical | Check database health |

---

## Database Operations

### Backup and Restore

```bash
# Manual backup
./scripts/backup-database.sh

# Restore from backup
kubectl exec -i deployment/postgres -n dantegpu-production -- \
  psql -U dante_user -d dante_core < backup_2025-10-06.sql

# List backups
ls -lh /backups/postgres/
```

### Performance Tuning

```bash
# Check slow queries
kubectl exec -it deployment/postgres -n dantegpu-production -- \
  psql -U dante_user -d dante_core -c \
  "SELECT query, mean_exec_time, calls FROM pg_stat_statements 
   ORDER BY mean_exec_time DESC LIMIT 10;"

# Analyze table
kubectl exec -it deployment/postgres -n dantegpu-production -- \
  psql -U dante_user -d dante_core -c "ANALYZE jobs;"

# Vacuum table
kubectl exec -it deployment/postgres -n dantegpu-production -- \
  psql -U dante_user -d dante_core -c "VACUUM ANALYZE jobs;"

# Reindex
kubectl exec -it deployment/postgres -n dantegpu-production -- \
  psql -U dante_user -d dante_core -c "REINDEX TABLE jobs;"
```

---

## Troubleshooting

### Pod Won't Start

```bash
# Check events
kubectl describe pod <pod-name> -n dantegpu-production

# Common issues:
# - ImagePullBackOff: Check image name and registry
# - CrashLoopBackOff: Check logs with --previous
# - Pending: Check resource requests and node capacity
```

### High Latency

```bash
# Check database query performance
# Check Redis hit rate
# Check network latency between services
# Check if HPA is scaling properly
```

### Memory Leak

```bash
# Monitor memory over time in Grafana
# Check for goroutine leaks (Go services)
# Profile application
# Increase memory limits temporarily
# Deploy fix
```

---

For architecture details, see [ARCHITECTURE.md](./ARCHITECTURE.md)

