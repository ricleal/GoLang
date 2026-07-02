package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/ricleal/GoLang/real_time_metrics_agg/internal/aggregator"
	"github.com/ricleal/GoLang/real_time_metrics_agg/internal/alert"
	"github.com/ricleal/GoLang/real_time_metrics_agg/internal/config"
	raftpkg "github.com/ricleal/GoLang/real_time_metrics_agg/internal/raft"
	"github.com/ricleal/GoLang/real_time_metrics_agg/internal/transport"
	"github.com/ricleal/GoLang/real_time_metrics_agg/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()

	logger := log.New(os.Stdout, fmt.Sprintf("[%s] ", cfg.NodeID), log.LstdFlags|log.Lmicroseconds)
	logger.Printf("starting node %s (gRPC: %s, Raft: %s)", cfg.NodeID, cfg.GrpcAddress(), cfg.RaftAddress())

	// --- Init Aggregator ---
	agg := aggregator.NewSlidingWindowAggregator(cfg.WindowDuration, cfg.BucketInterval)

	// --- Init Alert Engine ---
	alertEngine := alert.NewEngine(1000)

	// --- Init Raft Node ---
	raftDataDir := fmt.Sprintf("%s/%s", cfg.DataDir, cfg.NodeID)
	raftLogger := hclog.New(&hclog.LoggerOptions{
		Name:   fmt.Sprintf("raft-%s", cfg.NodeID),
		Level:  hclog.LevelFromString(cfg.LogLevel),
		Output: os.Stderr,
	})
	raftNode, err := raftpkg.NewClusterNode(cfg.NodeID, cfg.RaftAddress(), raftDataDir, cfg.Bootstrap, raftLogger)
	if err != nil {
		logger.Fatalf("failed to initialize Raft: %v", err)
	}

	// If a join address was provided, join the cluster
	if cfg.JoinAddress != "" {
		// Give Raft a moment to initialize
		time.Sleep(500 * time.Millisecond)
		if err := joinCluster(cfg.JoinAddress, cfg.NodeID, cfg.RaftAddress()); err != nil {
			logger.Printf("[WARN] join cluster: %v (retrying in background)", err)
			go func() {
				for i := 0; i < 10; i++ {
					time.Sleep(2 * time.Second)
					if err := joinCluster(cfg.JoinAddress, cfg.NodeID, cfg.RaftAddress()); err == nil {
						logger.Printf("[INFO] successfully joined cluster")
						return
					}
				}
				logger.Printf("[ERROR] failed to join cluster after retries")
			}()
		} else {
			logger.Printf("[INFO] joined cluster at %s", cfg.JoinAddress)
		}
	}

	// --- Init gRPC Server ---
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(1024*1024),
		grpc.MaxSendMsgSize(1024*1024),
	)

	metricsServer := transport.NewMetricsServer(agg, alertEngine, raftNode, cfg.NodeID, logger, 100)
	pb.RegisterMetricsAggregatorServer(grpcServer, metricsServer)

	// Health check service
	healthServer := health.NewServer()
	healthServer.SetServingStatus("metrics-aggregator", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// Reflection for debugging
	reflection.Register(grpcServer)

	// --- Start Listening ---
	lis, err := net.Listen("tcp", cfg.GrpcAddress())
	if err != nil {
		logger.Fatalf("failed to listen on %s: %v", cfg.GrpcAddress(), err)
	}

	// --- Graceful Shutdown ---
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	go func() {
		<-ctx.Done()
		logger.Printf("received signal, initiating graceful shutdown...")

		// Stop accepting new RPCs
		grpcServer.GracefulStop()

		// Shutdown Raft (with drain)
		if raftNode != nil {
			if raftNode.IsLeader() {
				logger.Printf("[INFO] leader stepping down...")
			}
			raftNode.Raft.LeaderCh()
			future := raftNode.Raft.Shutdown()
			if err := future.Error(); err != nil {
				logger.Printf("[WARN] raft shutdown: %v", err)
			}
		}

		logger.Printf("shutdown complete")
	}()

	logger.Printf("gRPC server listening on %s", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		logger.Printf("[INFO] gRPC serve stopped: %v", err)
	}

	// Wait for the shutdown goroutine to finish
	<-ctx.Done()
}

// joinCluster sends a JoinCluster RPC to an existing node.
func joinCluster(joinAddr, nodeID, raftAddr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, joinAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("dial %s: %w", joinAddr, err)
	}
	defer conn.Close()

	client := pb.NewMetricsAggregatorClient(conn)
	resp, err := client.JoinCluster(ctx, &pb.JoinClusterRequest{
		NodeId:      nodeID,
		RaftAddress: raftAddr,
	})
	if err != nil {
		return fmt.Errorf("join cluster rpc: %w", err)
	}
	if !resp.Ok {
		return fmt.Errorf("join cluster rejected")
	}
	return nil
}
