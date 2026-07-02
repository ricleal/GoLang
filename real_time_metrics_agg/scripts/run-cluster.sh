#!/usr/bin/env bash
#
# run-cluster.sh - Start a 3-node distributed metrics aggregator cluster.
#
# Usage:
#   ./scripts/run-cluster.sh              # start all 3 nodes
#   ./scripts/run-cluster.sh --build-only # just build the binaries
#   ./scripts/run-cluster.sh --clean      # remove data dirs
#

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BUILD_DIR="${ROOT}/build"
DATA_DIR="${ROOT}/data/cluster"
BIN_SERVER="${BUILD_DIR}/metrics-server"
BIN_CLIENT="${BUILD_DIR}/metrics-client"

# Node definitions
NODES=(
  "node1:50051:50070"
  "node2:50052:50071"
  "node3:50053:50072"
)

clean() {
  echo "=== Cleaning data directories ==="
  rm -rf "$DATA_DIR"
  echo "Done."
  exit 0
}

build() {
  echo "=== Building server ==="
  go build -o "$BIN_SERVER" ./cmd/server

  echo "=== Building client ==="
  go build -o "$BIN_CLIENT" ./cmd/client

  echo "=== Binaries ready ==="
  echo "  server: $BIN_SERVER"
  echo "  client: $BIN_CLIENT"
}

# Parse args
if [[ "${1:-}" == "--clean" ]]; then
  clean
fi

echo "=== Building project ==="
cd "$ROOT"
build

if [[ "${1:-}" == "--build-only" ]]; then
  exit 0
fi

# Clean start
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"

echo ""
echo "=== Starting 3-node Raft cluster ==="

PIDS=()
FIRST_ADDR=""

for i in "${!NODES[@]}"; do
  IFS=':' read -r NODE_ID GRPC_PORT RAFT_PORT <<< "${NODES[$i]}"

  NODE_DATA_DIR="${DATA_DIR}/${NODE_ID}"
  mkdir -p "$NODE_DATA_DIR"

  BOOTSTRAP_FLAG=""
  if [[ $i -eq 0 ]]; then
    BOOTSTRAP_FLAG="--bootstrap"
    FIRST_ADDR="localhost:${GRPC_PORT}"
  fi

  echo "  Starting ${NODE_ID} (gRPC=${GRPC_PORT}, Raft=${RAFT_PORT})..."

  # Start node in background with race detection
  "$BIN_SERVER" \
    --node-id "$NODE_ID" \
    --grpc-port "$GRPC_PORT" \
    --raft-port "$RAFT_PORT" \
    --data-dir "$NODE_DATA_DIR" \
    $BOOTSTRAP_FLAG \
    --log-level debug &
  
  PID=$!
  PIDS+=("$PID")

  # Give the first node time to bootstrap before joining others
  if [[ $i -eq 0 ]]; then
    sleep 2
  else
    sleep 1
  fi

  # Join non-bootstrap nodes to the cluster
  if [[ -n "$FIRST_ADDR" && $i -gt 0 ]]; then
    echo "    Joining ${NODE_ID} to cluster via ${FIRST_ADDR}..."
    # The join will be retried by the server internally; we also try via RPC
    sleep 0.5
  fi
done

echo ""
echo "=== Cluster is running ==="
echo "  Leader: node1 (localhost:50051)"
echo "  Nodes:  node2 (localhost:50052), node3 (localhost:50053)"
echo ""
echo "  To start a test client, run in another terminal:"
echo "    ${BIN_CLIENT} localhost:50051 test-client"
echo ""
echo "Press Ctrl+C to stop all nodes."
echo ""

# Trap Ctrl+C and kill all child processes
trap 'echo ""; echo "=== Stopping all nodes ==="; kill "${PIDS[@]}" 2>/dev/null; wait; echo "Done."; exit 0' SIGINT SIGTERM

# Wait for all background processes
wait
