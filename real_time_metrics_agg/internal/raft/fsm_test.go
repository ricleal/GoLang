package raft

import (
	"bytes"
	"encoding/json"
	"testing"

	hashraft "github.com/hashicorp/raft"
)

func TestClusterFSM_Apply_ValidAlert(t *testing.T) {
	fsm := &ClusterFSM{}

	event := AlertEvent{
		MetricName:   "cpu_usage",
		Threshold:    80.0,
		CurrentValue: 95.0,
		Severity:     "CRITICAL",
		Message:      "[CRITICAL] cpu_usage avg (95.00) crossed threshold (80.00)",
		TriggeredAt:  1000,
		LeaderID:     "node1",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	result := fsm.Apply(&hashraft.Log{Data: data})
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}

	if len(fsm.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(fsm.Alerts))
	}

	got := fsm.Alerts[0]
	if got.MetricName != "cpu_usage" {
		t.Fatalf("expected metric 'cpu_usage', got '%s'", got.MetricName)
	}
	if got.Threshold != 80.0 {
		t.Fatalf("expected threshold 80.0, got %f", got.Threshold)
	}
	if got.Severity != "CRITICAL" {
		t.Fatalf("expected CRITICAL, got %s", got.Severity)
	}
	if got.LeaderID != "node1" {
		t.Fatalf("expected leaderID 'node1', got '%s'", got.LeaderID)
	}
}

func TestClusterFSM_Apply_InvalidJSON(t *testing.T) {
	fsm := &ClusterFSM{}

	result := fsm.Apply(&hashraft.Log{Data: []byte("not valid json")})
	if result == nil {
		t.Fatal("expected error for invalid JSON")
	}

	if len(fsm.Alerts) != 0 {
		t.Fatal("expected no alerts added for invalid JSON")
	}
}

func TestClusterFSM_Apply_MultipleAlerts(t *testing.T) {
	fsm := &ClusterFSM{}

	for i := 0; i < 5; i++ {
		event := AlertEvent{
			MetricName:   "cpu_usage",
			CurrentValue: float64(i),
			Message:      "alert",
		}
		data, _ := json.Marshal(event)
		fsm.Apply(&hashraft.Log{Data: data})
	}

	if len(fsm.Alerts) != 5 {
		t.Fatalf("expected 5 alerts, got %d", len(fsm.Alerts))
	}

	for i, a := range fsm.Alerts {
		if a.CurrentValue != float64(i) {
			t.Fatalf("alert %d: expected value %d, got %f", i, i, a.CurrentValue)
		}
	}
}

func TestClusterFSM_Apply_MaxAlertsCap(t *testing.T) {
	fsm := &ClusterFSM{}

	for i := 0; i < 1010; i++ {
		event := AlertEvent{
			MetricName:   "cpu",
			CurrentValue: float64(i),
		}
		data, _ := json.Marshal(event)
		fsm.Apply(&hashraft.Log{Data: data})
	}

	if len(fsm.Alerts) > 1000 {
		t.Fatalf("expected at most 1000 alerts, got %d", len(fsm.Alerts))
	}

	// Should have the most recent ones
	lastVal := fsm.Alerts[len(fsm.Alerts)-1].CurrentValue
	if lastVal != 1009 {
		t.Fatalf("expected last value 1009, got %f", lastVal)
	}
}

func TestClusterFSM_SnapshotAndRestore(t *testing.T) {
	fsm := &ClusterFSM{}

	// Add some alerts
	for i := 0; i < 10; i++ {
		event := AlertEvent{
			MetricName:   "cpu",
			CurrentValue: float64(i),
			Message:      "alert",
		}
		data, _ := json.Marshal(event)
		fsm.Apply(&hashraft.Log{Data: data})
	}

	// Take snapshot
	snap, err := fsm.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	// Persist to buffer
	var buf bytes.Buffer
	mockSink := &mockSnapshotSink{Writer: &buf}
	if err := snap.Persist(mockSink); err != nil {
		t.Fatalf("persist: %v", err)
	}

	// Restore into new FSM
	newFSM := &ClusterFSM{}
	if err := newFSM.Restore(&nopReadCloser{Reader: bytes.NewReader(buf.Bytes())}); err != nil {
		t.Fatalf("restore: %v", err)
	}

	if len(newFSM.Alerts) != 10 {
		t.Fatalf("expected 10 alerts after restore, got %d", len(newFSM.Alerts))
	}

	for i, a := range newFSM.Alerts {
		if a.CurrentValue != float64(i) {
			t.Fatalf("restored alert %d: expected value %d, got %f", i, i, a.CurrentValue)
		}
	}
}

func TestClusterFSM_Restore_InvalidData(t *testing.T) {
	fsm := &ClusterFSM{}
	err := fsm.Restore(&nopReadCloser{Reader: bytes.NewReader([]byte("not json"))})
	if err == nil {
		t.Fatal("expected error for invalid restore data")
	}
}

func TestClusterSnapshot_Release(t *testing.T) {
	snap := &ClusterSnapshot{}
	// Should not panic
	snap.Release()
}

// --- Mocks ---

type mockSnapshotSink struct {
	hashraft.SnapshotSink
	Writer *bytes.Buffer
	cancel bool
}

func (s *mockSnapshotSink) Write(p []byte) (n int, err error) {
	return s.Writer.Write(p)
}

func (s *mockSnapshotSink) Close() error {
	return nil
}

func (s *mockSnapshotSink) Cancel() error {
	s.cancel = true
	return nil
}

type nopReadCloser struct {
	*bytes.Reader
}

func (n *nopReadCloser) Close() error {
	return nil
}
