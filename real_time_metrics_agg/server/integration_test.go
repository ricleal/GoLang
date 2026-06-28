//go:build integration

package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"exp/real_time_metrics_agg/aggregator"
	"exp/real_time_metrics_agg/coordinator"
	pb "exp/real_time_metrics_agg/gen"

	"github.com/redis/go-redis/v9"
	tcredismod "github.com/testcontainers/testcontainers-go/modules/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// startRedisContainer starts a Redis 7 container via testcontainers and returns
// the host:port address and a cleanup function.
func startRedisContainer(ctx context.Context) (addr string, terminate func(), _ error) {
	container, err := tcredismod.Run(ctx, "redis:7-alpine")
	if err != nil {
		return "", nil, fmt.Errorf("redis container: %w", err)
	}

	// ConnectionString() returns a redis:// URI, but go-redis wants host:port.
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return "", nil, fmt.Errorf("redis host: %w", err)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		container.Terminate(ctx)
		return "", nil, fmt.Errorf("redis port: %w", err)
	}

	return fmt.Sprintf("%s:%s", host, port.Port()), func() { container.Terminate(ctx) }, nil
}

// startTestServer wires up the aggregator, coordinator, and gRPC server on a
// random port, then returns the server instance and the chosen listener address.
func startTestServer(ctx context.Context, t *testing.T, rdb *redis.Client, nodeID string) (addr string, stop func()) {
	t.Helper()

	agg := aggregator.New()
	coord := coordinator.New(nodeID, rdb, agg.Thresholds())
	srv := newServer(nodeID, rdb, agg, coord)

	// Start background loops (same as main()).
	ctx, cancel := context.WithCancel(ctx)
	go srv.registerNode(ctx)
	go srv.publishAggregates(ctx)
	go coord.RunLeaderElection(ctx, func() {}, func() {})
	go coord.RunClusterEval(ctx)
	go coord.SubscribeAlerts(ctx, func(a coordinator.ClusterAlert) {
		srv.broadcastClusterAlert(&pb.Alert{
			MetricName:  a.MetricName,
			Average:     a.ClusterAvg,
			Threshold:   a.Threshold,
			Message:     a.Message,
			TimestampMs: a.TimestampMs,
		})
	})

	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterMetricsServiceServer(grpcSrv, srv)

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	go grpcSrv.Serve(lis)

	return lis.Addr().String(), func() {
		cancel()
		srv.drainAndClose()
		grpcSrv.GracefulStop()
	}
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestIntegration_EndToEndAlertFires(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// 1. Start Redis.
	redisAddr, cleanup, err := startRedisContainer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	t.Logf("Redis running at %s", redisAddr)

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	// 2. Start the gRPC server wired to this Redis instance.
	serverAddr, stopServer := startTestServer(ctx, t, rdb, "integration-node")
	defer stopServer()
	t.Logf("gRPC server running at %s", serverAddr)

	// 3. Connect a client and open a bidirectional stream.
	conn, err := grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewMetricsServiceClient(conn)
	stream, err := client.StreamMetrics(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	// 4. Send metric events well above the cpu threshold (80).
	//    Send 20 events spaced 50 ms apart so they fill the sliding window.
	const numEvents = 20
	for i := 0; i < numEvents; i++ {
		evt := &pb.MetricEvent{
			NodeId:      "integration-client",
			MetricName:  "cpu",
			Value:       95.0,
			TimestampMs: time.Now().UnixMilli(),
			Seq:         int64(i + 1),
		}
		if err := stream.Send(evt); err != nil {
			t.Fatalf("send event %d: %v", i, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Logf("sent %d cpu events at 95.0 (threshold=80)", numEvents)

	// 5. Read alerts from the stream (the server's alert ticker fires every 1 s).
	//    We expect at least one LOCAL ALERT within a reasonable timeout.
	done := make(chan *pb.Alert, 1)
	go func() {
		for {
			alert, err := stream.Recv()
			if err != nil {
				return
			}
			done <- alert
		}
	}()

	select {
	case alert := <-done:
		if alert.MetricName != "cpu" {
			t.Errorf("expected alert for 'cpu', got %q", alert.MetricName)
		}
		if alert.Average <= alert.Threshold {
			t.Errorf("alert average %.2f should exceed threshold %.2f", alert.Average, alert.Threshold)
		}
		t.Logf("ALERT received: %s (avg=%.2f, threshold=%.2f)", alert.Message, alert.Average, alert.Threshold)

	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for alert — no alert received within 15 s")
	}
}

func TestIntegration_Deduplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	redisAddr, cleanup, err := startRedisContainer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	serverAddr, stopServer := startTestServer(ctx, t, rdb, "dedup-node")
	defer stopServer()

	conn, err := grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewMetricsServiceClient(conn)
	stream, err := client.StreamMetrics(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	// Send a value with seq=1.
	_ = stream.Send(&pb.MetricEvent{
		NodeId: "dedup-client", MetricName: "cpu", Value: 50,
		TimestampMs: time.Now().UnixMilli(), Seq: 1,
	})
	// Re-send the same seq — should be deduplicated (not recorded).
	_ = stream.Send(&pb.MetricEvent{
		NodeId: "dedup-client", MetricName: "cpu", Value: 9999,
		TimestampMs: time.Now().UnixMilli(), Seq: 1,
	})
	// Send a new seq.
	_ = stream.Send(&pb.MetricEvent{
		NodeId: "dedup-client", MetricName: "cpu", Value: 50,
		TimestampMs: time.Now().UnixMilli(), Seq: 2,
	})

	// Now send high values so an alert fires — if dedup failed the average
	// would be heavily skewed by the 9999.
	for i := 0; i < 15; i++ {
		_ = stream.Send(&pb.MetricEvent{
			NodeId: "dedup-client", MetricName: "cpu", Value: 90,
			TimestampMs: time.Now().UnixMilli(), Seq: int64(i + 10),
		})
		time.Sleep(50 * time.Millisecond)
	}

	done := make(chan *pb.Alert, 1)
	go func() {
		alert, _ := stream.Recv()
		if alert != nil {
			done <- alert
		}
	}()

	select {
	case alert := <-done:
		// The average should be ~(50+50+15*90)/17 ≈ 85.3, NOT skewed by 9999.
		if alert.Average > 200 {
			t.Errorf("deduplication likely failed — average %.2f suggests 9999 was counted", alert.Average)
		}
		t.Logf("Dedup check passed: avg=%.2f (reasonable range)", alert.Average)

	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for alert in dedup test")
	}
}

func TestIntegration_AlertCooldown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	redisAddr, cleanup, err := startRedisContainer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	serverAddr, stopServer := startTestServer(ctx, t, rdb, "cooldown-node")
	defer stopServer()

	conn, err := grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewMetricsServiceClient(conn)
	stream, err := client.StreamMetrics(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	// Fill window with high values.
	for i := 0; i < 20; i++ {
		_ = stream.Send(&pb.MetricEvent{
			NodeId: "cooldown-client", MetricName: "cpu", Value: 95,
			TimestampMs: time.Now().UnixMilli(), Seq: int64(i + 1),
		})
		time.Sleep(50 * time.Millisecond)
	}

	// Collect alerts for 3 s.
	//
	// The client receives BOTH local alerts (from the aggregator's alertTicker,
	// subject to the 5 s cooldown) AND cluster alerts (from the leader via
	// Redis Pub/Sub, fired every 1 s with no cooldown).
	//
	// With cooldown working:  ~1 local alert + ~3 cluster alerts = ~4 total
	// Without cooldown:       ~3 local alerts + ~3 cluster alerts = ~6 total
	// We check that the total is ≤ 5 to confirm cooldown is suppressing the
	// local side.
	const maxExpected = 5
	alertCount := 0
	overLimit := make(chan struct{}, 1)
	go func() {
		for {
			_, err := stream.Recv()
			if err != nil {
				return
			}
			alertCount++
			if alertCount > maxExpected {
				overLimit <- struct{}{}
			}
		}
	}()

	select {
	case <-overLimit:
		t.Fatalf("received %d alerts in 3 s — exceeds max %d; cooldown likely not working",
			alertCount, maxExpected)
	case <-time.After(3 * time.Second):
		t.Logf("Alert count after 3 s: %d (max expected with cooldown: %d)",
			alertCount, maxExpected)
		if alertCount > maxExpected {
			t.Errorf("cooldown may not be working: %d alerts in 3 s (max expected %d)",
				alertCount, maxExpected)
		}
	}
}
