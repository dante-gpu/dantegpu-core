# DanteGPU Console

The renter-facing web console for DanteGPU: browse the GPU marketplace, rent any
GPU and pay per second in USDC, watch live cost + logs, deposit/withdraw, and
onboard as a provider. Settlement is secured on Solana.

## Stack

- **Vite 6** + **React 18** + **TypeScript** (strict)
- **Tailwind CSS v4** (token-driven design system in `src/index.css`)
- **TanStack Query v5** for server state
- **Solana wallet-adapter** (Phantom, Solflare) + **@solana/spl-token** for USDC
- **Recharts** for the live cost meter, **lucide-react** for icons

## Design language

A dark "inferno" console: near-black ink base, ember-orange primary, electric
cyan secondary. Funnel Display for headings, Inter for body, JetBrains Mono for
numerics. All colors flow through the Tailwind v4 `@theme` block so screens stay
consistent.

## Getting started

```bash
npm install
cp .env.example .env   # set VITE_API_BASE_URL, USDC mint, deposit address
npm run dev            # http://localhost:5273
```

In dev, `/api` is proxied to the gateway (`VITE_API_BASE_URL`, default
`http://localhost:8080`) so the browser never hits CORS.

## Environment

| Variable | Purpose |
| --- | --- |
| `VITE_API_BASE_URL` | api-gateway origin |
| `VITE_SOLANA_CLUSTER` | `devnet` / `mainnet-beta` |
| `VITE_SOLANA_RPC_URL` | explicit RPC (overrides cluster) |
| `VITE_USDC_MINT` | SPL mint billed in |
| `VITE_USDC_DECIMALS` | mint decimals (6 for USDC) |
| `VITE_PLATFORM_DEPOSIT_ADDRESS` | treasury wallet that receives deposits |

## Structure

```
src/
  lib/         api client, types, formatting, solana + USDC helpers
  providers/   QueryClient, Solana wallet, Auth session
  hooks/       marketplace, balance, on-chain USDC, job polling, deposit
  components/  ui/ primitives + domain components (GpuCard, CostMeter, LogStream)
  pages/       Login, Register, Dashboard, Marketplace, RentalSession,
               MyRentals, Wallet, ProviderOnboarding, Settings
```

## Build

```bash
npm run build      # tsc -b && vite build
npm run preview
```

## How it talks to the backend

All calls go through the api-gateway under `/api/v1`:

- **Auth** — `/auth/login`, `/auth/register`, `/auth/me`
- **Marketplace** — `/billing/marketplace`
- **Wallet** — `/billing/user/:id/balance`, `/billing/wallet/:id/deposit|withdraw|transactions`
- **Pricing** — `/billing/pricing/rates|estimate`
- **Jobs / rentals** — `POST /jobs`, `GET /jobs/:id`, `DELETE /jobs/:id`

Deposits are real on-chain SPL USDC transfers signed by the connected wallet,
then reported to the billing service for crediting.
