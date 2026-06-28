//go:build integration

package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	tcredismod "github.com/testcontainers/testcontainers-go/modules/redis"
)

func startRedisContainer(t *testing.T) (addr string, rdb *redis.Client, cleanup func()) {
	t.Helper()

	ctx := context.Background()
	container, err := tcredismod.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("redis container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("redis host: %v", err)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("redis port: %v", err)
	}

	addr = fmt.Sprintf("%s:%s", host, port.Port())
	rdb = redis.NewClient(&redis.Options{Addr: addr})

	return addr, rdb, func() {
		rdb.Close()
		container.Terminate(ctx)
	}
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestIntegration_ResolveServerAddr_RedisDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer suppressLogs(t)()

	_ /* redisAddr */, rdb, cleanup := startRedisContainer(t)
	defer cleanup()

	// Register a fake node as a real server would.
	if err := rdb.Set(context.Background(), "metrics:node:test-node", "50099", 30*time.Second).Err(); err != nil {
		t.Fatalf("register node: %v", err)
	}

	*serverAddr = ""
	*redisAddr = rdb.Options().Addr

	addr := resolveServerAddr(context.Background())
	if addr != "localhost:50099" {
		t.Errorf("expected localhost:50099, got %s", addr)
	}
}

func TestIntegration_ResolveServerAddr_RedisNoNodes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer suppressLogs(t)()

	redisAddrStr, _, cleanup := startRedisContainer(t)
	defer cleanup()

	*serverAddr = ""
	*redisAddr = redisAddrStr

	addr := resolveServerAddr(context.Background())
	if addr != "localhost:50051" {
		t.Errorf("expected fallback localhost:50051, got %s", addr)
	}
}

func TestIntegration_ResolveServerAddr_RedisMultipleNodes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer suppressLogs(t)()

	_, rdb, cleanup := startRedisContainer(t)
	defer cleanup()

	// Register multiple nodes.
	nodes := map[string]string{
		"metrics:node:node-a": "50051",
		"metrics:node:node-b": "50052",
		"metrics:node:node-c": "50053",
	}
	for key, port := range nodes {
		if err := rdb.Set(context.Background(), key, port, 30*time.Second).Err(); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}

	*serverAddr = ""
	*redisAddr = rdb.Options().Addr

	// Run resolveServerAddr several times and verify we see different nodes
	// (random selection from the pool).
	seen := make(map[string]bool)
	for i := 0; i < 30; i++ {
		addr := resolveServerAddr(context.Background())
		seen[addr] = true
	}

	// With 3 nodes and 30 attempts, we should have seen all of them.
	for expected := range nodes {
		addr := "localhost:" + nodes[expected]
		if !seen[addr] {
			t.Errorf("never discovered %s (%s) after 30 attempts", expected, addr)
		}
	}
}
