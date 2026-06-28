//go:build integration

package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	tcredismod "github.com/testcontainers/testcontainers-go/modules/redis"
)

func suppressLogs(t *testing.T) func() {
	t.Helper()
	h := slog.Default().Handler()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	return func() { slog.SetDefault(slog.New(h)) }
}

func startRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()

	ctx := context.Background()
	container, err := tcredismod.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("redis container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("host: %v", err)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("port: %v", err)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	return rdb, func() {
		rdb.Close()
		container.Terminate(ctx)
	}
}

// ── Publish + read back ──────────────────────────────────────────────────────

func TestIntegration_PublishAndReadAggregates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer suppressLogs(t)()

	rdb, cleanup := startRedis(t)
	defer cleanup()

	c := New("node-publish", rdb, nil)
	ctx := context.Background()

	averages := map[string]float64{"cpu": 85.5, "network": 500_000}
	c.PublishLocalAggregates(ctx, averages)

	// Read directly from Redis and verify the stored JSON.
	data, err := rdb.Get(ctx, "metrics:agg:node-publish").Bytes()
	if err != nil {
		t.Fatalf("get from redis: %v", err)
	}

	var na NodeAggregates
	if err := json.Unmarshal(data, &na); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if na.NodeID != "node-publish" {
		t.Errorf("expected NodeID 'node-publish', got %q", na.NodeID)
	}
	if na.Metrics["cpu"] != 85.5 {
		t.Errorf("expected cpu=85.5, got %f", na.Metrics["cpu"])
	}
	if na.Metrics["network"] != 500_000 {
		t.Errorf("expected network=500000, got %f", na.Metrics["network"])
	}
}

// ── Leader election ──────────────────────────────────────────────────────────

func TestIntegration_LeaderAcquireAndVerify(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer suppressLogs(t)()

	rdb, cleanup := startRedis(t)
	defer cleanup()

	c := New("node-a", rdb, nil)
	ctx := context.Background()

	if !c.tryAcquireOrRenewLeader(ctx) {
		t.Fatal("expected to acquire leader lock on empty Redis")
	}
	if !c.isLeader(ctx) {
		t.Error("isLeader() should return true after acquiring")
	}
}

func TestIntegration_LeaderElection_OneWins(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer suppressLogs(t)()

	rdb, cleanup := startRedis(t)
	defer cleanup()

	c1 := New("node-a", rdb, nil)
	c2 := New("node-b", rdb, nil)
	ctx := context.Background()

	acquired1 := c1.tryAcquireOrRenewLeader(ctx)
	acquired2 := c2.tryAcquireOrRenewLeader(ctx)

	if !acquired1 {
		t.Error("node-a should have acquired the lock (first attempt)")
	}
	if acquired2 {
		t.Error("node-b should NOT have acquired the lock (already held)")
	}
	if !c1.isLeader(ctx) {
		t.Error("node-a should still be leader")
	}
	if c2.isLeader(ctx) {
		t.Error("node-b should not be leader")
	}
}

func TestIntegration_LeaderRenewal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer suppressLogs(t)()

	rdb, cleanup := startRedis(t)
	defer cleanup()

	c := New("node-renew", rdb, nil)
	ctx := context.Background()

	if !c.tryAcquireOrRenewLeader(ctx) {
		t.Fatal("initial acquire failed")
	}

	// Renew several times — should succeed each time.
	for i := 0; i < 3; i++ {
		if !c.tryAcquireOrRenewLeader(ctx) {
			t.Errorf("renewal attempt %d failed", i+1)
		}
		if !c.isLeader(ctx) {
			t.Errorf("not leader after renewal %d", i+1)
		}
	}
}

func TestIntegration_LeaderTransferOnKeyDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer suppressLogs(t)()

	rdb, cleanup := startRedis(t)
	defer cleanup()

	c1 := New("node-a", rdb, nil)
	c2 := New("node-b", rdb, nil)
	ctx := context.Background()

	c1.tryAcquireOrRenewLeader(ctx)

	// Simulate leader crash by deleting the lock key.
	rdb.Del(ctx, leaderKey)

	// Now node-b should be able to acquire.
	if !c2.tryAcquireOrRenewLeader(ctx) {
		t.Fatal("node-b should acquire after lock key deleted")
	}
	if !c2.isLeader(ctx) {
		t.Error("node-b should be leader after acquisition")
	}
	if c1.isLeader(ctx) {
		t.Error("node-a should not be leader after lock was taken")
	}
}

// ── evalAndPublish ───────────────────────────────────────────────────────────

func TestIntegration_EvalAndPublishAlert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer suppressLogs(t)()

	rdb, cleanup := startRedis(t)
	defer cleanup()

	thresholds := map[string]float64{"cpu": 80, "default": 100}
	c := New("eval-node", rdb, thresholds)
	ctx := context.Background()

	// Pre-populate two nodes' aggregate keys (above cpu threshold).
	for _, node := range []string{"node-a", "node-b"} {
		na := NodeAggregates{
			NodeID:  node,
			Metrics: map[string]float64{"cpu": 90.0},
		}
		data, _ := json.Marshal(na)
		if err := rdb.Set(ctx, "metrics:agg:"+node, data, 30*time.Second).Err(); err != nil {
			t.Fatalf("seed %s: %v", node, err)
		}
	}

	// Subscribe to alerts BEFORE triggering eval.
	sub := rdb.Subscribe(ctx, AlertsChannel)
	defer sub.Close()
	// Give the subscription time to propagate.
	time.Sleep(200 * time.Millisecond)

	// Trigger cluster evaluation.
	c.evalAndPublish(ctx)

	// Read the alert from Pub/Sub.
	select {
	case msg := <-sub.Channel():
		var alert ClusterAlert
		if err := json.Unmarshal([]byte(msg.Payload), &alert); err != nil {
			t.Fatalf("unmarshal alert: %v", err)
		}
		if alert.MetricName != "cpu" {
			t.Errorf("expected MetricName 'cpu', got %q", alert.MetricName)
		}
		if alert.ClusterAvg != 90.0 {
			t.Errorf("expected ClusterAvg 90.0, got %f", alert.ClusterAvg)
		}
		if alert.Threshold != 80.0 {
			t.Errorf("expected Threshold 80.0, got %f", alert.Threshold)
		}
		if alert.Message == "" {
			t.Error("expected non-empty Message")
		}
		if alert.TimestampMs == 0 {
			t.Error("expected non-zero TimestampMs")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for cluster alert")
	}
}

func TestIntegration_EvalAndPublish_NoAlertBelowThreshold(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer suppressLogs(t)()

	rdb, cleanup := startRedis(t)
	defer cleanup()

	thresholds := map[string]float64{"cpu": 80, "default": 100}
	c := New("eval-node", rdb, thresholds)
	ctx := context.Background()

	// Pre-populate with values BELOW threshold.
	na := NodeAggregates{
		NodeID:  "node-a",
		Metrics: map[string]float64{"cpu": 30.0},
	}
	data, _ := json.Marshal(na)
	rdb.Set(ctx, "metrics:agg:node-a", data, 30*time.Second)

	sub := rdb.Subscribe(ctx, AlertsChannel)
	defer sub.Close()
	time.Sleep(200 * time.Millisecond)

	c.evalAndPublish(ctx)

	// Should NOT receive an alert.
	select {
	case msg := <-sub.Channel():
		t.Fatalf("unexpected alert received: %s", msg.Payload)
	case <-time.After(1 * time.Second):
		// Expected — no alert should fire.
	}
}

// ── SubscribeAlerts ──────────────────────────────────────────────────────────

func TestIntegration_SubscribeAlerts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer suppressLogs(t)()

	rdb, cleanup := startRedis(t)
	defer cleanup()

	c := New("sub-node", rdb, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var received []ClusterAlert

	// Start subscriber in background.
	go c.SubscribeAlerts(ctx, func(a ClusterAlert) {
		mu.Lock()
		received = append(received, a)
		mu.Unlock()
	})

	// Give subscription time to establish.
	time.Sleep(300 * time.Millisecond)

	// Publish two alerts directly to the channel.
	for i := 0; i < 2; i++ {
		alert := ClusterAlert{
			MetricName:  "cpu",
			ClusterAvg:  85.0,
			Threshold:   80.0,
			Message:     "test alert",
			TimestampMs: time.Now().UnixMilli(),
		}
		data, _ := json.Marshal(alert)
		if err := rdb.Publish(ctx, AlertsChannel, string(data)).Err(); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}

	// Wait for both messages.
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	count := len(received)
	mu.Unlock()

	if count != 2 {
		t.Errorf("expected 2 alerts, got %d", count)
	}
}
