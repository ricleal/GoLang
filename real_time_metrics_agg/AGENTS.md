# Project Guidelines — Real-Time Metrics Aggregator

## Code Style

- Go 1.22+ with standard formatting (`gofmt`/`go fmt`)
- Use `log` stdlib for server logging, `hclog` for Raft internals
- Error handling: always check returned errors; wrap with `fmt.Errorf("context: %w", err)`
- Prefer `sync.RWMutex` for read-heavy concurrency; upgrade to `sync.Mutex` when the critical section mutates state
- All exported types and functions must have doc comments
- Test files go alongside the code they test (`_test.go` in the same package)

## Architecture

```
cmd/
  server/main.go   — entry point, wires gRPC + Raft + aggregator
  client/main.go   — simulated metric client / demo
internal/
  aggregator/       — sliding window, bucket-based time aggregation
  alert/            — threshold engine (WARN / CRITICAL)
  raft/             — hashicorp/raft wrapper (FSM, cluster management)
  transport/        — gRPC server + client (bidirectional streaming)
  config/           — CLI flags + env var config
pb/                 — generated protobuf code (do not edit by hand)
proto/              — .proto service definition
```

## Build and Test

```bash
# Build with race detector
go build -race -o build/metrics-server ./cmd/server
go build -race -o build/metrics-client ./cmd/client

# Run all tests with race detector
go test -race -count=1 ./...

# Run a specific package
go test -race -v -count=1 ./internal/aggregator/...

# Regenerate protobuf (if proto/metrics.proto changes)
protoc --go_out=. --go_opt=module=github.com/ricleal/GoLang/real_time_metrics_agg \
  --go-grpc_out=. --go-grpc_opt=module=github.com/ricleal/GoLang/real_time_metrics_agg \
  proto/metrics.proto

# Run a single-node cluster
./build/metrics-server --node-id node1 --grpc-port 50051 --raft-port 50070 \
  --data-dir /tmp/raft-metrics --bootstrap --window 10s

# Run a client
./build/metrics-client 127.0.0.1:50051 test-client-1
```

## Key Conventions

- **gRPC streaming**: bidirectional `StreamMetrics` RPC — client pushes `Metric` messages, server streams back `AggregatedMetric` snapshots every 2s
- **Sliding window**: bucket-based; `advance()` only drops buckets entirely before `[now - window]`. `Snapshot()` acquires a full write lock because `advance()` mutates bucket state
- **Raft**: alert events are replicated via `ReplicateAlert()` (leader only, best-effort). The `ClusterFSM` caps at 1000 entries
- **Backpressure**: bounded channel (`streamSem`) limits concurrent gRPC streams; `select/default` in ticker loop drops aggregates when a client is slow
- **Graceful shutdown**: `signal.NotifyContext` triggers gRPC `GracefulStop()` → Raft `Shutdown()`
- **Alert threshold precedence**: `CRITICAL` takes priority over `WARN` — if `Avg >= Critical`, only a CRITICAL alert is produced
- **Config**: CLI flags with env var fallback (`NODE_ID`, `GRPC_PORT`, `RAFT_PORT`, etc.)
