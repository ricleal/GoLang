package coordinator

import (
	"context"
	"encoding/json"
	"testing"
)

// ── JSON round-trip ──────────────────────────────────────────────────────────

func TestNodeAggregatesJSON(t *testing.T) {
	na := NodeAggregates{
		NodeID: "node-1",
		Metrics: map[string]float64{
			"cpu":     85.5,
			"network": 500_000,
		},
	}

	data, err := json.Marshal(na)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded NodeAggregates
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.NodeID != "node-1" {
		t.Errorf("expected NodeID 'node-1', got %q", decoded.NodeID)
	}
	if decoded.Metrics["cpu"] != 85.5 {
		t.Errorf("expected cpu=85.5, got %f", decoded.Metrics["cpu"])
	}
	if decoded.Metrics["network"] != 500_000 {
		t.Errorf("expected network=500000, got %f", decoded.Metrics["network"])
	}
}

func TestClusterAlertJSON(t *testing.T) {
	alert := ClusterAlert{
		MetricName:  "cpu",
		ClusterAvg:  90.0,
		Threshold:   80.0,
		Message:     "CLUSTER ALERT cpu: cluster_avg=90.00 > threshold=80.00 (across 3 nodes)",
		TimestampMs: 1_234_567_890,
	}

	data, err := json.Marshal(alert)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ClusterAlert
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.MetricName != "cpu" {
		t.Errorf("expected MetricName 'cpu', got %q", decoded.MetricName)
	}
	if decoded.ClusterAvg != 90.0 {
		t.Errorf("expected ClusterAvg 90.0, got %f", decoded.ClusterAvg)
	}
	if decoded.Threshold != 80.0 {
		t.Errorf("expected Threshold 80.0, got %f", decoded.Threshold)
	}
	if decoded.TimestampMs != 1_234_567_890 {
		t.Errorf("expected TimestampMs 1234567890, got %d", decoded.TimestampMs)
	}
	if decoded.Message == "" {
		t.Error("expected non-empty Message")
	}
}

// ── New ──────────────────────────────────────────────────────────────────────

func TestNew(t *testing.T) {
	thresholds := map[string]float64{"cpu": 80, "default": 100}
	c := New("test-node", nil, thresholds)

	if c.nodeID != "test-node" {
		t.Errorf("expected nodeID 'test-node', got %q", c.nodeID)
	}
	if c.rdb != nil {
		t.Error("expected nil rdb (not provided)")
	}
	if c.thresholds["cpu"] != 80 {
		t.Errorf("expected cpu threshold 80, got %f", c.thresholds["cpu"])
	}
	if c.thresholds["default"] != 100 {
		t.Errorf("expected default threshold 100, got %f", c.thresholds["default"])
	}
}

func TestNew_NilThresholds(t *testing.T) {
	// Passing nil thresholds should not panic — the map is stored as-is.
	c := New("test-node", nil, nil)
	if c.thresholds != nil {
		t.Error("expected thresholds to be nil")
	}
}

// ── PublishLocalAggregates — nil-safety ──────────────────────────────────────

func TestPublishLocalAggregates_EmptyMapDoesNotPanic(t *testing.T) {
	c := New("safe-node", nil, nil)

	// These should return early without touching rdb (which is nil).
	c.PublishLocalAggregates(context.Background(), nil)
	c.PublishLocalAggregates(context.Background(), map[string]float64{})
}

// ── threshold fallback logic (exercised by evalAndPublish) ───────────────────

func TestThresholdFallback(t *testing.T) {
	// Simulate the threshold-lookup logic used in evalAndPublish.
	thresholds := map[string]float64{"cpu": 80, "default": 100}

	tests := []struct {
		metric   string
		expected float64
		label    string
	}{
		{"cpu", 80, "explicit threshold"},
		{"network", 100, "fallback to default"},
		{"unknown", 100, "fallback to default"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got := thresholds["default"]
			if v, ok := thresholds[tt.metric]; ok {
				got = v
			}
			if got != tt.expected {
				t.Errorf("metric %q: expected %.0f, got %.0f", tt.metric, tt.expected, got)
			}
		})
	}
}
