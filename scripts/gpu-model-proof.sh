#!/usr/bin/env bash
#
# gpu-model-proof.sh — prove that a RENTED GPU runs a REAL model.
#
# Dispatches an AI-inference task through the DanteGPU flow (NATS JetStream ->
# provider-daemon) whose payload loads a real quantized LLM and generates text
# on the local GPU. The daemon's captured stdout is the model's answer, the
# device line proves it ran on the accelerator (Apple Metal / "Device(gpu, 0)").
#
# Defaults to MLX (Apple Silicon). Requires: go, nats-server, python3.
# It creates a throwaway venv and installs mlx-lm on first run, and downloads a
# ~0.7 GB model the first time (cached afterwards). A JetStream nats-server is
# started and torn down; nothing you already run is touched.
#
# Usage: ./scripts/gpu-model-proof.sh [--model HF_MLX_REPO] [--prompt "..."]

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

MODEL="mlx-community/Llama-3.2-1B-Instruct-4bit"
PROMPT="In one sentence, what is a GPU and why is it good for AI?"
while [ $# -gt 0 ]; do
  case "$1" in
    --model) MODEL="$2"; shift 2 ;;
    --prompt) PROMPT="$2"; shift 2 ;;
    *) echo "unknown arg: $1"; exit 2 ;;
  esac
done

PORT_NATS=4222
VENV="/tmp/dante-mlx-venv"
TMP="$(mktemp -d /tmp/dante-gpuproof.XXXXXX)"
DAEMON_BIN="$TMP/dante-daemon"
NATS_PID=""; DAEMON_PID=""

cleanup() {
  [ -n "$DAEMON_PID" ] && kill "$DAEMON_PID" 2>/dev/null || true
  pkill -f "$DAEMON_BIN" 2>/dev/null || true
  [ -n "$NATS_PID" ] && kill "$NATS_PID" 2>/dev/null || true
  rm -rf "$TMP" 2>/dev/null || true
}
trap cleanup EXIT
export GOFLAGS=-mod=mod

echo "== ensuring mlx-lm is available =="
if ! "$VENV/bin/python" -c "import mlx_lm" 2>/dev/null; then
  python3 -m venv "$VENV"
  "$VENV/bin/pip" install --quiet --upgrade pip
  "$VENV/bin/pip" install --quiet mlx-lm
fi
"$VENV/bin/python" -c "import mlx.core as mx; print('   accelerator:', mx.default_device())"

echo "== building daemon + starting NATS JetStream =="
go build -C clients/provider-daemon -o "$DAEMON_BIN" ./cmd/daemon
go build -C cmd/setup-nats -o "$TMP/setup-nats" .
mkdir -p "$TMP/nats"
nats-server -js -sd "$TMP/nats" -p $PORT_NATS >"$TMP/nats.log" 2>&1 &
NATS_PID=$!
sleep 1.5
"$TMP/setup-nats" >/dev/null
"$DAEMON_BIN" --config clients/provider-daemon/configs/config.yaml >"$TMP/daemon.log" 2>&1 &
DAEMON_PID=$!
sleep 3
grep -q "Successfully subscribed to JetStream" "$TMP/daemon.log" \
  && echo "   daemon subscribed to tasks.dispatch.provider-daemon-01.*"

echo "== dispatching an AI-inference task over NATS (runs $MODEL on the GPU) =="
JOB="llm-$(date +%s)"
SCRIPT="PYTHONWARNINGS=ignore HF_HUB_DISABLE_PROGRESS_BARS=1 $VENV/bin/python $ROOT/scripts/gpu_infer.py '$MODEL' '$PROMPT' 2>/dev/null"
mkdir -p "$TMP/pub"
cat > "$TMP/pub/go.mod" <<EOF
module pub
go 1.24
require github.com/nats-io/nats.go v1.39.1
EOF
cat > "$TMP/pub/main.go" <<'EOF'
package main
import ("encoding/json";"fmt";"os";"time";"github.com/nats-io/nats.go")
func main(){
  nc,_:=nats.Connect("nats://localhost:4222"); defer nc.Close()
  js,_:=nc.JetStream()
  job,script:=os.Args[1],os.Args[2]
  t:=map[string]any{"job_id":job,"user_id":"proof","job_type":"ai-inference",
    "job_name":"real-llm-on-gpu",
    "job_params":map[string]any{"script_content":script,"script_interpreter":"/bin/sh"},
    "execution_type":"script","assigned_provider_id":"provider-daemon-01",
    "dispatched_at":time.Now().UTC().Format(time.RFC3339)}
  d,_:=json.Marshal(t)
  ack,err:=js.Publish(fmt.Sprintf("tasks.dispatch.provider-daemon-01.%s",job),d)
  if err!=nil{fmt.Println("publish:",err);os.Exit(1)}
  fmt.Printf("   dispatched %s (TASKS stream seq %d)\n",job,ack.Sequence)
}
EOF
(cd "$TMP/pub" && go mod tidy >/dev/null 2>&1 && go run . "$JOB" "$SCRIPT")

echo "== waiting for the GPU to run the model =="
for _ in $(seq 1 40); do
  grep -q "Task execution finished" "$TMP/daemon.log" && break
  sleep 1
done

echo
echo "== the rented GPU's answer (daemon-captured stdout) =="
"$VENV/bin/python" - "$TMP/daemon.log" <<'PY'
import json, re, sys
log = open(sys.argv[1]).read()
m = re.search(r'"stdout": "((?:[^"\\]|\\.)*)"', log)
if not m:
    print("no model output captured"); sys.exit(1)
out = json.loads('"' + m.group(1) + '"')
r = json.loads(out)
print(f"   device          {r['device']}")
print(f"   model           {r['model']}")
print(f"   prompt          {r['prompt']}")
print(f"   tokens/sec      {r['tokens_per_sec']}  ({r['tokens']} tokens, {r['gpu_peak_mem_gb']} GB GPU)")
print()
print("   >>", r['output'])
PY
echo
echo "== proof complete: a real model ran on the rented GPU, dispatched via DanteGPU =="
