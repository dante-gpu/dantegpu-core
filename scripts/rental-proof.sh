#!/usr/bin/env bash
#
# rental-proof.sh — end-to-end, verifiable proof that the DanteGPU rental flow
# works with real code on real infrastructure. Nothing here is mocked:
#
#   1. DETECT  — the provider-daemon detects this host's actual GPU.
#   2. PRICE   — the billing pricing engine computes the real USDC cost.
#   3. PERSIST — a real metered session is written to (and read back from)
#                Postgres through the live billing store.
#   4. DISPATCH— a task is published over NATS JetStream; the daemon receives
#                and executes it, reporting status back over NATS.
#
# It spins up a throwaway Postgres cluster and a JetStream-enabled nats-server,
# so it never touches any Postgres/NATS you already run. Everything is torn down
# on exit.
#
# Requirements: go, nats-server, initdb/pg_ctl/psql (PostgreSQL client+server),
# python3. Docker and an NVIDIA GPU are NOT required — script execution and
# Apple/AMD/Intel detection work without them.
#
# Usage:  ./scripts/rental-proof.sh [--hours N] [--no-live]
#   --no-live   skip the Postgres + NATS stages (detection + pricing only)

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

HOURS=2
LIVE=1
while [ $# -gt 0 ]; do
  case "$1" in
    --hours) HOURS="$2"; shift 2 ;;
    --no-live) LIVE=0; shift ;;
    *) echo "unknown arg: $1"; exit 2 ;;
  esac
done

PORT_PG=5599
PORT_NATS=4222
TMP="$(mktemp -d /tmp/dante-proof.XXXXXX)"
PGDATA="$TMP/pg"
DAEMON_BIN="$TMP/dante-daemon"
PROOF_BIN="$TMP/rental-proof"
NATS_PID=""
DAEMON_PID=""

cleanup() {
  [ -n "$DAEMON_PID" ] && kill "$DAEMON_PID" 2>/dev/null || true
  pkill -f "$DAEMON_BIN" 2>/dev/null || true
  [ -n "$NATS_PID" ] && kill "$NATS_PID" 2>/dev/null || true
  pg_ctl -D "$PGDATA" -w stop >/dev/null 2>&1 || true
  rm -rf "$TMP" 2>/dev/null || true
}
trap cleanup EXIT

export GOFLAGS=-mod=mod

echo "== building daemon + rental-proof =="
go build -C clients/provider-daemon -o "$DAEMON_BIN" ./cmd/daemon
go build -C services/billing-payment-service -o "$PROOF_BIN" ./cmd/rental-proof

echo
echo "== [1] real GPU detection =="
"$DAEMON_BIN" --get-gpus-json | python3 -m json.tool

if [ "$LIVE" -eq 0 ]; then
  echo
  echo "== [2] pricing + receipt (detection + engine only) =="
  "$PROOF_BIN" --daemon "$DAEMON_BIN" --hours "$HOURS"
  exit 0
fi

echo
echo "== [2] throwaway Postgres + dante databases (fixed init script) =="
initdb -D "$PGDATA" -U dante_user --auth=trust >/dev/null 2>&1
pg_ctl -D "$PGDATA" -o "-p $PORT_PG -k /tmp -c listen_addresses=''" -w start >/dev/null 2>&1
psql -h /tmp -p $PORT_PG -U dante_user -d postgres -qc "CREATE DATABASE dante_auth OWNER dante_user;"
psql -h /tmp -p $PORT_PG -U dante_user -d dante_auth -q -v ON_ERROR_STOP=1 \
  -f scripts/db_setup/00_create_user_and_databases.sql >/dev/null
echo "databases: $(psql -h /tmp -p $PORT_PG -U dante_user -d postgres -tAc \
  "SELECT string_agg(datname,', ') FROM pg_database WHERE datname LIKE 'dante%'")"

echo
echo "== [3] real detection + pricing + LIVE Postgres session persist =="
DSN="postgres://dante_user@/dante_billing?host=/tmp&port=$PORT_PG&sslmode=disable"
"$PROOF_BIN" --daemon "$DAEMON_BIN" --hours "$HOURS" --dsn "$DSN" --meter-seconds 2

echo "-- rows actually in Postgres --"
psql -h /tmp -p $PORT_PG -U dante_user -d dante_billing -c \
  "SELECT status, gpu_model, allocated_vram_mb, total_cost, provider_earnings FROM rental_sessions WHERE status='completed';"

echo
echo "== [4] real NATS JetStream dispatch → daemon executes the task =="
rm -rf "$TMP/nats"; mkdir -p "$TMP/nats"
nats-server -js -sd "$TMP/nats" -p $PORT_NATS >"$TMP/nats.log" 2>&1 &
NATS_PID=$!
sleep 1.5
go build -C cmd/setup-nats -o "$TMP/setup-nats" .
"$TMP/setup-nats" >/dev/null
"$DAEMON_BIN" --config clients/provider-daemon/configs/config.yaml >"$TMP/daemon.log" 2>&1 &
DAEMON_PID=$!
sleep 3
grep -q "Successfully subscribed to JetStream" "$TMP/daemon.log" \
  && echo "daemon subscribed to tasks.dispatch.provider-daemon-01.*"

JOB="proofjob-$(date +%s)"
mkdir -p "$TMP/pub"
cat > "$TMP/pub/go.mod" <<EOF
module pub
go 1.24
require github.com/nats-io/nats.go v1.39.1
EOF
cat > "$TMP/pub/main.go" <<EOF
package main
import ("encoding/json";"fmt";"time";"github.com/nats-io/nats.go";"os")
func main(){
  nc,_:=nats.Connect("nats://localhost:$PORT_NATS"); defer nc.Close()
  js,_:=nc.JetStream()
  job:=os.Args[1]
  t:=map[string]any{"job_id":job,"user_id":"proof","job_type":"script_execution",
    "job_name":"rental-proof-task","job_params":map[string]any{"script_content":"echo hello-from-dantegpu"},
    "execution_type":"script","assigned_provider_id":"provider-daemon-01",
    "dispatched_at":time.Now().UTC().Format(time.RFC3339)}
  d,_:=json.Marshal(t)
  ack,err:=js.Publish(fmt.Sprintf("tasks.dispatch.provider-daemon-01.%s",job),d)
  if err!=nil{fmt.Println("publish:",err);return}
  fmt.Printf("published %s (stream seq %d)\n",job,ack.Sequence)
}
EOF
(cd "$TMP/pub" && go mod tidy >/dev/null 2>&1 && go run . "$JOB")
sleep 2
echo "-- daemon execution result --"
grep -iE "Task processed and ACKed|Task execution finished" "$TMP/daemon.log" | tail -2

echo
echo "== proof complete — receipt JSON at /tmp/dante-rental-receipt.json =="
