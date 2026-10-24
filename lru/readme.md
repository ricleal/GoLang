# LRU Cache

This directory contains two LRU cache implementations:

- `New` uses a map and Go's `container/list` package.
- `NewEnhanced` uses a map plus a custom doubly linked list.

Both support adding or updating an entry, retrieving an entry, promoting accessed entries to most recently used, and evicting the least recently used entry when full.

## Run the tests

From the repository root, run:

```sh
go test ./lru
```

The tests cover value updates, recency updates, eviction order, and zero or negative capacity. To run the tests from this directory instead, use:

```sh
go test
```

## Compare performance

From the repository root, run:

```sh
go test ./lru -run '^$' -bench '^BenchmarkLRU$' -benchmem -count=3
```

The benchmark preloads each cache with 1024 entries. Each timed iteration adds an entry and retrieves an older entry, exercising insertion, eviction, lookup, and recency changes. Setup is outside the timed section. `ns/op` is the time for that pair of operations, not for a single method call.

On one Linux run with an Intel Xeon E5-2699 v4, three samples were:

| Implementation | Time per Add/Get pair | Bytes per pair | Allocations per pair |
| --- | ---: | ---: | ---: |
| `container/list` | 767-797 ns | 95 B | 3 |
| Map + custom doubly linked list | 582-624 ns | 55 B | 1 |

For this workload, the enhanced implementation was about 22% faster and used about 42% fewer bytes per iteration. Results depend on the Go version, hardware, and workload; run the benchmark on your target environment before drawing conclusions.

Both designs have expected O(1) map lookup, insertion, recency updates, and eviction. The enhanced version removes `container/list` element overhead, but maintains its own link pointers and list invariants. Both use O(capacity) memory, and each new entry still requires an allocation.
