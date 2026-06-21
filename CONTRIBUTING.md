# Contributing to DanteGPU Core

Thanks for considering a contribution. This repo values honest, working code over
breadth. A small change that is real and tested beats a large change that is
aspirational.

## Ground rules

1. **Never mark something done until the code does it.** If a path is stubbed,
   label it stubbed in the code, in [docs/STATUS.md](docs/STATUS.md), and in the
   six-verb table in the [README](README.md). The README, STATUS, and ROADMAP must
   never disagree.
2. **No mock data in a path that claims to be real.** If a function cannot do the
   real thing yet, it should fail clearly or be marked partial, not return
   plausible fake values.
3. **Keep money exact.** Use `shopspring/decimal` for all currency math. No floats
   for balances, prices, fees, or payouts.
4. **One canonical implementation per concern.** Do not add a parallel
   `foo-service` next to `foo`. If you are replacing something, remove the old one
   in the same change.

## Project shape

Each service is an independent Go module in its own directory with its own
`go.mod`. Shared types live in `common/`. Services discover each other through
Consul and communicate events over NATS JetStream. The canonical schema is in
`database/migrations/`.

The canonical services are: `api-gateway`, `auth-service`,
`provider-registry-service`, `gpu-service`, `rental-service`,
`billing-payment-service`, `scheduler-orchestrator-service`, `storage-service`,
`gpu-monitoring-service`, `redis-cache-service`. The provider side is
`provider-daemon` and `provider-gui`.

## Local development

```bash
# Bring up infrastructure and services
docker compose up -d

# Apply migrations
./database/run_migrations.sh

# Build a single service (each is its own module)
cd billing-payment-service && go build ./...

# Run that module's tests
go test ./...
```

For the frontend:

```bash
cd gpu-rental-frontend
npm install
npm run dev
```

## Before you open a PR

- `go build ./...` passes in every module you touched
- `go test ./...` passes (add tests for new logic; the billing and scheduling cores
  especially need coverage)
- `gofmt` is clean
- You updated `docs/STATUS.md` and the README six-verb table if you changed what
  works
- No secrets, keys, or passwords are committed

## Commit and PR style

- Conventional commit subjects: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`,
  `chore:`, `perf:`, `ci:`
- Describe what changed and why, and call out anything still stubbed
- Do not use em dashes in code, comments, docs, or commit messages

## Good first issues

The roadmap names the highest-leverage work. The cleanest starting points:

- Replace the api-gateway mock user map with real `auth-service` calls (Phase 1)
- Publish `gpu-monitoring-service` usage to billing over NATS (Phase 2)
- Add unit tests for the `billing-payment-service` pricing engine (Phase 4)
- Real AMD/Apple Silicon detection in `provider-daemon` (Phase 5)

See [ROADMAP.md](ROADMAP.md) for the full plan and pass/fail criteria.
