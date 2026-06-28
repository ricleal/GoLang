package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	leaderKey     = "metrics:leader"
	leaderTTL     = 5 * time.Second
	leaderRenew   = 2 * time.Second
	aggKeyPrefix  = "metrics:agg:"
	aggTTL        = 10 * time.Second
	AlertsChannel = "metrics:alerts"
	evalInterval  = time.Second
)

// NodeAggregates is the JSON payload each node writes to its Redis key.
type NodeAggregates struct {
	NodeID  string             `json:"node_id"`
	Metrics map[string]float64 `json:"metrics"`
}

// ClusterAlert is published by the leader to the alerts Pub/Sub channel.
type ClusterAlert struct {
	MetricName  string  `json:"metric_name"`
	ClusterAvg  float64 `json:"cluster_avg"`
	Threshold   float64 `json:"threshold"`
	Message     string  `json:"message"`
	TimestampMs int64   `json:"timestamp_ms"`
}

// Coordinator handles leader election and cluster-wide alert evaluation.
//
// Every node:
//   - Periodically writes its local per-metric averages to Redis
//     (key: metrics:agg:<nodeID>, TTL: 10 s).
//   - Competes for the leader lock (metrics:leader, TTL: 5 s, renewed every 2 s).
//
// The leader:
//   - Reads all metrics:agg:* keys every second.
//   - Computes the cluster-wide average per metric.
//   - Publishes a ClusterAlert to the metrics:alerts Pub/Sub channel when a
//     threshold is crossed.
//
// All nodes:
//   - Subscribe to metrics:alerts and forward incoming ClusterAlerts to their
//     connected gRPC streams via the handler passed to SubscribeAlerts.
type Coordinator struct {
	nodeID     string
	rdb        *redis.Client
	thresholds map[string]float64
}

// New creates a Coordinator.  thresholds maps metric names to alert thresholds;
// "default" is used as the fallback for unknown metrics.
func New(nodeID string, rdb *redis.Client, thresholds map[string]float64) *Coordinator {
	return &Coordinator{nodeID: nodeID, rdb: rdb, thresholds: thresholds}
}

// isLeader returns true when this node currently holds the leader lock.
func (c *Coordinator) isLeader(ctx context.Context) bool {
	val, err := c.rdb.Get(ctx, leaderKey).Result()
	return err == nil && val == c.nodeID
}

// tryAcquireOrRenewLeader attempts to acquire the leader lock, or renews it if
// this node already holds it.
func (c *Coordinator) tryAcquireOrRenewLeader(ctx context.Context) bool {
	ok, err := c.rdb.SetNX(ctx, leaderKey, c.nodeID, leaderTTL).Result()
	if err != nil {
		return false
	}
	if ok {
		return true // freshly acquired
	}
	// Lock exists – renew only if we own it.
	val, err := c.rdb.Get(ctx, leaderKey).Result()
	if err != nil || val != c.nodeID {
		return false
	}
	_ = c.rdb.Expire(ctx, leaderKey, leaderTTL).Err()
	return true
}

// PublishLocalAggregates writes the node's current metric averages to Redis so
// the leader can collect them for cluster-wide evaluation.
func (c *Coordinator) PublishLocalAggregates(ctx context.Context, averages map[string]float64) {
	if len(averages) == 0 {
		return
	}
	data, err := json.Marshal(NodeAggregates{NodeID: c.nodeID, Metrics: averages})
	if err != nil {
		return
	}
	_ = c.rdb.Set(ctx, aggKeyPrefix+c.nodeID, data, aggTTL).Err()
}

// RunLeaderElection runs the election loop until ctx is cancelled.
// onBecomeLeader / onLoseLeader are called once on each role transition.
func (c *Coordinator) RunLeaderElection(ctx context.Context, onBecomeLeader, onLoseLeader func()) {
	ticker := time.NewTicker(leaderRenew)
	defer ticker.Stop()

	wasLeader := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			isLeader := c.tryAcquireOrRenewLeader(ctx)
			switch {
			case isLeader && !wasLeader:
				slog.Info("became leader", "node", c.nodeID)
				onBecomeLeader()
			case !isLeader && wasLeader:
				slog.Info("lost leader role", "node", c.nodeID)
				onLoseLeader()
			}
			wasLeader = isLeader
		}
	}
}

// RunClusterEval collects all nodes' aggregates from Redis every second and,
// when this node is the leader, publishes a ClusterAlert for every metric whose
// cluster-wide average exceeds its threshold.
func (c *Coordinator) RunClusterEval(ctx context.Context) {
	ticker := time.NewTicker(evalInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !c.isLeader(ctx) {
				continue
			}
			c.evalAndPublish(ctx)
		}
	}
}

type accum struct {
	sum float64
	n   int
}

func (c *Coordinator) evalAndPublish(ctx context.Context) {
	// Use SCAN instead of KEYS to avoid blocking Redis for O(N) over all keys.
	var keys []string
	iter := c.rdb.Scan(ctx, 0, aggKeyPrefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		slog.Warn("redis scan failed", "err", err)
		return
	}
	if len(keys) == 0 {
		return
	}

	totals := make(map[string]*accum)
	for _, key := range keys {
		raw, err := c.rdb.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var na NodeAggregates
		if err := json.Unmarshal(raw, &na); err != nil {
			continue
		}
		for metric, avg := range na.Metrics {
			if totals[metric] == nil {
				totals[metric] = &accum{}
			}
			totals[metric].sum += avg
			totals[metric].n++
		}
	}

	now := time.Now().UnixMilli()
	for metric, acc := range totals {
		clusterAvg := acc.sum / float64(acc.n)
		threshold := c.thresholds["default"]
		if t, ok := c.thresholds[metric]; ok {
			threshold = t
		}
		if clusterAvg <= threshold {
			continue
		}
		alert := ClusterAlert{
			MetricName:  metric,
			ClusterAvg:  clusterAvg,
			Threshold:   threshold,
			Message:     fmt.Sprintf("CLUSTER ALERT %s: cluster_avg=%.2f > threshold=%.2f (across %d nodes)", metric, clusterAvg, threshold, acc.n),
			TimestampMs: now,
		}
		data, _ := json.Marshal(alert)
		if err := c.rdb.Publish(ctx, AlertsChannel, string(data)).Err(); err != nil {
			slog.Warn("failed to publish cluster alert", "err", err)
		} else {
			slog.Info("cluster alert published", "metric", metric,
				"avg", fmt.Sprintf("%.2f", clusterAvg), "nodes", acc.n)
		}
	}
}

// SubscribeAlerts blocks until ctx is cancelled, invoking handler for every
// ClusterAlert published on the alerts Pub/Sub channel.
func (c *Coordinator) SubscribeAlerts(ctx context.Context, handler func(ClusterAlert)) {
	sub := c.rdb.Subscribe(ctx, AlertsChannel)
	defer sub.Close()

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var alert ClusterAlert
			if err := json.Unmarshal([]byte(msg.Payload), &alert); err != nil {
				slog.Warn("malformed cluster alert payload", "err", err)
				continue
			}
			handler(alert)
		}
	}
}
