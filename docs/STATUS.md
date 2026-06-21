# Service status

The honest, per-service state of the repo, derived from a code-level audit (not
from READMEs or comments). Use this with the six-verb table in the
[README](../README.md) and the [ROADMAP](../ROADMAP.md).

Legend: **Working** = substantial real logic on real dependencies. **Partial** =
real skeleton with a named gap. **Stub/Demo** = placeholder or development-only.
**Removed** = deleted in the cleanup (was dead or superseded).

## Backend services

| Service | State | What is real | Key gap |
|---------|-------|--------------|---------|
| `billing-payment-service` | Working | Solana RPC wallets/balance/deposit/withdraw, decimal pricing engine (base + VRAM + power + fee), session billing, PostgreSQL store | Dynamic pricing factors hardcoded; provider payout written to DB but not sent on-chain; `GetTransactionHistory` is a TODO |
| `auth-service` | Working | PostgreSQL + bcrypt + JWT login/register/profile | 2FA/OAuth not implemented; dual `main.go` and `main-sqlite.go`; JWT secret has a hardcoded default; profile reads `X-User-ID` header not claims |
| `provider-registry-service` | Working | Provider + GPU-capability registry, Consul registration, DB connection retry, full CRUD | No real capability benchmarking; verification is a status flip, not a proof |
| `scheduler-orchestrator-service` | Partial | NATS JetStream consumer, job state machine, Consul registration, billing validation on submit | K8s scheduler extender is an empty stub; matching is type+count only (ignores VRAM/power); usage not sent to billing |
| `storage-service` | Working | MinIO object storage, presigned URLs, Consul, middleware stack | Health check is a static `{status:UP}` with no MinIO ping |
| `redis-cache-service` | Working | Cache helpers with TTLs and invalidation for GPU/session/pricing/job state | Library only (its `main` is a demo); no HTTP server, imported by other services |
| `gpu-service` | Working | Read-only GPU catalog with JOINs across instances/providers/models, pagination | Single file, read-only, no auth, no retry |
| `rental-service` | Working | Rental creation with PostgreSQL transaction (availability + balance check, fund lock, GPU status update) | Connection info is mocked; no expiration; not wired to billing settlement; single file |
| `gpu-monitoring-service` | Partial | Real `nvidia-smi` sampling (memory/util/temp/power), WebSocket stream | Falls back to mock GPUs if `nvidia-smi` absent; no persistence; usage never published to billing |
| `api-gateway` | Partial | chi router, JWT middleware, CORS, Consul/NATS health, proxy routing | In-memory mock users (plaintext); billing endpoints return mock data (`GetUserBalance` returns a hardcoded `100.0`); duplicate `main-simple.go` |

## Provider side

| Component | State | What is real | Key gap |
|-----------|-------|--------------|---------|
| `provider-daemon` | Partial | GPU detection for NVIDIA (`nvidia-smi`), AMD (`rocm-smi`), and Apple Silicon (`system_profiler`); Docker + script execution; NATS task consumption; gopsutil system metrics | Billing client is explicitly stubbed (`GetFinancialSummary` returns mock); several VRAM values are hardcoded mappings rather than queried |
| `cmd/provider` | Working | A second, monolithic provider daemon (~2.3k lines): multi-GPU detect, Docker lifecycle, NATS, Solana wallet | Overlaps `provider-daemon`; consolidation is a future decision |
| `provider-gui` | Working | Tauri desktop app (~2.9k lines Rust): spawns the daemon, parses real `system_profiler`, manages lifecycle | Financial summary inherits the daemon stub; wallet id hardcoded |
| `dantegpu-device-plugin` | Working | Kubernetes device plugin gRPC (ListAndWatch, Allocate, GetPreferredAllocation) | GPU enumeration is a static count, not runtime discovery |

## Frontends

| App | State | Notes |
|-----|-------|-------|
| `gpu-rental-frontend` | Partial | Real React/Vite SPA: auth flow, catalog fetch, active rentals. Missing payment UI; API base URL is inconsistent (:8080 vs :8090) |
| `provider-gui` | Working | See provider side above |
| `provider-web-app` | Demo | Marketing landing page with hardcoded terminal logs; no platform features |
| `mobile-app` | Stub | 1 of 6 screens; undefined imports; will not run as-is |
| `user-dashboard` | Stub | API-client library with no UI and no `package.json`; referenced by `k8s/production/ingress.yaml` so kept for now, flagged for completion-or-removal |

## Infrastructure and data

| Item | State | Notes |
|------|-------|-------|
| `database/migrations` | Working | 13 migrations, ~68 tables, partitioning, stored procedures, views. The strongest artifact in the repo |
| `monitoring-logging-service` | Demo | The Go service is empty; the value is the docker-compose Prometheus/Grafana/Loki/Alertmanager stack (Grafana password is hardcoded, fix in Phase 4) |
| `infrastructure/`, `k8s/` | Config | Redis/MinIO/NATS/Consul/Prometheus config and K8s manifests; not all audited |
| `test-mock-services` | Demo | In-memory mocks for local dev (hardcoded JWT); not production |
| `ory-kratos-integration` | Partial | A working Kratos client, but not integrated with `auth-service`. Experimental alternative identity path; `auth-service` is canonical |
| `src/` (Rust) | Stub | Single-file fallback daemon controller; orphaned |

## Removed in the cleanup

These were dead (no `main.go`, nothing imported them, not in compose or k8s) or
fake and unwired. Deleting them is what let the CI matrix be pointed at the real
services. They remain in git history.

| Removed | Why | Superseded by |
|---------|-----|---------------|
| `billing-service` | No entrypoint, orphaned handlers | `billing-payment-service` |
| `provider-registry` | No entrypoint; unsafe SQL arg building | `provider-registry-service` |
| `scheduler` | No entrypoint, handler-only | `scheduler-orchestrator-service` |
| `stripe-paypal-integration` | PayPal 100% faked, wired to nothing; billing is Solana-only | (none) |

Eleven aspirational progress reports were moved from the repo root to
[archive/](archive/).
