#!/usr/bin/env python3
"""Run real LLM inference on the local GPU and print a JSON result.

This is the payload a DanteGPU rental task executes to prove a rented GPU runs a
real model. It uses MLX (Apple Silicon / Metal) by default; the device line in
the output shows which accelerator actually ran the model.

Requires: mlx, mlx-lm (pip install mlx-lm, ideally in a venv).

Usage: gpu_infer.py [MODEL] [PROMPT]
"""
import sys, time, json
import mlx.core as mx
from mlx_lm import load, generate

MODEL = sys.argv[1] if len(sys.argv) > 1 else "mlx-community/Llama-3.2-1B-Instruct-4bit"
PROMPT = sys.argv[2] if len(sys.argv) > 2 else "In one sentence, what is a GPU and why is it good for AI?"

t0 = time.time()
model, tokenizer = load(MODEL)
load_s = time.time() - t0

messages = [{"role": "user", "content": PROMPT}]
prompt = tokenizer.apply_chat_template(messages, add_generation_prompt=True)

mx.reset_peak_memory()
t1 = time.time()
text = generate(model, tokenizer, prompt=prompt, max_tokens=80, verbose=False)
gen_s = time.time() - t1

ntok = len(tokenizer.encode(text))
print(json.dumps({
    "device": str(mx.default_device()),
    "model": MODEL,
    "prompt": PROMPT,
    "output": text.strip(),
    "tokens": ntok,
    "tokens_per_sec": round(ntok / gen_s, 2) if gen_s else 0,
    "load_seconds": round(load_s, 2),
    "gen_seconds": round(gen_s, 2),
    "gpu_peak_mem_gb": round(mx.get_peak_memory() / 1e9, 3),
}))
