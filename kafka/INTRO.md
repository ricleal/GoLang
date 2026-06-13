# Kafka Concepts

A concise introduction to Apache Kafka's core abstractions: **topics**, **partitions**, and **consumer groups**.

---

## Topics

A **topic** is a named stream of messages (records). Producers write to topics,
consumers read from them. Topics are the primary unit of organisation in Kafka —
analogous to a table in a database or a queue in traditional message brokers.

```sh
# Create a topic with default settings
kafka-topics.sh --bootstrap-server localhost:9092 --create --topic claims-auto
```

Unlike traditional queues, messages in Kafka are **not deleted after being read**.
They persist for a configurable retention period (e.g. 7 days, or 1 GB of data),
which allows consumers to rewind and replay messages.

```sh
# List all topics
kafka-topics.sh --bootstrap-server localhost:9092 --list

# Describe a topic (partitions, replication, config)
kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic claims-auto
```

---

## Partitions

A **partition** is an ordered, immutable sequence of messages within a topic.
Each partition is a single append-only log stored on disk.

```
Topic "claims-auto"
┌────────────────────────────────────────────┐
│ Partition 0  │ msg-0 │ msg-1 │ msg-2 │ ... │
├────────────────────────────────────────────┤
│ Partition 1  │ msg-0 │ msg-1 │ msg-2 │ ... │
├────────────────────────────────────────────┤
│ Partition 2  │ msg-0 │ msg-1 │ msg-2 │ ... │
├────────────────────────────────────────────┤
│ Partition 3  │ msg-0 │ msg-1 │ msg-2 │ ... │
├────────────────────────────────────────────┤
│ Partition 4  │ msg-0 │ msg-1 │ msg-2 │ ... │
└────────────────────────────────────────────┘
```

### Key concepts

| Concept | Meaning |
|---|---|
| **Offset** | A sequential ID assigned to every message within a partition. Offsets start at 0 and increment by 1. |
| **Key** | An optional value used to determine which partition a message lands on. Messages with the same key always go to the same partition. |
| **Ordering** | Kafka guarantees **total order within a partition** — messages are read in the exact order they were written. There is **no ordering across partitions**. |

### Why multiple partitions?

1. **Parallelism** — A topic with N partitions can be consumed by up to N
   consumers in parallel (one per partition).
2. **Horizontal scaling** — Partitions can be distributed across multiple brokers
   in a cluster.
3. **Throughput** — Writes and reads are distributed across partitions, so more
   partitions = more throughput (up to the cluster's capacity).

### Partitioning strategies

| Strategy | Behaviour | Use case |
|---|---|---|
| **Round-robin** (no key) | Messages are spread evenly across all partitions | Load balancing, no ordering needed |
| **Hash partitioner** (keyed) | `hash(key) % num_partitions` routes the same key to the same partition | Per-key ordering (e.g. per customer) |
| **Custom partitioner** | User-defined logic | Geo-routing, tenant isolation |

> In this experiment, claims are keyed by `CustomerID` using a hash partitioner.
> This guarantees that all claims from the same customer arrive in order on the
> same partition — and therefore are processed in order by the consumer.

### Important numbers

```
5 partitions per topic   → up to 5 consumers in the same group can run in parallel
1 partition for the DLQ  → the DLQ is always read by a single consumer
```

---

## Consumer Groups

A **consumer group** is a set of consumers that cooperate to read messages from
one or more topics. Kafka assigns each partition to exactly **one consumer**
within the group.

```
Consumer group "workers"
┌────────────────┐    ┌────────────────┐    ┌────────────────┐
│ Consumer A     │    │ Consumer B     │    │ Consumer C     │
│ Partitions 0,1 │    │ Partitions 2,3 │    │ Partition 4    │
└────────────────┘    └────────────────┘    └────────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                 ┌────────────┴────────────┐
                 │ Topic "claims-auto"     │
                 │ Partitions 0 .. 4       │
                 └─────────────────────────┘
```

### Fan-out vs Competing consumers

**Fan-out** — Each consumer group receives **all messages independently**.
If two groups (`group-claims-auto` and `group-claims-home`) subscribe to the
same topic, both get a full copy of every message.

```
Producer ──→ claims-auto ──→ group-claims-auto  (all 5 partitions)
                         └─→ group-claims-home   (all 5 partitions)
```

**Competing consumers** — Multiple consumers **within the same group** split the
partitions between them. Each message goes to exactly one consumer.

```
Producer ──→ claims-auto ──→ group "workers" ──→ Consumer A (partitions 0,1)
                                               └─→ Consumer B (partitions 2,3)
                                               └─→ Consumer C (partition 4)
```

### Rebalancing

When a consumer joins or leaves a group, Kafka triggers a **rebalance** —
partitions are reassigned among the remaining members. The consumer must
flush any in-flight work and re-acquire its partition assignment.

```
Before (3 consumers)           After Consumer B crashes (2 consumers)
┌─────┐ ┌─────┐ ┌─────┐       ┌─────┐ ┌─────┐
│  A  │ │  B  │ │  C  │       │  A  │ │  C  │
│ 0,1 │ │ 2,3 │ │  4  │  ──→  │0,1,2│ │ 3,4 │
└─────┘ └─────┘ └─────┘       └─────┘ └─────┘
```

### Offset management

Each consumer group tracks its **committed offset** per partition — the next
message the group expects to read.

```
Partition 0 ──→ [msg-0] [msg-1] [msg-2] [msg-3] [msg-4] ──→ ...
                    ↑
              committed offset = 2 (next read will be msg-2)
```

- Offsets are committed to an internal Kafka topic (`__consumer_offsets`).
- On restart, a consumer resumes from its last committed offset.
- **At-least-once delivery**: offsets are committed _after_ processing. If the
  consumer crashes before committing, messages are re-delivered on restart.

```sh
# Inspect consumer group offsets and lag
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group group-claims-auto --describe
```

### Lag

**Lag** = newest offset − committed offset. It represents how far behind a
consumer group is.

```
              committed     newest
                  ↓            ↓
Partition 0: [0][1][2][3][4][5][6]
                  │←─ lag = 4 ─→│
```

The inspector tool in this project shows live lag for every consumer group:

```
  GROUP              TOPIC        PARTITION  COMMITTED  NEWEST  LAG
  group-claims-auto  claims-auto  0          100        980     880
  group-claims-auto  claims-auto  1          95         970     875
```

---

## Putting it all together

```
               ┌──────────────────────────────────┐
               │          Kafka Cluster           │
               │                                  │
Producer ─────→│ claims-auto (5 partitions)       │←──── Consumer group A
               │   ├─ partition 0                 │      (fan-out)
               │   ├─ partition 1                 │
               │   ├─ partition 2                 │←──── Consumer group B
               │   ├─ partition 3                 │      (fan-out)
               │   └─ partition 4                 │
               │                                  │
               │ claims-home  (5 partitions)      │←──── Consumer group C
               │ claims-life  (5 partitions)      │←──── Consumer group D
               │                                  │
Producer ─────→│ claims-dlq   (1 partition)       │←──── DLQ reader
               └──────────────────────────────────┘
                       │
                       ▼
                 DuckDB (per consumer group)
```

Each producer writes to the appropriate topic using a **hash partitioner** keyed
by `CustomerID`. Each consumer group reads from one or more topics; group
members split partitions among themselves via **rebalancing**. Failed messages
land in the **dead-letter queue** for later inspection or replay.

---

## Further reading

- [Official Kafka documentation](https://kafka.apache.org/documentation/)
- [FAILURES.md](./FAILURES.md) — failure scenarios reproduced in this experiment
- [readme.md](./readme.md) — project overview and usage
