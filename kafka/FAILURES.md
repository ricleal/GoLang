# Failure Simulation Guide

This guide walks through four failure scenarios you can reproduce locally.
Each scenario shows what to run, what to observe, and how the system recovers.

## Prerequisites

Kafka must be running and the `claims-dlq` topic must exist:

```sh
cd kafka
docker compose down -v   # clean slate (drops all offsets and data)
docker compose up -d
```

Keep the inspector running in a dedicated terminal throughout all scenarios:

```sh
go run ./kafka/inspector/ -interval 3s
```

---

## Scenario 1 — Consumer Crash (OOM-kill simulation)

**What happens**: The consumer process exits hard (`os.Exit(1)`) after processing
N messages, simulating an OOM-kill or segfault. Because Kafka offsets are only
committed *after* a successful DuckDB flush, the in-flight batch is **re-delivered**
on restart — no data is lost.

### Step by step

```sh
# Terminal A — producer running continuously
go run ./kafka/producer/

# Terminal B — consumer that crashes after 50 messages
go run ./kafka/consumer/ -topic claims-auto -chaos-crash-after 50
```

**Observe**:
- The consumer logs `chaos: crashing consumer` and exits.
- The inspector shows a lag spike on `claims-auto` (the uncommitted batch sits in Kafka).
- Restart the consumer (same command without the flag):

```sh
go run ./kafka/consumer/ -topic claims-auto
```

- The consumer resumes from the last committed offset. Lag drops back to zero.
- The DLQ stays empty — no messages were lost or rerouted.

```sh
go run ./kafka/dlq/   # should print "claims-dlq is empty"
```

---

## Scenario 2 — Database Failure with DLQ Fallback

**What happens**: The consumer simulates a DB failure on every flush attempt
(100% probability). After 3 retries with exponential backoff (1 s, 2 s, 4 s),
the in-flight batch is forwarded to `claims-dlq` and offsets are committed so
the consumer keeps moving.

### Step by step

```sh
# Terminal A — producer
go run ./kafka/producer/

# Terminal B — consumer with 100% DB failure rate, small batch so DLQ fills quickly
go run ./kafka/consumer/ \
    -topic claims-auto \
    -batch-size 5 \
    -flush-interval 5s \
    -chaos-fail-db-prob 1.0
```

**Observe in consumer logs**:
```
level=ERROR msg="chaos: injected DB failure" attempt=1 pending=5
level=WARN  msg="flush retry"               attempt=2 backoff=1s
level=ERROR msg="chaos: injected DB failure" attempt=2 pending=5
level=WARN  msg="flush retry"               attempt=3 backoff=2s
level=ERROR msg="chaos: injected DB failure" attempt=3 pending=5
level=ERROR msg="all flush retries exhausted, routing to DLQ" pending=5
level=WARN  msg="batch drained to DLQ"      count=5
```

After a few minutes, inspect the DLQ:

```sh
go run ./kafka/dlq/
```

You will see a table like:

```
DLQ_OFFSET  ORIG_TOPIC    ORIG_PART  ORIG_OFFSET  CONSUMER_GROUP        FAILED_AT             ERROR
0           claims-auto   2          1041         group-claims-auto     2026-04-18T10:05:01Z  chaos: simulated database failure
1           claims-auto   0          877          group-claims-auto     2026-04-18T10:05:01Z  chaos: simulated database failure
...
```

Once the "database" is healthy again (stop using the chaos flag), replay the
dead letters back to the original topic:

```sh
go run ./kafka/dlq/ -replay
```

A healthy consumer will re-process them.

---

## Scenario 3 — Poison Pills (Malformed Messages)

**What happens**: The producer injects a small fraction of messages with broken
JSON. The consumer cannot unmarshal them, routes each one to `claims-dlq`
immediately (no retry needed), and continues processing good messages without
interruption.

### Step by step

```sh
# Terminal A — producer injecting ~5% poison pills
go run ./kafka/producer/ -poison-rate 0.05

# Terminal B — consumer (no chaos flags needed)
go run ./kafka/consumer/ -topic claims-auto
```

**Observe in consumer logs**:
```
level=ERROR msg="unmarshal (poison pill)" error="..." topic=claims-auto offset=42
level=WARN  msg="message routed to DLQ"   original_topic=claims-auto original_offset=42 reason="unmarshal-error: ..."
```

The consumer's throughput counter keeps climbing — good messages are unaffected.

Inspect the DLQ to see the poison pills:

```sh
go run ./kafka/dlq/
```

The `dlq-error` column shows the JSON parse error. The raw broken bytes are
preserved in the DLQ message value so you can debug them later.

After fixing the producer (remove `-poison-rate`), the DLQ accumulates no new
entries. You can **not** replay poison pills back to the original topic (they
would fail again); instead you examine them and fix the producer.

---

## Scenario 4 — Slow Consumer (Growing Lag)

**What happens**: Each message takes 200 ms to process. The producer runs at
full speed. Lag grows steadily in the inspector. Adding a second competing
consumer (same group) cuts the lag in half because the partitions are split.

### Step by step

```sh
# Terminal A — producer at full speed
go run ./kafka/producer/

# Terminal B — slow consumer
go run ./kafka/consumer/ -topic claims-auto -group workers -chaos-slow-ms 200
```

Watch the inspector — lag on `claims-auto` grows:
```
GROUP    TOPIC        PARTITION  COMMITTED  NEWEST  LAG
workers  claims-auto  0          100        980     880
workers  claims-auto  1          95         970     875
...
```

Now add a second consumer **with the same group**:

```sh
# Terminal C — second consumer in the same group
go run ./kafka/consumer/ -topic claims-auto -group workers -chaos-slow-ms 200
```

Kafka rebalances: each consumer now owns ~2–3 partitions instead of 5. Lag
growth slows. Adding a third instance would slow it further (up to 5 consumers
for 5 partitions). Beyond that, extra consumers sit idle.

---

## Combining Scenarios

You can stack chaos flags:

```sh
# slow consumer that also crashes after 200 messages
go run ./kafka/consumer/ \
    -topic claims-auto \
    -chaos-slow-ms 100 \
    -chaos-crash-after 200
```

```sh
# producer with poison pills feeding a consumer with DB failures
go run ./kafka/producer/ -poison-rate 0.02
go run ./kafka/consumer/ \
    -batch-size 10 \
    -chaos-fail-db-prob 0.3
```

In this last case you will see both poison pills and batch failures landing in
the DLQ simultaneously, each with distinct `dlq-error` values.

---

## DLQ Replay Workflow

```
producer → claims-auto/home/life
               │
               ▼ (unmarshal error OR all DB retries exhausted)
           claims-dlq
               │
               ▼ go run ./kafka/dlq/ -replay
         claims-auto/home/life   (original topic, re-processed by healthy consumer)
```

1. Fix the root cause (bad producer code, DB restored).
2. Run `go run ./kafka/dlq/ -replay` once.
3. The healthy consumer re-processes the replayed messages.
4. Verify with `go run ./kafka/dlq/` — offset count stays the same (replay does
   not delete DLQ messages; it only re-publishes them).
