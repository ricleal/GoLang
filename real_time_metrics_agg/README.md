# Distributed Real-Time Metrics Aggregator

A production-grade, distributed real-time metrics aggregation and alerting engine built in Go — designed for the high-concurrency, low-latency patterns that LiveKit deals with daily.

## Architecture

### System Overview

```mermaid
graph TB
    subgraph Clients["Metric Clients"]
        C1["Client 1<br/>gRPC stream"]
        C2["Client 2<br/>gRPC stream"]
        CN["Client N<br/>gRPC stream"]
    end

    subgraph Node1["Server Node 1 (Leader)<br/>:50051"]
        GRPC1["gRPC Server<br/>StreamMetrics"]
        AGG1["Sliding Window<br/>Aggregator"]
        ALERT1["Alert Engine<br/>WARN / CRITICAL"]
        RAFT1["Raft Node<br/>Leader"]
        GRPC1 --> AGG1
        AGG1 --> ALERT1
        ALERT1 -.->|replicate| RAFT1
    end

    subgraph Node2["Server Node 2 (Follower)<br/>:50052"]
        GRPC2["gRPC Server"]
        AGG2["Sliding Window<br/>Aggregator"]
        ALERT2["Alert Engine"]
        RAFT2["Raft Node<br/>Follower"]
        GRPC2 --> AGG2
        AGG2 --> ALERT2
        ALERT2 -.->|replicate| RAFT2
    end

    subgraph Node3["Server Node 3 (Follower)<br/>:50053"]
        GRPC3["gRPC Server"]
        AGG3["Sliding Window<br/>Aggregator"]
        ALERT3["Alert Engine"]
        RAFT3["Raft Node<br/>Follower"]
        GRPC3 --> AGG3
        AGG3 --> ALERT3
        ALERT3 -.->|replicate| RAFT3
    end

    C1 -->|gRPC StreamMetrics| GRPC1
    C2 -->|gRPC StreamMetrics| GRPC2
    CN -->|gRPC StreamMetrics| GRPC1

    GRPC1 -.->|aggregated<br/>snapshots| C1
    GRPC2 -.->|aggregated<br/>snapshots| C2

    RAFT1 ===|Raft TCP :50070| RAFT2
    RAFT1 ===|Raft TCP :50070| RAFT3
    RAFT2 ===|Raft TCP :50071| RAFT3

    style C1 fill:#e1f5fe,stroke:#0288d1
    style C2 fill:#e1f5fe,stroke:#0288d1
    style CN fill:#e1f5fe,stroke:#0288d1
    style Node1 fill:#e8f5e9,stroke:#388e3c
    style Node2 fill:#e8f5e9,stroke:#388e3c
    style Node3 fill:#e8f5e9,stroke:#388e3c
    style RAFT1 fill:#fff3e0,stroke:#f57c00
    style RAFT2 fill:#fff3e0,stroke:#f57c00
    style RAFT3 fill:#fff3e0,stroke:#f57c00
```

### Data Flow

```mermaid
sequenceDiagram
    participant C as Metric Client
    participant S as gRPC Server
    participant A as Aggregator
    participant AE as Alert Engine
    participant R as Raft Cluster

    Note over C,R: gRPC Bidirectional Stream established
    C->>S: StreamMetrics()
    S-->>C: stream accepted

    loop Every ~50ms
        C->>S: Metric(name, value, ts)
        S->>A: AddMetric(name, value, ts)
        A->>A: advance() sliding window
        A->>A: insert into bucket
        S->>A: Snapshot(name)
        A-->>S: AggregatedMetric(avg, p95, p99, ...)
        S->>AE: Evaluate(snapshot)
        alt threshold crossed
            AE-->>S: [Alert]
            S->>R: ReplicateAlert() [if leader]
            R-->>S: committed
        end
    end

    loop Every 2s
        S->>A: SnapshotAll()
        A-->>S: [AggregatedMetric...]
        S-->>C: aggregated snapshots
    end
```

## Key Components

### 1. Transport Layer — gRPC Bidirectional Streaming (`internal/transport/`)

Uses gRPC bidirectional streaming to establish long-lived connections between clients and servers. The proto service definition:

```protobuf
service MetricsAggregator {
  rpc StreamMetrics(stream Metric) returns (stream AggregatedMetric);
  rpc SetThreshold(SetThresholdRequest) returns (SetThresholdResponse);
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
  rpc JoinCluster(JoinClusterRequest) returns (JoinClusterResponse);
}
```

- **Backpressure**: Bounded channel (`streamSem`) limits concurrent streams per node
- **Graceful degradation**: Alerts are replicated to Raft best-effort; failures don't block metrics ingestion

### 2. Sliding Window Aggregator (`internal/aggregator/`)

Thread-safe, lock-based sliding window over configurable duration (default: 10s window, 1s buckets).

```mermaid
%%{init: {'theme': 'neutral'}}%%
graph LR
    subgraph T0["T = 0s"]
        B0_0["[0-1s]<br/>values: A,B"]
        B0_1["[1-2s]<br/>values: C"]
        B0_2["[2-3s]<br/>values: D,E,F"]
        B0_3["[3-4s]<br/>values: ∅"]
        B0_4["[4-5s]<br/>values: ∅"]
        B0_5["[5-6s]<br/>values: ∅"]
        B0_6["[6-7s]<br/>values: ∅"]
        B0_7["[7-8s]<br/>values: ∅"]
        B0_8["[8-9s]<br/>values: ∅"]
        B0_9["[9-10s]<br/>values: ∅"]
    end

    subgraph T1["T = 4s — advance by 2 buckets"]
        B1_0["[2-3s]<br/>values: D,E,F"]
        B1_1["[3-4s]<br/>values: ∅"]
        B1_2["[4-5s]<br/>values: ∅"]
        B1_3["[5-6s]<br/>values: ∅"]
        B1_4["[6-7s]<br/>values: ∅"]
        B1_5["[7-8s]<br/>values: ∅"]
        B1_6["[8-9s]<br/>values: ∅"]
        B1_7["[9-10s]<br/>values: ∅"]
        B1_8["[10-11s]<br/>values: G"]
        B1_9["[11-12s]<br/>values: ∅"]
    end

    T0 -.->|"advance: now=T+4s, windowStart=T-6s, no buckets dropped"| T1

    style B0_0 fill:#ffcdd2,stroke:#e53935
    style B0_1 fill:#ffcdd2,stroke:#e53935
    style B0_2 fill:#c8e6c9,stroke:#388e3c
    style B1_0 fill:#c8e6c9,stroke:#388e3c
    style B1_8 fill:#bbdefb,stroke:#1976d2
    style T0 fill:#f5f5f5,stroke:#9e9e9e
    style T1 fill:#f5f5f5,stroke:#9e9e9e
```

- **Concurrency**: `sync.RWMutex` per metric state → switched to full `sync.Mutex` on `Snapshot()` because `advance()` mutates bucket state
- **Windowing**: Bucket-based sliding window that only drops buckets entirely before `[now - window]` — preserves data still in range
- **Stats**: Sum, Avg, Min, Max, Count, P50, P95, P99 percentiles

### 3. Alert Engine (`internal/alert/`)

Configurable threshold-based alerting with two severity levels:

- **WARN**: Triggered when metric crosses `warn` threshold
- **CRITICAL**: Triggered when metric crosses `critical` threshold

Thresholds are set dynamically via gRPC (`SetThreshold` RPC).

### 4. Raft Consensus (`internal/raft/`)

Uses `hashicorp/raft` for distributed consensus and leader election.

```mermaid
sequenceDiagram
    participant N1 as Node 1 (Leader)
    participant N2 as Node 2 (Follower)
    participant N3 as Node 3 (Follower)

    Note over N1,N3: Leader Election
    N2->>N2: heartbeat timeout
    N2->>N2: start election, term++
    N2->>N1: RequestVote
    N2->>N3: RequestVote
    N1-->>N2: Vote granted
    N3-->>N2: Vote granted
    N2->>N2: elected Leader (term 2)
    N2->>N1: AppendEntries (heartbeat)
    N2->>N3: AppendEntries (heartbeat)

    Note over N1,N3: Alert Replication
    N1->>N1: Alert triggered<br/>(client connected to Node 1)
    N1->>N2: ReplicateAlert() → Raft Apply
    N1->>N3: ReplicateAlert() → Raft Apply
    N2->>N2: FSM.Apply() → append alert
    N3->>N3: FSM.Apply() → append alert
    N2-->>N1: committed
    N3-->>N1: committed

    Note over N1,N3: Graceful Step-down
    N1->>N1: SIGTERM received
    N1->>N1: step down, shutdown
    N2->>N2: heartbeat timeout
    N2->>N2: start election, term++
    N3-->>N2: vote
    N2->>N2: elected Leader (term 3)
```

- **Leader Election**: Automatic election via Raft's heartbeat mechanism
- **State Replication**: Alert events are replicated across the cluster as Raft log entries
- **FSM**: In-memory `ClusterFSM` stores recent alert events (max 1000)
- **Snapshots**: Periodic snapshots every 30s / 128 entries

### 5. Graceful Shutdown & Health (`cmd/server/main.go`)

```mermaid
sequenceDiagram
    participant OS as OS Signal
    participant Main as main goroutine
    participant gRPC as gRPC Server
    participant Raft as Raft Cluster

    OS->>Main: SIGINT / SIGTERM / SIGQUIT
    Main->>Main: signal.NotifyContext<br/>ctx.Done()
    Main->>gRPC: GracefulStop()
    gRPC->>gRPC: stop accepting new RPCs
    gRPC->>gRPC: drain in-flight streams
    gRPC-->>Main: stopped
    Main->>Raft: Shutdown()
    Raft->>Raft: leader steps down
    Raft->>Raft: persist state
    Raft-->>Main: shutdown complete
    Main->>Main: os.Exit(0)

    Note over OS,Raft: Orderly drain: gRPC first, then Raft —<br/>ensures no data loss on scale-down
```

- **OS Signal Handling**: `SIGINT`, `SIGTERM`, `SIGQUIT` trigger graceful shutdown via `signal.NotifyContext`
- **Drain Sequence**: gRPC `GracefulStop()` → Raft shutdown (leader steps down, cluster re-elects)
- **Health Check**: gRPC health endpoint reports `SERVING` status and leader state

## Getting Started

### Prerequisites

- Go 1.22+
- `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc`

### Build

```bash
cd real_time_metrics_agg
go build -race -o build/metrics-server ./cmd/server
go build -race -o build/metrics-client ./cmd/client
```

### Run Single Node

```bash
# Terminal 1: Start server
./build/metrics-server \
  --node-id node1 \
  --grpc-port 50051 \
  --raft-port 50070 \
  --data-dir /tmp/raft-metrics \
  --bootstrap \
  --window 10s

# Terminal 2: Start client
./build/metrics-client 127.0.0.1:50051 test-client-1
```

### Run 3-Node Cluster

```bash
./scripts/run-cluster.sh
```

This starts nodes on ports:
| Node  | gRPC Port | Raft Port |
|-------|-----------|-----------|
| node1 | 50051     | 50070     |
| node2 | 50052     | 50071     |
| node3 | 50053     | 50072     |

Then start clients:
```bash
./build/metrics-client 127.0.0.1:50051 client-a
./build/metrics-client 127.0.0.1:50052 client-b
```

## Configuration

| Flag            | Env           | Default     | Description                      |
|-----------------|---------------|-------------|----------------------------------|
| `--node-id`     | `NODE_ID`     | `node1`     | Unique node identifier           |
| `--grpc-port`   | `GRPC_PORT`   | `50051`     | gRPC server port                 |
| `--raft-port`   | `RAFT_PORT`   | `50070`     | Raft consensus port              |
| `--data-dir`    | `DATA_DIR`    | `/tmp/raft-metrics` | Raft data directory      |
| `--bootstrap`   | —             | `false`     | Bootstrap Raft cluster (first node) |
| `--join`        | `JOIN_ADDR`   | `""`        | Join existing cluster            |
| `--window`      | —             | `10s`       | Sliding window duration          |
| `--bucket`      | —             | `1s`        | Aggregation bucket interval      |
| `--log-level`   | `LOG_LEVEL`   | `info`      | Log level (debug/info/warn/error)|

## Key Distributed Systems Concepts Demonstrated

### Data Races
Always run with `-race` flag. The aggregator uses `sync.RWMutex` to protect shared metric state. Running `go build -race` and `go run -race` catches subtle concurrency issues.

### Backpressure
- **Stream level**: Bounded channel (`streamSem`) limits concurrent gRPC streams per node
- **Aggregate level**: Buffered channel (`aggCh`) with `select/default` drops aggregates when client is slow

### Graceful Shutdown
- OS signals (`SIGINT`, `SIGTERM`) trigger ordered shutdown
- gRPC stops accepting new requests, drains in-flight streams
- Raft leader steps down, cluster re-elects

### Idempotency
- Metric timestamps are client-provided, enabling dedup on reconnect (TODO)
- Raft FSM apply is idempotent (appends to alert list)

## Project Structure

```
real_time_metrics_agg/
├── proto/
│   └── metrics.proto          # gRPC service definition
├── pb/                         # Generated protobuf code
│   ├── metrics.pb.go
│   └── metrics_grpc.pb.go
├── cmd/
│   ├── server/main.go          # Entry point with graceful shutdown
│   └── client/main.go          # Metrics streaming client
├── internal/
│   ├── config/config.go        # CLI flags + env var configuration
│   ├── aggregator/aggregator.go # Sliding window aggregation
│   ├── alert/engine.go         # Threshold-based alerting
│   ├── raft/node.go            # Raft consensus (hashicorp/raft)
│   └── transport/
│       ├── server.go           # gRPC server implementation
│       └── client.go           # gRPC client + simulation
└── scripts/
    └── run-cluster.sh          # 3-node cluster launcher
```

## LiveKit Relevance

This project directly exercises the exact patterns LiveKit's Distributed Systems Engineer role requires:

| LiveKit Requirement | This Project |
|---------------------|-------------|
| **Go fluency** | Pure Go implementation, gRPC, Raft integration |
| **Low-latency transport** | gRPC bidirectional streaming, no polling |
| **Concurrency** | `sync.RWMutex`, channels, bounded semaphores |
| **Distributed consensus** | Hashicorp Raft: leader election, log replication |
| **Observability** | Structured logging, health checks, metrics mapping |
| **Graceful degradation** | Backpressure, graceful shutdown, Raft failover |
| **Race detection** | `-race` flag throughout development |