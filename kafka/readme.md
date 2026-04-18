# Kafka Experiment

A hands-on experiment with Apache Kafka in Go. Demonstrates producers, consumers
(fan-out / competing-consumers), and a live inspection tool.

## Architecture

| Package | Role |
|---|---|
| `claims` | Shared data types and topic constants |
| `producer` | Publishes random insurance claims to Kafka topics |
| `consumer` | Subscribes to one or more topics; prints live stats |
| `inspector` | Displays per-partition offsets and consumer-group lag |
| `dlq` | Reads and optionally replays messages from the dead-letter topic |

**Topics**: `claims-auto`, `claims-home`, `claims-life` — 5 partitions each. `claims-dlq` — 1 partition, dead-letter sink.  
Messages are keyed by `CustomerID` (hash partitioner) so the same customer always
hits the same partition, avoiding hot spots.

> **Per-customer ordering guarantee** — Kafka guarantees total ordering within a
> partition. Because every claim from a given customer is routed to the same
> partition, consumers always receive that customer's claims in the exact order
> they were produced. There is no ordering guarantee *across* different customers,
> since those may land on different partitions processed concurrently.

## Prerequisites

- Docker + Docker Compose
- Go 1.25+

## Start Kafka

```sh
cd kafka
docker compose up -d
```

Starts Apache Kafka 4.1.2 in **KRaft mode** (no ZooKeeper) and creates the three
topics automatically via a one-shot init container.

## Producer

```sh
go run ./kafka/producer/
```

| Flag | Default | Description |
|---|---|---|
| `-broker` | `localhost:9092` | Kafka broker address |
| `-workers` | `5` | Number of concurrent producer goroutines |
| `-poison-rate` | `0.0` | Fraction of messages sent with malformed JSON to simulate poison pills |

Press `Ctrl+C` to stop gracefully.

## Consumers

### Fan-out — one consumer per topic

Each consumer group receives **all messages** independently:

```sh
# Terminal A
go run ./kafka/consumer/ -topic claims-auto

# Terminal B
go run ./kafka/consumer/ -topic claims-home

# Terminal C
go run ./kafka/consumer/ -topic claims-life
```

### All-topics consumer

```sh
go run ./kafka/consumer/ -topic claims-auto -topic claims-home -topic claims-life
```

### Competing consumers (shared group)

Multiple instances with the same `-group` ID split the partitions between them:

```sh
go run ./kafka/consumer/ -topic claims-auto -group workers
go run ./kafka/consumer/ -topic claims-auto -group workers  # second terminal
```

| Flag | Default | Description |
|---|---|---|
| `-topic` | all topics | Topic to subscribe to (repeatable) |
| `-broker` | `localhost:9092` | Kafka broker address |
| `-group` | derived from topics | Consumer group ID |
| `-duckdb-path` | `/tmp` | DuckDB file path or directory; if directory, file is `<group>.duckdb` |
| `-flush-interval` | `10s` | Flush buffered events to DuckDB every interval |
| `-batch-size` | `1000` | Flush buffered events when this many are queued |
| `-chaos-crash-after` | `0` | Crash the process after N messages (0 = disabled) |
| `-chaos-fail-db-prob` | `0.0` | Probability (0–1) of a simulated DB flush failure per attempt |
| `-chaos-slow-ms` | `0` | Artificial delay (ms) added per message (0 = disabled) |

DuckDB allows one writer process per database file. Running multiple consumers is
safe as long as each process writes to a different DuckDB file (the default
directory behavior does this automatically per consumer group).

## Inspector

```sh
# Run once
go run ./kafka/inspector/

# Refresh every 3 seconds
go run ./kafka/inspector/ -interval 3s
```

| Flag | Default | Description |
|---|---|---|
| `-broker` | `localhost:9092` | Kafka broker address |
| `-interval` | `5s` | Refresh interval (`0` = run once) |

Shows per-partition oldest/newest offsets and lag per consumer group.

## Dead-Letter Queue (DLQ)

Messages that cannot be processed (poison pills, permanent DB failures) are
forwarded to `claims-dlq` with failure metadata in Kafka headers. The consumer
retries a flush up to 3 times with exponential backoff before routing to the DLQ.

```sh
# Inspect all dead letters
go run ./kafka/dlq/

# Inspect and replay them back to their original topics
go run ./kafka/dlq/ -replay
```

| Flag | Default | Description |
|---|---|---|
| `-broker` | `localhost:9092` | Kafka broker address |
| `-replay` | `false` | Re-publish each dead letter to its original topic |

## Fault tolerance

Stop the producer mid-run with `Ctrl+C`, then restart it:

```sh
go run ./kafka/producer/
```

Consumers resume from their **last committed offset** — no messages are lost. On
the very first start, consumers begin at `OffsetNewest` (i.e. they skip messages
produced before they joined). After that, committed offsets are used on every
restart (at-least-once delivery semantics).

See [FAILURES.md](FAILURES.md) for step-by-step instructions on simulating
consumer crashes, database failures, poison pills, and slow consumers.

## Stop Kafka

```sh
cd kafka
docker compose down
```