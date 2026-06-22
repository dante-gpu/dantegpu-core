# Verifiable rental proof

This document shows that the DanteGPU rental flow works end to end with **real
code on real infrastructure**, and how to reproduce it yourself in one command.

```bash
./scripts/rental-proof.sh
```

The script spins up a throwaway Postgres cluster and a JetStream-enabled
`nats-server` (it never touches any Postgres/NATS you already run), then tears
everything down on exit. Docker and an NVIDIA GPU are **not** required — script
execution and Apple/AMD/Intel detection work without them.

## What it proves

| Stage | Verb | Real code exercised |
|-------|------|---------------------|
| 1. Detect | LIST | `provider-daemon --get-gpus-json` detects the host's actual GPU |
| 2. Price | DISCOVER/RENT | the billing `pricing.Engine` computes the USDC cost |
| 3. Persist | METER/SETTLE | a metered session is written to and read back from Postgres via the live billing store |
| 4. Dispatch | RUN | a task is published over NATS JetStream; the daemon receives and **executes** it, reporting status back |

## A real run (Apple M4 Max, this author's machine)

```
== [1] real GPU detection ==
[ { "id": "apple-0", "name": "Apple M4 Max", "vram_total_mb": 49152 } ]

== [3] real detection + pricing + LIVE Postgres session persist ==
  Provider GPU           Apple M4 Max  (detected: apple-0)
  VRAM                   49152 MB (48 GB unified)
  Total hourly rate      1.2101 USDC      (0.25 base + 48 GB x 0.02 VRAM + power)
  Duration               2 h
  TOTAL CHARGED          2.5411 USDC
  Provider earnings      2.2991 USDC
  DB session (Postgres)  2c1ad49b-3982-4f80-b989-d2fae604b4c5
  DB status (read back)  completed
  DB total cost          2.541116 USDC
  Receipt SHA-256        8d9f9718...a82c8d359

-- rows actually in Postgres --
  status   |  gpu_model   | allocated_vram_mb | total_cost  | provider_earnings
 completed | Apple M4 Max |             49152 | 2.541115500 |       2.299104500

== [4] real NATS JetStream dispatch -> daemon executes the task ==
daemon subscribed to tasks.dispatch.provider-daemon-01.*
published proofjob-... (stream seq 1)
Task processed and ACKed successfully  {"job_id": "proofjob-..."}
Task execution finished  {"exitCode": 0, "stdout": "hello-from-dantegpu\n"}
```

The receipt's `SHA-256` is computed over the sorted `key=value` lines of
`/tmp/dante-rental-receipt.json` (excluding the hash itself) and can be
recomputed independently:

```bash
python3 - <<'PY'
import json, hashlib
r = json.load(open('/tmp/dante-rental-receipt.json')); h = r.pop('receipt_sha256')
canon = ''.join(f"{k}={r[k]}\n" for k in sorted(r))
print("match" if hashlib.sha256(canon.encode()).hexdigest()==h else "MISMATCH")
PY
```

The pricing is genuinely computed, not a fixed value: cost scales linearly with
duration (2 h -> 2.5411 USDC, 8 h -> 10.1650 USDC).

## Two real bugs this proof surfaced and fixed

Building the proof flushed out two genuine defects:

1. **Apple GPU VRAM reported as 0.** The daemon's Apple detection used a hardcoded
   per-chip table that only knew M1/M2/M3, so any M4 reported `vram_total_mb: 0`.
   Replaced with a real `sysctl hw.memsize` read of the unified memory.
2. **NATS task dispatch was completely broken.** The daemon's default subject
   pattern was `dante.tasks.dispatch.>` — it had the wrong stream prefix and no
   `%s` for the instance id, so `fmt.Sprintf` produced a garbage subject and no
   stream matched. Fixed to `tasks.dispatch.%s.*`, which the `TASKS` JetStream
   stream captures; the daemon now subscribes and executes dispatched tasks.

## Running a real model on the rented GPU

`scripts/gpu-model-proof.sh` goes one step further: it dispatches an
**AI-inference task** through the same NATS flow, and the daemon runs a real
quantized LLM on the host's GPU. The daemon-captured stdout is the model's
answer; the device line proves it ran on the accelerator.

```bash
./scripts/gpu-model-proof.sh
```

A real run on the Apple M4 Max GPU (Meta Llama 3.2 1B, 4-bit, via MLX/Metal):

```
== the rented GPU's answer (daemon-captured stdout) ==
   device          Device(gpu, 0)
   model           mlx-community/Llama-3.2-1B-Instruct-4bit
   prompt          In one sentence, what is a GPU and why is it good for AI?
   tokens/sec      235.71  (58 tokens, 0.81 GB GPU)

   >> A Graphics Processing Unit (GPU) is a specialized electronic circuit
      designed to quickly manipulate and alter spatial data ... making it
      particularly well-suited for ... Artificial Intelligence (AI).
```

So the full chain is real: detect the GPU, rent it, dispatch a job over NATS,
the daemon executes it, and a real Llama model generates text on the Metal GPU,
with the output returned through the daemon. The inference payload is
`scripts/gpu_infer.py` (MLX); swap `--model` for any `mlx-community` repo, or
point it at a CUDA box with a PyTorch payload for NVIDIA.

## What this is NOT

This proves the rental flow end to end **including real model execution on the
rented GPU**. What it does NOT yet show is a *distributed, multi-node* inference
run (one model sharded across many rented GPUs over the network, like
leyten/shard) — that needs several real accelerators and the cross-node
execution layer, which is a later milestone. Here a single host's GPU runs the
whole (small) model.
