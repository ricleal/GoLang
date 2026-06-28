package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	pb "exp/real_time_metrics_agg/gen"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func suppressLogs(t *testing.T) func() {
	t.Helper()
	h := slog.Default().Handler()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	return func() { slog.SetDefault(slog.New(h)) }
}

// echoServer is a minimal gRPC server that echoes every received event back as
// an alert — used to test client stream connect / reconnect.
type echoServer struct {
	pb.UnimplementedMetricsServiceServer
}

func (*echoServer) StreamMetrics(stream pb.MetricsService_StreamMetricsServer) error {
	for {
		evt, err := stream.Recv()
		if err != nil {
			return err
		}
		_ = stream.Send(&pb.Alert{
			MetricName: evt.MetricName,
			Message:    "echo",
		})
	}
}

func startEchoServer(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	pb.RegisterMetricsServiceServer(srv, &echoServer{})
	go srv.Serve(lis)
	t.Cleanup(srv.GracefulStop)
	return lis.Addr().String()
}

// ── resolveServerAddr — pure-logic paths (no external deps) ─────────────────

func TestResolveServerAddr_ExplicitServer(t *testing.T) {
	defer suppressLogs(t)()

	*serverAddr = "myhost:9999"
	*redisAddr = ""

	addr := resolveServerAddr(context.Background())
	if addr != "myhost:9999" {
		t.Errorf("expected myhost:9999, got %s", addr)
	}
}

func TestResolveServerAddr_Fallback(t *testing.T) {
	defer suppressLogs(t)()

	*serverAddr = ""
	*redisAddr = ""

	addr := resolveServerAddr(context.Background())
	if addr != "localhost:50051" {
		t.Errorf("expected fallback localhost:50051, got %s", addr)
	}
}

func TestResolveServerAddr_RedisUnreachable(t *testing.T) {
	defer suppressLogs(t)()

	*serverAddr = ""
	*redisAddr = "127.0.0.1:1" // unlikely to have Redis on port 1

	addr := resolveServerAddr(context.Background())
	if addr != "localhost:50051" {
		t.Errorf("expected fallback localhost:50051, got %s", addr)
	}
}

// ── metric definitions ───────────────────────────────────────────────────────

func TestMetricsDefinitions_HaveValidRanges(t *testing.T) {
	if len(metrics) == 0 {
		t.Fatal("metrics slice is empty")
	}
	for _, m := range metrics {
		if m.name == "" {
			t.Error("metric with empty name")
		}
		if m.baseValue < 0 {
			t.Errorf("metric %q has negative baseValue", m.name)
		}
		if m.jitter <= 0 {
			t.Errorf("metric %q has non-positive jitter", m.name)
		}
	}
}

// ── stream connect / reconnect ───────────────────────────────────────────────

// TestStreamConnect verifies the client can open a stream against a real
// gRPC server, send an event, and receive a response.
func TestStreamConnect(t *testing.T) {
	defer suppressLogs(t)()

	addr := startEchoServer(t)

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewMetricsServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := client.StreamMetrics(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	// Send one event.
	if err := stream.Send(&pb.MetricEvent{
		NodeId: "test", MetricName: "cpu", Value: 50,
	}); err != nil {
		t.Fatalf("send: %v", err)
	}

	// Read the echoed alert.
	alert, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	if alert.MetricName != "cpu" {
		t.Errorf("expected cpu metric, got %s", alert.MetricName)
	}
	_ = stream.CloseSend()
}
