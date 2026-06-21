# contracts

On-chain settlement for DanteGPU. This directory holds the integration with the
settlement layer and any future Solana programs.

## Decision: Covenant is the settlement layer

DanteGPU does not build its own on-chain escrow. It settles GPU rentals through
**Covenant** (the optimistic settlement protocol for AI-agent work: per-job USDC
PDA escrow, challenge window, bonded arbitration), which is already live on Solana.

The split:

- **DanteGPU** owns the marketplace, GPU connection, job execution, and the
  metering of real usage. Its provider daemon measures what was actually delivered.
- **Covenant** owns the money: a rental becomes a Covenant job. The renter locks
  USDC in a per-job escrow, the provider serves the GPU time, DanteGPU's meter
  reports usage, and the escrow releases to the provider on completion (or resolves
  through Covenant's arbitrator on dispute).

This is the metered-work plus conditional-settlement pairing: DanteGPU is the meter,
Covenant is the escrow that opens when the meter confirms delivery.

## Currency

Billing, pricing, escrow, and payout are denominated in **USDC**. The dGPU token is
reserved as an optional incentive, discount, and governance layer, not the unit of
account.

## Status

Integration design is written: see [COVENANT_INTEGRATION.md](COVENANT_INTEGRATION.md)
for the rental-to-Covenant-job mapping, the verified Covenant settlement
semantics, and the phased plan (M3.1 USDC denomination through M3.6 metered
settlement). Until it lands, `billing-payment-service` runs a custodial DB-ledger
model on devnet. The work here:

- Map a DanteGPU rental session to a Covenant job (createJob / submitWork /
  finalizePayment / raiseDispute).
- Feed the provider daemon's signed usage report into Covenant as the settlement
  signal.
- Replace the DB-only fund lock and the unexecuted payout path with Covenant escrow
  and release.
