package transport

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/ricleal/GoLang/real_time_metrics_agg/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MetricsClient streams metrics to the aggregator server.
type MetricsClient struct {
	conn      *grpc.ClientConn
	client    pb.MetricsAggregatorClient
	clientID  string
	logger    *log.Logger
	mu        sync.Mutex
	stream    pb.MetricsAggregator_StreamMetricsClient
	cancel    context.CancelFunc
	reconnect bool
}

// NewMetricsClient creates a new metrics streaming client.
func NewMetricsClient(serverAddr, clientID string, logger *log.Logger) (*MetricsClient, error) {
	conn, err := grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024)),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}
	return &MetricsClient{
		conn:      conn,
		client:    pb.NewMetricsAggregatorClient(conn),
		clientID:  clientID,
		logger:    logger,
		reconnect: true,
	}, nil
}

// ConnectStream establishes a bidirectional stream.
func (c *MetricsClient) ConnectStream() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := c.client.StreamMetrics(ctx)
	if err != nil {
		cancel()
		return fmt.Errorf("stream metrics: %w", err)
	}
	c.stream = stream
	c.cancel = cancel
	return nil
}

// SendMetric sends a single metric on the stream.
func (c *MetricsClient) SendMetric(name string, value float64, labels map[string]string) error {
	c.mu.Lock()
	stream := c.stream
	c.mu.Unlock()

	if stream == nil {
		return fmt.Errorf("stream not connected")
	}

	metric := &pb.Metric{
		Name:      name,
		Value:     value,
		Timestamp: time.Now().UnixNano(),
		ClientId:  c.clientID,
		Labels:    labels,
	}
	return stream.Send(metric)
}

// ReceiveAggregates listens for aggregated metrics from the server.
func (c *MetricsClient) ReceiveAggregates(ctx context.Context, aggCh chan<- *pb.AggregatedMetric) error {
	c.mu.Lock()
	stream := c.stream
	c.mu.Unlock()

	if stream == nil {
		return fmt.Errorf("stream not connected")
	}

	for {
		agg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		select {
		case aggCh <- agg:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// channel full, drop
		}
	}
}

// SetThreshold sends a threshold update to the server.
func (c *MetricsClient) SetThreshold(metricName string, warn, critical float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.client.SetThreshold(ctx, &pb.SetThresholdRequest{
		MetricName: metricName,
		Warn:       warn,
		Critical:   critical,
	})
	return err
}

// HealthCheck performs a health check on the server.
func (c *MetricsClient) HealthCheck() (*pb.HealthCheckResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return c.client.HealthCheck(ctx, &pb.HealthCheckRequest{Service: "metrics-aggregator"})
}

// Close cleanly disconnects.
func (c *MetricsClient) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	return c.conn.Close()
}

// SimulateClient continuously streams realistic metrics for demo/testing.
func SimulateClient(ctx context.Context, client *MetricsClient, logger *log.Logger) error {
	if err := client.ConnectStream(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	// Goroutine to print received aggregates
	aggCh := make(chan *pb.AggregatedMetric, 32)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err := client.ReceiveAggregates(ctx, aggCh); err != nil && ctx.Err() == nil {
				logger.Printf("[WARN] recv aggregates: %v", err)
				time.Sleep(time.Second)
			}
		}
	}()

	go func() {
		for agg := range aggCh {
			logger.Printf("[AGG] %s: avg=%.2f count=%d p95=%.2f p99=%.2f (window: %s)",
				agg.MetricName, agg.Avg, agg.Count, agg.P95, agg.P99,
				time.Duration(agg.WindowEnd-agg.WindowStart))
		}
	}()

	logger.Printf("[INFO] client %s streaming metrics...", client.clientID)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	// Simulate a few different metric types with different patterns
	metricPatterns := []struct {
		name   string
		base   float64
		amp    float64
		period int // ticks per cycle
		tick   int
	}{
		{"cpu_usage", 45.0, 30.0, 40, 0},
		{"memory_usage", 60.0, 20.0, 60, 0},
		{"requests_per_sec", 1000.0, 500.0, 30, 0},
		{"error_rate", 2.0, 8.0, 100, 0},
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			for i, p := range metricPatterns {
				p.tick++
				if p.tick >= p.period {
					p.tick = 0
				}
				// Sinusoidal pattern with noise
				phase := float64(p.tick) / float64(p.period) * 2 * 3.14159
				noise := (rand.Float64() - 0.5) * p.amp * 0.2
				val := p.base + p.amp*0.5 + p.amp*0.5*math.Sin(phase) + noise
				if val < 0 {
					val = 0
				}
				metricPatterns[i].tick = p.tick

				labels := map[string]string{"host": fmt.Sprintf("host-%d", rand.Intn(5))}
				if err := client.SendMetric(p.name, val, labels); err != nil {
					logger.Printf("[WARN] send %s: %v", p.name, err)
				}
			}
		}
	}
}
