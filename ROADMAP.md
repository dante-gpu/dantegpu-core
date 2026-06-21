# DanteGPU Core Roadmap

Where the project is, and the path to a production marketplace. Phases are ordered
by dependency: each one unblocks the next. Every phase lists concrete tasks and a
pass/fail bar so "done" is not a matter of opinion.

The guiding rule, taken from the rebuild of this repo: never mark a thing done
until the code does it. Status here is measured against running behavior, not
intent.

## Where we are

The backbone is real. A provider can register, its NVIDIA GPUs are detected, a
renter can browse the catalog, reserve a GPU, and have a job dispatched over NATS
and executed in Docker on the provider node. Billing math is correct and runs on
real Solana devnet wallets. The schema (68 tables, 13 migrations) is
production-grade.

What is not real yet: the gateway still authenticates against an in-memory mock
user map, metering data never reaches billing, and the provider-side on-chain
payout is recorded but not executed. Those are the first things the roadmap closes.

See [docs/STATUS.md](docs/STATUS.md) for the per-service starting point.

## Phase 0: Backbone (done)

The list, discover, rent, and run path, on real infrastructure.

- Provider and GPU-capability registry with Consul service discovery
- GPU catalog read API and renter frontend
- Rental reservations with PostgreSQL fund-locking and balance checks
- dGPU pricing engine (base + VRAM + power + platform fee) with decimal precision
- Solana wallet creation, balance, deposit, and withdrawal via real RPC
- NATS JetStream job dispatch and provider-daemon Docker/script execution
- MinIO object storage, Redis cache layer, Prometheus/Grafana/Loki stack

Pass: `docker compose up` brings the stack up; a renter can register, reserve a
listed GPU, and a job runs on a provider node. **Met.**

## Phase 1: Identity and gateway correctness

The gateway is the front door and it currently lies about who you are. Fix that
before anything else.

- Delete the mock user map in `api-gateway/internal/auth/user.go`
- Route all gateway auth through `auth-service` over HTTP
- Replace the gateway's "not yet implemented" billing endpoints with a real
  billing client that calls `billing-payment-service`
- Collapse the dual entrypoints to one canonical build each:
  `auth-service` (keep PostgreSQL `main.go`, drop `main-sqlite.go` or move it to an
  `examples/` build tag) and `api-gateway` (keep `cmd/main.go`, drop
  `main-simple.go`)
- Move the hardcoded JWT secret default to required configuration; fail closed if
  unset
- Read the user id from JWT claims, not the `X-User-ID` header

Pass: there is exactly one identity source; the gateway proxies real auth and real
balances; no hardcoded credentials or secrets remain in the auth path.

## Phase 2: Close the money loop

Make metered usage flow into billing and make the provider actually get paid.

- `gpu-monitoring-service` publishes per-minute usage samples to NATS
- `billing-payment-service` consumes usage and updates the live session cost
- A payout worker executes the on-chain dGPU transfer for each `provider_payouts`
  row and marks it settled (with retry and idempotency)
- Implement on-chain escrow for rentals (lock on start, release or refund on end)
  instead of a database-only lock
- Implement `GetTransactionHistory` (currently a TODO returning empty)

Pass: a full rental from start to finish meters real GPU time, bills the renter for
exactly that time, and lands dGPU in the provider wallet on devnet, verifiable by
transaction signature.

## Phase 3: Dynamic pricing and scheduling maturity

Make pricing respond to the market and scheduling respect real GPU constraints.

- Wire `getDynamicPricingFactors()` to live supply/demand from
  `provider-registry-service` (idle vs busy GPU counts, recent utilization)
- Replace type-and-count job matching with VRAM, power, and capability-aware
  matching in `scheduler-orchestrator-service`
- Implement the Kubernetes scheduler extender (currently an empty stub), OR remove
  the K8s integration path entirely and commit to the NATS dispatch model
- Add provider capacity pre-filtering and basic bin-packing

Pass: identical GPUs priced differently under different load; a job requesting more
VRAM than a GPU has is never scheduled onto it.

## Phase 4: Test coverage, CI, and security hardening

The real services have almost no tests and CI was testing the wrong ones.

- Unit tests for `billing-payment-service` (pricing math, escrow, payout),
  `scheduler-orchestrator-service` (matching, consumer), and
  `provider-registry-service`
- CI matrix points at the canonical services (this rebuild already removed the
  dead `billing-service`, `provider-registry`, and `scheduler` from the matrix)
- An end-to-end test that runs the full six-verb path against the compose stack
- Security pass: no plaintext secrets (the Grafana admin password is currently
  hardcoded in compose), rate limiting on the gateway, input validation audit
- Structured logging and graceful shutdown in the single-file services
  (`gpu-service`, `rental-service`)

Pass: CI builds and tests every canonical service against a real ephemeral stack;
coverage on the billing and scheduling cores is meaningful; no secret ships in the
repo.

## Phase 5: Breadth: hardware, payments UI, clients

Make the claims the old README made actually true, one at a time.

- Real AMD (ROCm), Intel, and Apple Silicon GPU detection in the provider daemon,
  replacing the hardcoded VRAM fallbacks
- Payment and wallet UI in `gpu-rental-frontend` (deposit, balance, rental cost)
- Decide the mobile app: finish the missing five screens or remove it
- Standardize the API base URL across all clients (they currently disagree)

Pass: a provider on a non-NVIDIA card lists accurate specs; a renter can fund a
wallet and watch a rental's cost from the browser.

## Phase 6: Decentralization and production

The original thesis: a permissionless marketplace settling on mainnet.

- One-command permissionless provider onboarding
- Provider attestation and dispute handling (proof that the rented GPU ran the job)
- Reputation scoring backed by real job outcomes
- Mainnet dGPU token, multi-signature platform treasury, audited payment flows
- Load and performance testing under realistic provider churn

Pass: a stranger joins a GPU with one command, serves paid jobs, builds reputation,
and is settled on mainnet, with disputes resolvable.

## Non-goals

- DanteGPU Core is not an inference or training engine. It schedules and meters
  compute; the workload runs in the provider's container. Distributed inference
  across providers is a separate concern and out of scope here.
- Not a custodial exchange. The platform settles rentals in dGPU; it does not hold
  fiat or operate order books.

## How to read status changes

When a phase task lands, update [docs/STATUS.md](docs/STATUS.md) and the six-verb
table in the README in the same change. The README, STATUS, and this roadmap are
meant to never disagree.
