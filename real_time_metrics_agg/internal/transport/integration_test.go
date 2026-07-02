package transport

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ricleal/GoLang/real_time_metrics_agg/internal/aggregator"
	"github.com/ricleal/GoLang/real_time_metrics_agg/internal/alert"
	"github.com/ricleal/GoLang/real_time_metrics_agg/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// testServer holds an in-memory gRPC test server.
type testServer struct {
	lis        *bufconn.Listener
	grpcServer *grpc.Server
	agg        *aggregator.SlidingWindowAggregator
	alertEng   *alert.Engine
	addr       string
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	agg := aggregator.NewSlidingWindowAggregator(10*time.Second, 1*time.Second)
	alertEng := alert.NewEngine(100)
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	metricsServer := NewMetricsServer(agg, alertEng, nil, "test-node", logger, 10)

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(bufSize),
		grpc.MaxSendMsgSize(bufSize),
	)
	pb.RegisterMetricsAggregatorServer(grpcServer, metricsServer)

	lis := bufconn.Listen(bufSize)

	ts := &testServer{
		lis:        lis,
		grpcServer: grpcServer,
		agg:        agg,
		alertEng:   alertEng,
		addr:       fmt.Sprintf("bufnet-%d", time.Now().UnixNano()),
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("gRPC serve error (expected on stop): %v", err)
		}
	}()

	return ts
}

func (ts *testServer) stop() {
	ts.grpcServer.GracefulStop()
	ts.lis.Close()
}

func bufDialer(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		return lis.Dial()
	}
}

func newTestClient(t *testing.T, lis *bufconn.Listener) (pb.MetricsAggregatorClient, *grpc.ClientConn) {
	t.Helper()

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(bufDialer(lis)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("failed to dial test server: %v", err)
	}
	return pb.NewMetricsAggregatorClient(conn), conn
}

func TestIntegration_HealthCheck(t *testing.T) {
	ts := newTestServer(t)
	defer ts.stop()

	client, conn := newTestClient(t, ts.lis)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.HealthCheck(ctx, &pb.HealthCheckRequest{Service: "metrics-aggregator"})
	if err != nil {
		t.Fatalf("health check: %v", err)
	}
	if resp.Status != "SERVING" {
		t.Fatalf("expected SERVING, got %s", resp.Status)
	}
	if resp.Leader {
		t.Fatal("expected leader=false (no raft)")
	}
}

func TestIntegration_SetThreshold(t *testing.T) {
	ts := newTestServer(t)
	defer ts.stop()

	client, conn := newTestClient(t, ts.lis)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.SetThreshold(ctx, &pb.SetThresholdRequest{
		MetricName: "cpu_usage",
		Warn:       50.0,
		Critical:   80.0,
	})
	if err != nil {
		t.Fatalf("set threshold: %v", err)
	}
	if !resp.Ok {
		t.Fatal("expected ok=true")
	}

	// Verify threshold was stored
	th := ts.alertEng.GetThreshold("cpu_usage")
	if th == nil {
		t.Fatal("expected threshold to be set")
	}
	if th.Warn != 50.0 {
		t.Fatalf("expected warn 50.0, got %f", th.Warn)
	}
}

func TestIntegration_StreamMetrics_ReceiveAggregates(t *testing.T) {
	ts := newTestServer(t)
	defer ts.stop()

	client, conn := newTestClient(t, ts.lis)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.StreamMetrics(ctx)
	if err != nil {
		t.Fatalf("stream metrics: %v", err)
	}

	// Send a few metrics and wait for server to process them.
	for i := 0; i < 10; i++ {
		metric := &pb.Metric{
			Name:      "cpu_usage",
			Value:     float64(50 + i*10),
			Timestamp: time.Now().UnixNano(),
			ClientId:  "test-client",
		}
		if err := stream.Send(metric); err != nil {
			t.Fatalf("send metric: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Wait for an aggregate response from the server's 2s ticker.
	aggCh := make(chan *pb.AggregatedMetric, 1)
	go func() {
		agg, recvErr := stream.Recv()
		if recvErr != nil {
			t.Logf("recv aggregate: %v", recvErr)
			return
		}
		aggCh <- agg
	}()

	select {
	case agg := <-aggCh:
		if agg.MetricName != "cpu_usage" {
			t.Fatalf("expected metric 'cpu_usage', got '%s'", agg.MetricName)
		}
		if agg.Count == 0 {
			t.Fatalf("expected count > 0, got %d", agg.Count)
		}
		if agg.Avg <= 0 {
			t.Fatalf("expected avg > 0, got %f", agg.Avg)
		}
		t.Logf("received aggregate: %s avg=%.2f count=%d p95=%.2f p99=%.2f",
			agg.MetricName, agg.Avg, agg.Count, agg.P95, agg.P99)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for aggregate")
	}

	stream.CloseSend()
}

func TestIntegration_StreamMetrics_AlertTriggered(t *testing.T) {
	ts := newTestServer(t)
	defer ts.stop()

	// Set a low threshold via the server directly
	ts.alertEng.SetThreshold("cpu_usage", 10.0, 50.0)

	client, conn := newTestClient(t, ts.lis)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.StreamMetrics(ctx)
	if err != nil {
		t.Fatalf("stream metrics: %v", err)
	}

	// Send a high metric that should trigger an alert
	metric := &pb.Metric{
		Name:      "cpu_usage",
		Value:     95.0,
		Timestamp: time.Now().UnixNano(),
		ClientId:  "test-client",
	}
	if err := stream.Send(metric); err != nil {
		t.Fatalf("send metric: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// The alert should have been logged in the engine
	alerts := ts.alertEng.RecentAlerts(10)
	found := false
	for _, a := range alerts {
		if a.MetricName == "cpu_usage" {
			found = true
			t.Logf("alert triggered: %s (severity=%s)", a.Message, a.Severity)
			break
		}
	}
	if !found {
		t.Fatal("expected alert for cpu_usage to be triggered")
	}

	stream.CloseSend()
}

func TestIntegration_ConcurrentStreams(t *testing.T) {
	ts := newTestServer(t)
	defer ts.stop()

	client, conn := newTestClient(t, ts.lis)
	defer conn.Close()

	numStreams := 3
	metricsPerStream := 10

	var wg sync.WaitGroup
	for i := 0; i < numStreams; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			stream, err := client.StreamMetrics(ctx)
			if err != nil {
				t.Errorf("stream %d: %v", id, err)
				return
			}

			clientID := fmt.Sprintf("client-%d", id)
			for j := 0; j < metricsPerStream; j++ {
				metric := &pb.Metric{
					Name:      fmt.Sprintf("metric_%d", id),
					Value:     float64(j * 10),
					Timestamp: time.Now().UnixNano(),
					ClientId:  clientID,
				}
				if err := stream.Send(metric); err != nil {
					t.Errorf("stream %d send: %v", id, err)
					return
				}
				time.Sleep(5 * time.Millisecond)
			}

			// Try to receive
			aggCh := make(chan *pb.AggregatedMetric, 1)
			go func() {
				agg, recvErr := stream.Recv()
				if recvErr == nil {
					aggCh <- agg
				}
			}()
			select {
			case <-aggCh:
			case <-time.After(3 * time.Second):
			}

			stream.CloseSend()
		}(i)
	}
	wg.Wait()

	// Verify metrics were aggregated
	snaps := ts.agg.SnapshotAll()
	if len(snaps) < 1 {
		t.Fatal("expected at least one metric snapshot after concurrent streams")
	}
	t.Logf("got %d metric aggregates from %d concurrent streams", len(snaps), numStreams)
}

func TestIntegration_JoinCluster_NoRaft(t *testing.T) {
	ts := newTestServer(t)
	defer ts.stop()

	client, conn := newTestClient(t, ts.lis)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// RaftNode is nil in test server, so JoinCluster should fail
	_, err := client.JoinCluster(ctx, &pb.JoinClusterRequest{
		NodeId:      "node2",
		RaftAddress: "127.0.0.1:50071",
	})
	if err == nil {
		t.Fatal("expected error when joining without raft initialized")
	}
}

func TestIntegration_StreamAndThresholdEndToEnd(t *testing.T) {
	ts := newTestServer(t)
	defer ts.stop()

	client, conn := newTestClient(t, ts.lis)
	defer conn.Close()

	// Set threshold via gRPC
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	setResp, err := client.SetThreshold(ctx, &pb.SetThresholdRequest{
		MetricName: "latency_ms",
		Warn:       100.0,
		Critical:   200.0,
	})
	cancel()
	if err != nil {
		t.Fatalf("set threshold: %v", err)
	}
	if !setResp.Ok {
		t.Fatal("expected ok=true")
	}

	// Open stream
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.StreamMetrics(ctx)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	// Send a critical-level metric (250 > critical=200).
	highMetric := &pb.Metric{
		Name:      "latency_ms",
		Value:     250.0,
		Timestamp: time.Now().UnixNano(),
		ClientId:  "e2e-client",
	}
	if err := stream.Send(highMetric); err != nil {
		t.Fatalf("send high metric: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Should have triggered a CRITICAL alert (avg=250 >= critical=200)
	alerts := ts.alertEng.RecentAlerts(10)
	hasCritical := false
	for _, a := range alerts {
		if a.MetricName == "latency_ms" && a.Severity == "CRITICAL" {
			hasCritical = true
			t.Logf("e2e alert: %s", a.Message)
		}
	}
	if !hasCritical {
		t.Fatal("expected a CRITICAL alert for latency_ms with avg=250 >= critical=200")
	}

	stream.CloseSend()
}
