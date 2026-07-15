# Distributed DB Failure Testing — Cassandra + Go

A hands-on exercise for exploring distributed database concepts, fault
tolerance, and tunable consistency using a 3-node Apache Cassandra cluster
and a small Go client.

## What's here

| File | Purpose |
|---|---|
| [docker-compose.yml](docker-compose.yml) | 3-node `TestCluster` Cassandra cluster (`cassandra-1`, `cassandra-2`, `cassandra-3`), each with its `9042` port published to a distinct host port (`9042`/`9043`/`9044`) |
| [docker-compose.scale.yml](docker-compose.scale.yml) | Override file that adds a 4th node (`cassandra-4`) for Experiment 3 |
| [main.go](main.go) | Go client that continuously writes/reads a row against the cluster |
| [network-partition.sh](network-partition.sh) | Simulates a network partition (`docker network disconnect`/`connect`) for Experiment 4, as opposed to a hard `docker stop` |

The Go client uses [gocql](https://github.com/gocql/gocql) and lives in the
root Go module (`exp`), so it can be run directly with `go run` from the
repository root.

## 1. Start the cluster

From the `distributed_db` directory:

```bash
docker compose up -d
```

Cassandra takes a while to boot (JVM start + gossip handshake). `cassandra-2`
and `cassandra-3` also wait out a fixed startup delay (20s/120s) before
starting the Cassandra process itself, on top of waiting for the previous
node to be healthy — vnode bootstrapping needs gossip to fully converge
between joins, or two nodes can race the token allocator and crash with a
"Bootstrap Token collision" / "address already exists" error. Expect the
full cluster to take a few minutes to become healthy. Wait for all three
containers to become healthy:

```bash
docker compose ps
```

Then verify the cluster sees all 3 nodes as `UN` (Up/Normal):

```bash
docker exec -it cassandra-1 nodetool status
```

Expected output (addresses will differ):

```
Datacenter: datacenter1
=======================
Status=Up/Down
|/ State=Normal/Leaving/Joining/Moving
--  Address      Load        Tokens  Owns (effective)  Host ID   Rack
UN  172.x.x.2    ...         16      ...                ...       rack1
UN  172.x.x.3    ...         16      ...                ...       rack1
UN  172.x.x.4    ...         16      ...                ...       rack1
```

If any node shows `DN` (Down/Normal), give it more time — first boot can take
1-2 minutes per node while gossip converges.

> **Troubleshooting a crashed node on first boot:** if `docker compose ps -a`
> shows `cassandra-2` or `cassandra-3` as `Exited`, check
> `docker logs cassandra-3` for `Bootstrap Token collision` or `already
> exists, cancelling join`. These are gossip-timing races during vnode
> bootstrap. Once a node crashes mid-join, its data volume can be left with
> inconsistent `system.local`/`system.peers` state that keeps failing on
> plain restarts. The reliable fix is a full reset:
> ```bash
> docker compose down -v
> docker compose up -d
> ```
> `-v` removes the named volumes, so every node bootstraps from scratch.

## 2. Run the Go client

From the repository root:

```bash
go run ./distributed_db
```

On first run it creates the `test_ks` keyspace (`NetworkTopologyStrategy`,
`datacenter1: 3`) and the `users` table, then loops forever:

- generates an incrementing `id`
- `INSERT`s a row, printing `[WRITE SUCCESS]` or `[WRITE ERROR] <message>`
- `SELECT`s the row back, printing `[READ SUCCESS]` or `[READ ERROR] <message>`
- sleeps 1 second

The consistency level used for the read/write loop is controlled by a single
constant at the top of `main.go`:

```go
const consistencyLevel = gocql.Quorum // <-- CHANGE ME: gocql.Quorum or gocql.One
```

Stop the client at any time with `Ctrl+C`.

> **Multiple contact points:** the client connects using all three published
> ports (`127.0.0.1:9042`, `:9043`, `:9044`), one per node, with
> `cluster.DisableInitialHostLookup = true`. This matters because Cassandra
> advertises peers using their internal Docker network addresses, which
> aren't reachable from the host — without this, discovering peers via
> `system.peers` would just produce unreachable hosts. Using three
> independent, explicit contact points also means the client survives the
> loss of **any single node**, including `cassandra-1` — not just
> `cassandra-2`/`cassandra-3`.

## Experiment 1 — Quorum failure

1. Make sure `consistencyLevel = gocql.Quorum` in [main.go](main.go) (the default).
2. Start the client: `go run ./distributed_db`. Confirm you see steady
   `[WRITE SUCCESS]` / `[READ SUCCESS]` lines.
3. Kill one node:
   ```bash
   docker stop cassandra-2
   ```
   Keep watching the client output — it should **keep succeeding**. With
   `RF=3` and `QUORUM`, a query needs acks from `⌈3/2⌉ = 2` replicas, and
   with only 1 node down, 2 replicas are still reachable. This also holds if
   you kill `cassandra-1` instead, since the client has an independent
   contact point for every node (see the note above) — try it.
4. Kill a second node:
   ```bash
   docker stop cassandra-3
   ```
   Now only `cassandra-1` is up. Watch the client output switch to
   `[WRITE ERROR]` / `[READ ERROR]`, typically something like
   `Cannot achieve consistency level QUORUM` or `Not enough replicas available`.

**Why it fails:** with `RF=3`, `QUORUM` requires 2 of 3 replicas to
acknowledge every read/write. With 2 of 3 nodes down, only 1 replica is
reachable — the majority is lost, so Cassandra refuses the operation rather
than risk an inconsistent result. This is the classic CAP theorem trade-off:
the cluster chooses **consistency** over **availability**.

## Experiment 2 — Tunable consistency

1. Leave `cassandra-2` and `cassandra-3` stopped (only `cassandra-1` running).
2. Stop the Go client (`Ctrl+C`) and edit [main.go](main.go):
   ```go
   const consistencyLevel = gocql.One // changed from gocql.Quorum
   ```
3. Restart the client: `go run ./distributed_db`.
4. Observe that `[WRITE SUCCESS]` / `[READ SUCCESS]` resume, even though 2 of
   3 nodes are still down.

**Why it works now:** `ONE` only requires a single replica to acknowledge
the operation. As long as `cassandra-1` is reachable and holds a replica of
the row, the operation succeeds. This demonstrates the CAP theorem trade-off
in the other direction: the cluster now favors **availability** over strict
consistency — reads may return stale data, and if a client had written to a
now-dead replica that never propagated, that write could be invisible until
the node comes back and repairs.

## Experiment 3 — Scale up without downtime

While the Go client is still running (with `consistencyLevel = gocql.One` or
`gocql.Quorum`, either works once nodes are back):

1. Bring the two stopped nodes back up:
   ```bash
   docker start cassandra-2 cassandra-3
   ```
   Watch `nodetool status` on `cassandra-1` until both show `UN` again.
   If you were running with `Quorum`, the client output should recover as
   soon as 2 of 3 nodes are reachable.

2. Add a 4th node to the cluster **without stopping anything** — Cassandra
   nodes can join a running cluster, and existing data is automatically
   rebalanced across the new node's token ranges:
   ```bash
   docker compose -f docker-compose.yml -f docker-compose.scale.yml up -d cassandra-4
   ```
3. Watch the new node join and the cluster rebalance load in real time:
   ```bash
   watch -n 2 docker exec cassandra-1 nodetool status
   ```
   You'll see `cassandra-4` appear as `UN` (Up/Normal), and the `Load`/`Owns`
   percentages shift across all 4 nodes as `cassandra-4` takes over part of
   the token ring — all while the Go client keeps writing and reading rows
   with zero downtime.
4. Optionally, run a repair so existing data is fully streamed to the new
   node's owned ranges:
   ```bash
   docker exec cassandra-4 nodetool repair -full
   ```

This demonstrates horizontal scalability: capacity can be added to a live
Cassandra cluster with no downtime, and the cluster automatically
redistributes data and ownership among all nodes.

## Experiment 4 — Network partition vs. hard failure

`docker stop` simulates a crash: the process dies immediately and gossip
marks the node down right away. Real-world outages are more often a
**network partition** — the process keeps running and keeps its state, but
can't reach (or be reached by) the rest of the cluster. [network-partition.sh](network-partition.sh)
simulates this using `docker network disconnect`/`connect` instead of
stopping the container.

1. With all nodes up and the Go client running (`consistencyLevel = gocql.Quorum`),
   partition one node away from the cluster network:
   ```bash
   ./network-partition.sh isolate cassandra-3
   ```
2. Watch `cassandra-1`'s view of the ring — `cassandra-3` will eventually flip
   to `DN` once gossip's failure detector (phi accrual) times it out, which
   takes longer than an immediate `docker stop` would:
   ```bash
   watch -n 2 docker exec cassandra-1 nodetool status
   ```
   Meanwhile the client keeps succeeding (only 1 of 3 replicas is
   unreachable, same as Experiment 1's single-node-down case).
3. Check the partitioned node's own perspective — from inside `cassandra-3`,
   the *other* nodes look down, even though `cassandra-3` itself is perfectly
   healthy:
   ```bash
   docker exec cassandra-3 nodetool status
   ```
4. Heal the partition:
   ```bash
   ./network-partition.sh heal cassandra-3
   ```
   `cassandra-3` rejoins gossip **without a restart or replay of the join
   sequence** (unlike Experiment 3's brand-new node) — it already has its
   token ownership and data, it just resumes talking to its peers. Watch it
   flip back to `UN`.
5. Use `./network-partition.sh status` at any point to see which containers
   are currently attached to the cluster network.

**Why this matters:** a partitioned-but-alive node is a fundamentally
different failure mode than a dead one. It can keep accepting client
connections and even serve stale local reads (if a client's contact point
happens to be that node), it holds onto in-flight state instead of releasing
it immediately, and its reconnection path (resume gossip) is cheaper and
faster than a dead node's (full restart, commit log replay, hint replay).
Systems that only ever get tested against `kill -9`/`docker stop` can hide
partition-specific bugs — e.g. split-brain-style double writes accepted by
both sides of a partition, or a client stuck waiting on a connection to a
node that's alive but unreachable rather than one that fails fast.

## Cleanup

```bash
docker compose -f docker-compose.yml -f docker-compose.scale.yml down -v
```

The `-v` flag removes the named volumes so the next run starts from a clean
cluster.
