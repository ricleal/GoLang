package transport

import (
	"context"
	"log"
	"time"

	"github.com/ricleal/GoLang/real_time_metrics_agg/internal/aggregator"
	"github.com/ricleal/GoLang/real_time_metrics_agg/internal/alert"
	"github.com/ricleal/GoLang/real_time_metrics_agg/internal/raft"
	"github.com/ricleal/GoLang/real_time_metrics_agg/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MetricsServer implements the pb.MetricsAggregatorServer.
type MetricsServer struct {
	pb.UnimplementedMetricsAggregatorServer
	Aggregator  *aggregator.SlidingWindowAggregator
	AlertEngine *alert.Engine
	RaftNode    *raft.ClusterNode
	NodeID      string
	logger      *log.Logger

	// Backpressure: bounded channel per client stream
	streamSem chan struct{}
}

// NewMetricsServer creates a new gRPC metrics server.
func NewMetricsServer(
	agg *aggregator.SlidingWindowAggregator,
	alertEngine *alert.Engine,
	raftNode *raft.ClusterNode,
	nodeID string,
	logger *log.Logger,
	maxConcurrentStreams int,
) *MetricsServer {
	if maxConcurrentStreams <= 0 {
		maxConcurrentStreams = 100
	}
	return &MetricsServer{
		Aggregator:  agg,
		AlertEngine: alertEngine,
		RaftNode:    raftNode,
		NodeID:      nodeID,
		logger:      logger,
		streamSem:   make(chan struct{}, maxConcurrentStreams),
	}
}

// StreamMetrics handles bidirectional streaming of metrics.
func (s *MetricsServer) StreamMetrics(stream pb.MetricsAggregator_StreamMetricsServer) error {
	// Acquire backpressure slot
	select {
	case s.streamSem <- struct{}{}:
		defer func() { <-s.streamSem }()
	case <-stream.Context().Done():
		return stream.Context().Err()
	}

	clientID := "unknown"
	s.logger.Printf("[INFO] new metric stream established")

	// Start a periodic ticker to send aggregated snapshots back to client
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Channel for outgoing aggregates
	aggCh := make(chan *pb.AggregatedMetric, 16)

	// Goroutine to send aggregates back to client
	go func() {
		for {
			select {
			case <-stream.Context().Done():
				return
			case snap, ok := <-aggCh:
				if !ok {
					return
				}
				if err := stream.Send(snap); err != nil {
					s.logger.Printf("[WARN] send aggregate to %s: %v", clientID, err)
					return
				}
			}
		}
	}()

	// Main loop: receive metrics from client
	recvDone := make(chan struct{})
	go func() {
		defer close(recvDone)
		for {
			metric, err := stream.Recv()
			if err != nil {
				s.logger.Printf("[INFO] stream from %s closed: %v", clientID, err)
				return
			}
			clientID = metric.ClientId

			ts := time.Unix(0, metric.Timestamp)
			s.Aggregator.AddMetric(metric.Name, metric.Value, ts)

			// Check thresholds
			snap := s.Aggregator.Snapshot(metric.Name)
			if snap != nil {
				alerts := s.AlertEngine.Evaluate(snap)
				for _, a := range alerts {
					s.logger.Printf("[ALERT] %s", a.Message)
					// Replicate alert to Raft cluster (best-effort)
					if s.RaftNode != nil && s.RaftNode.IsLeader() {
						event := raft.AlertEvent{
							MetricName:   a.MetricName,
							Threshold:    a.Threshold,
							CurrentValue: a.CurrentValue,
							Severity:     a.Severity,
							Message:      a.Message,
							TriggeredAt:  a.TriggeredAt,
							LeaderID:     s.NodeID,
						}
						if err := s.RaftNode.ReplicateAlert(event); err != nil {
							s.logger.Printf("[WARN] replicate alert: %v", err)
						}
					}
				}
			}
		}
	}()

	// Periodic aggregate push
	for {
		select {
		case <-stream.Context().Done():
			<-recvDone
			return stream.Context().Err()
		case <-ticker.C:
			snaps := s.Aggregator.SnapshotAll()
			for _, snap := range snaps {
				select {
				case aggCh <- snap:
				default:
					// Drop if channel full (backpressure)
				}
			}
		}
	}
}

// SetThreshold sets alert thresholds for a metric.
func (s *MetricsServer) SetThreshold(ctx context.Context, req *pb.SetThresholdRequest) (*pb.SetThresholdResponse, error) {
	s.AlertEngine.SetThreshold(req.MetricName, req.Warn, req.Critical)
	s.logger.Printf("[INFO] threshold set: %s warn=%.2f critical=%.2f", req.MetricName, req.Warn, req.Critical)
	return &pb.SetThresholdResponse{Ok: true}, nil
}

// HealthCheck returns the health status of the node.
func (s *MetricsServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	isLeader := false
	if s.RaftNode != nil {
		isLeader = s.RaftNode.IsLeader()
	}
	return &pb.HealthCheckResponse{
		Status: "SERVING",
		Leader: isLeader,
	}, nil
}

// JoinCluster is a unary RPC to join a node to the Raft cluster.
func (s *MetricsServer) JoinCluster(ctx context.Context, req *pb.JoinClusterRequest) (*pb.JoinClusterResponse, error) {
	if s.RaftNode == nil {
		return nil, status.Error(codes.FailedPrecondition, "raft not initialized")
	}
	if err := s.RaftNode.Join(req.NodeId, req.RaftAddress); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.logger.Printf("[INFO] node %s joined cluster at %s", req.NodeId, req.RaftAddress)
	return &pb.JoinClusterResponse{Ok: true}, nil
}
