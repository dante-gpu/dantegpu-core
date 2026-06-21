# Covenant settlement integration (M3 design)

How a DanteGPU GPU rental settles on-chain through Covenant. This is the M3
keystone: it defines the mapping, the call ownership, and the phased plan. It is
grounded in Covenant's actual on-chain program, not assumptions.

## Roles

- **DanteGPU** owns the marketplace, GPU connection, job execution, and the
  metering of real usage (the M2 loop produces a signed usage record).
- **Covenant** owns the money: per-job USDC PDA escrow, an optimistic challenge
  window, and a bonded 2-of-3 arbitrator.

A rental becomes a Covenant job:

| DanteGPU | Covenant |
|---|---|
| Renter | `poster` (funds and locks USDC) |
| Provider | `taker` (accepts, delivers, gets paid) |
| Rental spec (gpu model, vram, rate, duration, rental_id) | off-chain spec JSON, `spec_hash` on-chain |
| Signed metering receipt (GPU-seconds, per-minute samples) | the delivered work: `work_hash` + `delivery_uri` |
| Settlement currency: **USDC** | `token_mint` = USDC |

## Covenant settlement semantics (verified from `programs/covenant`)

This is the load-bearing constraint, taken from the program:

- `create_job`: locks a fixed `amount` of USDC in a per-job PDA escrow
  (`JobEscrow`, seeds `[b"job", poster, spec_hash]`). The amount is set at
  creation and cannot grow.
- `submit_work`: taker posts `work_hash` + `delivery_uri`, starts the challenge
  window (`challenge_end`).
- `finalize_payment` (happy path, after the challenge window, no dispute):
  transfers the **full** `amount` to the taker and closes the escrow. There is
  no partial release on the happy path.
- `resolve_dispute` (2-of-3 arbitrator) supports three outcomes:
  `FavorTaker` (full to taker), `FavorPoster` (full refund to poster), and
  `Split { taker_amount }` (taker gets `taker_amount`, poster refunded the
  rest). Partial settlement exists **only** through dispute resolution.
- `cancel_job`: refunds the escrow to the poster (before acceptance, or under the
  accepted-but-undelivered condition).

## The metered-billing problem

A GPU rental's real cost is known only at the **end** (metered GPU-seconds), but
Covenant locks `amount` at the **start** and the happy path releases the full
`amount`. So escrow-the-max-then-refund-the-unused is not expressible on
Covenant's happy path today. Two honest ways forward:

### Phase 1 (ships on Covenant as-is): fixed-duration prepaid rentals

The renter books a fixed block (for example, 2 hours of an RTX 4090 at a fixed
price). Escrow `amount` = block price = final settled price. Covenant's
full-release finalize fits exactly:

1. Renter starts a rental for a fixed duration and price `P`.
2. DanteGPU creates a Covenant job: `poster` = renter, `amount` = `P`,
   `token_mint` = USDC, spec = rental spec, `challenge_period` = short (for
   example 1 hour). Renter's USDC is locked.
3. Provider accepts (`accept_job`) and the M2 loop runs and meters the rental.
4. When the block ends, the provider `submit_work` with `work_hash` = SHA-256 of
   the signed metering receipt and `delivery_uri` = the receipt stored via
   `storage-service` (MinIO) or Vercel Blob.
5. After the challenge window with no dispute, `finalize_payment` releases `P` to
   the provider.
6. If the provider under-delivered (the metering receipt shows the GPU was idle
   or the SLA was missed), the renter calls `raise_dispute`; the arbitrator uses
   the receipt as evidence and resolves `FavorPoster` (refund) or
   `Split { taker_amount }` (pay for what was actually delivered).

Metering here is the **SLA proof**, not the price. This is shippable now and is
the recommended first integration.

### Phase 2 (needs a small Covenant addition): true metered billing

To bill per actual GPU-second and auto-refund the unused escrow without a
dispute, Covenant needs one new instruction:

> **`settle_metered(taker_amount)`**: like `finalize_payment`, but releases
> `taker_amount <= amount` to the taker and refunds `amount - taker_amount` to
> the poster. Authorized by a metering oracle the poster trusts (the DanteGPU
> metering signer) or co-signed by poster + taker. The Split math already exists
> in `resolve_dispute`; this exposes it on the happy path.

With `settle_metered`, the rental escrows a max upfront, meters real usage, and
settles the exact metered cost at the end with automatic refund. This is the
clean end state and a well-scoped, low-risk Covenant change (cross-repo work,
tracked against the Covenant repo).

## Who calls what

- **DanteGPU `billing-payment-service`** becomes the Covenant client. It replaces
  the current DB-only `LockFunds`/`UnlockFunds` escrow with Covenant job calls.
  It uses the Covenant HTTP API (`covenant.run/api/*`) or `covenant-sdk` via a
  thin Go client (Covenant settles on Solana; the calls are create/accept/
  submit/finalize/dispute).
- **Renter** is the `poster`: in Phase 1 the renter pre-deposits USDC and
  DanteGPU funds the escrow on the renter's behalf (custodial), or the renter's
  wallet signs `create_job` directly (non-custodial, via the frontend).
- **Provider daemon** drives `accept_job` and `submit_work` (it already produces
  the metering receipt in the M2 loop). The receipt is uploaded to
  `storage-service`; its hash is `work_hash`.
- **Settlement crank**: a worker (in `billing-payment-service`) calls
  `finalize_payment` after the challenge window for delivered, undisputed jobs.
  This is the provider payout, executed on-chain, replacing the current
  "payout recorded in DB but not paid".

## What changes in DanteGPU

- `billing-payment-service`:
  - denominate pricing, balances, escrow, and payout in **USDC** (drop dGPU as
    the unit of account; keep it as an optional incentive layer).
  - replace `internal/solana` direct-transfer escrow and the DB fund-lock with a
    Covenant client (`internal/covenant`).
  - `StartRentalSession` -> create (or prepare) a Covenant job.
  - `EndRentalSession` -> provider `submit_work` with the receipt; schedule
    `finalize_payment` after the challenge window.
  - add the settlement crank (finalize delivered, undisputed jobs).
- `provider-daemon`: emit the signed metering receipt at session end, upload it,
  and call `submit_work` (or hand the receipt to `billing-payment-service` to do
  so).
- `gpu-rental-frontend`: Phase 1 fixed-duration rental UI; USDC deposit / wallet
  connect; rental SLA + dispute view.
- `storage-service`: store metering receipts (the `delivery_uri` target).

## Phased plan

- **M3.1** USDC denomination across `billing-payment-service` (config, pricing,
  models). No Covenant dependency; ships independently.
- **M3.2** `internal/covenant` Go client in `billing-payment-service` (create /
  accept / submit / finalize / dispute against Covenant API).
- **M3.3** Map `StartRentalSession` / `EndRentalSession` to the Covenant job
  lifecycle (Phase 1 fixed-duration). Metering receipt -> `work_hash` +
  `delivery_uri`.
- **M3.4** Settlement crank: `finalize_payment` after the challenge window
  (provider payout on-chain). Closes the M2 payout gap.
- **M3.5** Dispute path: surface `raise_dispute` with the metering receipt as
  evidence; arbitrator resolution.
- **M3.6** (cross-repo, Covenant) add `settle_metered` to enable true metered
  billing with auto-refund; switch DanteGPU to escrow-max-then-settle-actual.

## Open questions

- Custodial vs non-custodial poster in Phase 1 (DanteGPU funds escrow vs renter
  wallet signs). Non-custodial is cleaner but needs frontend wallet signing on
  `create_job`.
- Challenge period for compute: 24h is long for a finished GPU job. A short
  window (minutes to 1 hour) with the metering receipt as immediate evidence is
  likely right; tune against Covenant's `min_challenge_period`.
- Whether `submit_work` is driven by the provider daemon directly or proxied
  through `billing-payment-service` (which holds the receipt and the Covenant
  client).
