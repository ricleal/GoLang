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

**Topics**: `claims-auto`, `claims-home`, `claims-life` — 5 partitions each.  
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

## Fault tolerance

Stop the producer mid-run with `Ctrl+C`, then restart it:

```sh
go run ./kafka/producer/
```

Consumers resume from their **last committed offset** — no messages are lost. On
the very first start, consumers begin at `OffsetNewest` (i.e. they skip messages
produced before they joined). After that, committed offsets are used on every
restart (at-least-once delivery semantics).

## Stop Kafka

```sh
cd kafka
docker compose down
```