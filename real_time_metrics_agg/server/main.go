package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"exp/real_time_metrics_agg/aggregator"
	"exp/real_time_metrics_agg/coordinator"
	pb "exp/real_time_metrics_agg/gen"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

const (
	backpressureBuffer  = 128
	alertInterval       = time.Second
	redisNodeExpiry     = 10 * time.Second
	clusterAlertBuffer  = 16
)

var (
	port      = flag.Int("port", 50051, "gRPC listener port")
	nodeID    = flag.String("node-id", "node-1", "unique identifier for this worker")
	redisAddr = flag.String("redis", "localhost:6379", "Redis address (host:port)")
)

type metricsServer struct {
	pb.UnimplementedMetricsServiceServer

	agg    *aggregator.Aggregator
	coord  *coordinator.Coordinator
	rdb    *redis.Client
	nodeID string

	// connMu guards activeConns and clusterSubs.
	connMu      sync.Mutex
	activeConns map[string]context.CancelFunc
	// clusterSubs delivers leader-published cluster alerts to every active stream.
	clusterSubs map[string]chan *pb.Alert

	wg sync.WaitGroup
}

func newServer(nodeID string, rdb *redis.Client, agg *aggregator.Aggregator, coord *coordinator.Coordinator) *metricsServer {
	return &metricsServer{
		agg:         agg,
		coord:       coord,
		rdb:         rdb,
		nodeID:      nodeID,
		activeConns: make(map[string]context.CancelFunc),
		clusterSubs: make(map[string]chan *pb.Alert),
	}
}

// broadcastClusterAlert fans out a cluster-wide alert to all connected streams.
func (s *metricsServer) broadcastClusterAlert(alert *pb.Alert) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	for _, ch := range s.clusterSubs {
		select {
		case ch <- alert:
		default: // non-blocking; slow consumers are skipped
		}
	}
}

// StreamMetrics is the bidirectional streaming RPC handler.
func (s *metricsServer) StreamMetrics(stream pb.MetricsService_StreamMetricsServer) error {
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	key := fmt.Sprintf("%s-%d", s.nodeID, time.Now().UnixNano())
	clusterCh := make(chan *pb.Alert, clusterAlertBuffer)

	s.connMu.Lock()
	s.activeConns[key] = cancel
	s.clusterSubs[key] = clusterCh
	s.connMu.Unlock()
	defer func() {
		s.connMu.Lock()
		delete(s.activeConns, key)
		delete(s.clusterSubs, key)
		s.connMu.Unlock()
	}()

	s.wg.Add(1)
	defer s.wg.Done()

	// Bounded channel for backpressure on incoming metric events.
	incoming := make(chan *pb.MetricEvent, backpressureBuffer)
	recvDone := make(chan error, 1)

	go func() {
		for {
			evt, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					recvDone <- nil
				} else {
					recvDone <- err
				}
				return
			}
			select {
			case incoming <- evt:
			default:
				slog.Warn("backpressure: dropping metric event",
					"node", evt.NodeId, "metric", evt.MetricName)
			}
		}
	}()

	alertTicker := time.NewTicker(alertInterval)
	defer alertTicker.Stop()

	seenSeqs := make(map[string]int64)

	for {
		select {
		case <-ctx.Done():
			return status.Error(codes.Canceled, "server shutting down")

		case err := <-recvDone:
			return err

		case evt := <-incoming:
			// Deduplication on reconnect.
			if evt.Seq > 0 {
				if evt.Seq <= seenSeqs[evt.NodeId] {
					slog.Debug("dedup: skipping seen seq",
						"node", evt.NodeId, "seq", evt.Seq)
					continue
				}
				seenSeqs[evt.NodeId] = evt.Seq
			}
			s.agg.Record(evt.MetricName, evt.Value)

		case <-alertTicker.C:
			// Local node alerts (each node evaluates its own window).
			for _, a := range s.agg.CheckAlerts() {
				msg := &pb.Alert{
					MetricName:  a.MetricName,
					Average:     a.Average,
					Threshold:   a.Threshold,
					Message:     fmt.Sprintf("[%s] LOCAL ALERT %s: avg=%.2f > threshold=%.2f", s.nodeID, a.MetricName, a.Average, a.Threshold),
					TimestampMs: time.Now().UnixMilli(),
				}
				if err := stream.Send(msg); err != nil {
					return err
				}
			}

		case alert := <-clusterCh:
			// Cluster-wide alert forwarded from the leader via Redis Pub/Sub.
			if err := stream.Send(alert); err != nil {
				return err
			}
		}
	}
}

func (s *metricsServer) registerNode(ctx context.Context) {
	key := fmt.Sprintf("metrics:node:%s", s.nodeID)
	_ = s.rdb.Set(ctx, key, *port, redisNodeExpiry).Err()

	ticker := time.NewTicker(redisNodeExpiry / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = s.rdb.Del(context.Background(), key).Err()
			slog.Info("node deregistered from Redis", "node", s.nodeID)
			return
		case <-ticker.C:
			if err := s.rdb.Set(ctx, key, *port, redisNodeExpiry).Err(); err != nil {
				slog.Warn("redis heartbeat failed", "err", err)
			}
		}
	}
}

// publishAggregates periodically writes local averages to Redis so the leader
// can collect them for cluster-wide evaluation.
func (s *metricsServer) publishAggregates(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.coord.PublishLocalAggregates(ctx, s.agg.Averages())
		}
	}
}

func (s *metricsServer) drainAndClose() {
	s.connMu.Lock()
	for _, cancel := range s.activeConns {
		cancel()
	}
	s.connMu.Unlock()
	s.wg.Wait()
}

func main() {
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: *redisAddr})
	redisOK := rdb.Ping(ctx).Err() == nil
	if !redisOK {
		slog.Warn("Redis unavailable - running in standalone mode (no leader election)")
	} else {
		slog.Info("connected to Redis", "addr", *redisAddr)
	}

	agg := aggregator.New()
	coord := coordinator.New(*nodeID, rdb, agg.Thresholds())
	srv := newServer(*nodeID, rdb, agg, coord)

	// Node registration heartbeat.
	go srv.registerNode(ctx)

	if redisOK {
		// Publish this node's local averages for the leader to aggregate.
		go srv.publishAggregates(ctx)

		// Compete for the leader role and evaluate cluster-wide alerts when elected.
		go coord.RunLeaderElection(ctx, func() {}, func() {})
		go coord.RunClusterEval(ctx)

		// Subscribe to cluster alerts published by the leader and broadcast to
		// all streams on this node.
		go coord.SubscribeAlerts(ctx, func(a coordinator.ClusterAlert) {
			srv.broadcastClusterAlert(&pb.Alert{
				MetricName:  a.MetricName,
				Average:     a.ClusterAvg,
				Threshold:   a.Threshold,
				Message:     a.Message,
				TimestampMs: a.TimestampMs,
			})
		})
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		slog.Error("listen failed", "err", err)
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterMetricsServiceServer(grpcSrv, srv)

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	slog.Info("server started", "node", *nodeID, "port", *port)
	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Error("grpc serve error", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutdown signal received", "node", *nodeID)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	cancel()
	drained := make(chan struct{})
	go func() {
		srv.drainAndClose()
		close(drained)
	}()

	select {
	case <-drained:
		slog.Info("all streams drained cleanly")
	case <-time.After(10 * time.Second):
		slog.Warn("drain timeout reached - forcing stop")
	}

	grpcSrv.GracefulStop()
	_ = rdb.Close()
	slog.Info("server stopped", "node", *nodeID)
}
