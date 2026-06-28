package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	pb "exp/real_time_metrics_agg/gen"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	serverAddr = flag.String("server", "", "server address (host:port); if empty, uses --redis for discovery")
	nodeID     = flag.String("node-id", "client-1", "client node identifier")
	interval   = flag.Duration("interval", 100*time.Millisecond, "how often to send a metric event")
	redisAddr  = flag.String("redis", "", "Redis address for node discovery (used when --server is empty)")
)

var seqCounter atomic.Int64

type metricSpec struct {
	name      string
	baseValue float64
	jitter    float64
}

// Base values are intentionally set above the server thresholds so the
// sliding-window average crosses them and alerts fire within a few seconds.
//
//	cpu       threshold 80   → mean ~87.5  (range 75–100)
//	network   threshold 1 M  → mean ~1.15M (range 1M–1.3M)
//	requests  threshold 100  → mean ~110   (range 90–130)
var metrics = []metricSpec{
	{name: "cpu", baseValue: 75.0, jitter: 25.0},
	{name: "network", baseValue: 1_000_000, jitter: 300_000},
	{name: "requests", baseValue: 90.0, jitter: 40.0},
}

func streamMetrics(ctx context.Context, client pb.MetricsServiceClient) error {
	stream, err := client.StreamMetrics(ctx)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}

	recvDone := make(chan error, 1)
	go func() {
		for {
			alert, err := stream.Recv()
			if err != nil {
				if err != io.EOF && ctx.Err() == nil {
					slog.Warn("recv error", "err", err)
				}
				recvDone <- err
				return
			}
			slog.Warn("ALERT",
				"metric", alert.MetricName,
				"avg", fmt.Sprintf("%.2f", alert.Average),
				"threshold", fmt.Sprintf("%.2f", alert.Threshold),
				"msg", alert.Message,
			)
		}
	}()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = stream.CloseSend()
			return nil

		case err := <-recvDone:
			return err

		case <-ticker.C:
			spec := metrics[rand.Intn(len(metrics))]
			value := spec.baseValue + rand.Float64()*spec.jitter
			seq := seqCounter.Add(1)

			evt := &pb.MetricEvent{
				NodeId:      *nodeID,
				MetricName:  spec.name,
				Value:       value,
				TimestampMs: time.Now().UnixMilli(),
				Seq:         seq,
			}
			if err := stream.Send(evt); err != nil {
				return fmt.Errorf("send: %w", err)
			}
			slog.Debug("sent",
				"metric", spec.name,
				"value", fmt.Sprintf("%.2f", value),
				"seq", seq,
			)
		}
	}
}

// resolveServerAddr picks a server address using Redis-based service discovery
// when --server is not set explicitly. Falls back to localhost:50051.
func resolveServerAddr(ctx context.Context) string {
	if *serverAddr != "" {
		return *serverAddr
	}
	if *redisAddr == "" {
		slog.Warn("neither --server nor --redis provided; falling back to localhost:50051")
		return "localhost:50051"
	}

	rdb := redis.NewClient(&redis.Options{Addr: *redisAddr})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Warn("redis unreachable, falling back to localhost:50051", "err", err)
		return "localhost:50051"
	}

	keys, err := rdb.Keys(ctx, "metrics:node:*").Result()
	if err != nil || len(keys) == 0 {
		slog.Warn("no nodes registered in Redis, falling back to localhost:50051")
		return "localhost:50051"
	}

	// Pick a random node.
	key := keys[rand.Intn(len(keys))]
	port, err := rdb.Get(ctx, key).Result()
	if err != nil {
		slog.Warn("failed to read node port from Redis, falling back to localhost:50051", "key", key)
		return "localhost:50051"
	}

	addr := fmt.Sprintf("localhost:%s", port)
	slog.Info("discovered server via Redis", "node_key", key, "address", addr)
	return addr
}

func main() {
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx, cancel := context.WithCancel(context.Background())
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		slog.Info("shutting down — closing stream")
		cancel()
	}()

	// Resolve target server address (direct or via Redis discovery).
	addr := resolveServerAddr(ctx)

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("failed to dial server", "addr", addr, "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	grpcClient := pb.NewMetricsServiceClient(conn)

	for {
		if ctx.Err() != nil {
			slog.Info("client stopped")
			return
		}

		slog.Info("connecting", "server", addr, "node", *nodeID)
		if err := streamMetrics(ctx, grpcClient); err != nil && ctx.Err() == nil {
			slog.Warn("stream error - reconnecting in 2s", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}
}
