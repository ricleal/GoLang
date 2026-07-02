package alert

import (
	"testing"

	"github.com/ricleal/GoLang/real_time_metrics_agg/pb"
)

func TestNewEngine(t *testing.T) {
	e := NewEngine(100)
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if len(e.thresholds) != 0 {
		t.Fatal("expected no thresholds initially")
	}
	if cap(e.alerts) != 100 {
		t.Fatalf("expected alert cap 100, got %d", cap(e.alerts))
	}
}

func TestSetAndGetThreshold(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu_usage", 50.0, 80.0)

	th := e.GetThreshold("cpu_usage")
	if th == nil {
		t.Fatal("expected non-nil threshold")
	}
	if th.MetricName != "cpu_usage" {
		t.Fatalf("expected metric name 'cpu_usage', got '%s'", th.MetricName)
	}
	if th.Warn != 50.0 {
		t.Fatalf("expected warn threshold 50.0, got %f", th.Warn)
	}
	if th.Critical != 80.0 {
		t.Fatalf("expected critical threshold 80.0, got %f", th.Critical)
	}
}

func TestGetThreshold_Nonexistent(t *testing.T) {
	e := NewEngine(100)
	if th := e.GetThreshold("nonexistent"); th != nil {
		t.Fatal("expected nil for nonexistent metric")
	}
}

func TestRemoveThreshold(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu_usage", 50.0, 80.0)
	e.RemoveThreshold("cpu_usage")
	if th := e.GetThreshold("cpu_usage"); th != nil {
		t.Fatal("expected nil after removal")
	}
}

func TestEvaluate_NoThreshold(t *testing.T) {
	e := NewEngine(100)
	snap := &pb.AggregatedMetric{
		MetricName: "cpu_usage",
		Avg:        90.0,
	}
	alerts := e.Evaluate(snap)
	if alerts != nil {
		t.Fatal("expected no alerts when no threshold set")
	}
}

func TestEvaluate_WarnThreshold(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu_usage", 50.0, 80.0)

	snap := &pb.AggregatedMetric{
		MetricName: "cpu_usage",
		Avg:        65.0,
	}
	alerts := e.Evaluate(snap)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Severity != SeverityWarn {
		t.Fatalf("expected severity WARN, got %s", alerts[0].Severity)
	}
	if alerts[0].MetricName != "cpu_usage" {
		t.Fatalf("expected metric 'cpu_usage', got '%s'", alerts[0].MetricName)
	}
	if alerts[0].Threshold != 50.0 {
		t.Fatalf("expected threshold 50.0, got %f", alerts[0].Threshold)
	}
	if alerts[0].CurrentValue != 65.0 {
		t.Fatalf("expected current value 65.0, got %f", alerts[0].CurrentValue)
	}
	if alerts[0].Message == "" {
		t.Fatal("expected non-empty alert message")
	}
}

func TestEvaluate_CriticalThreshold(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu_usage", 50.0, 80.0)

	snap := &pb.AggregatedMetric{
		MetricName: "cpu_usage",
		Avg:        95.0,
	}
	alerts := e.Evaluate(snap)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Severity != SeverityCritical {
		t.Fatalf("expected severity CRITICAL, got %s", alerts[0].Severity)
	}
	if alerts[0].Threshold != 80.0 {
		t.Fatalf("expected threshold 80.0, got %f", alerts[0].Threshold)
	}
}

func TestEvaluate_BelowThreshold(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu_usage", 50.0, 80.0)

	snap := &pb.AggregatedMetric{
		MetricName: "cpu_usage",
		Avg:        30.0,
	}
	alerts := e.Evaluate(snap)
	if alerts != nil {
		t.Fatal("expected no alerts when below threshold")
	}
}

func TestEvaluate_AtExactWarnThreshold(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu_usage", 50.0, 80.0)

	snap := &pb.AggregatedMetric{
		MetricName: "cpu_usage",
		Avg:        50.0,
	}
	alerts := e.Evaluate(snap)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert at exact warn threshold, got %d", len(alerts))
	}
}

func TestEvaluate_AtExactCriticalThreshold(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu_usage", 50.0, 80.0)

	snap := &pb.AggregatedMetric{
		MetricName: "cpu_usage",
		Avg:        80.0,
	}
	alerts := e.Evaluate(snap)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert at exact critical threshold, got %d", len(alerts))
	}
	if alerts[0].Severity != SeverityCritical {
		t.Fatalf("expected CRITICAL at exact critical threshold")
	}
}

func TestEvaluate_MultipleMetrics(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu", 50.0, 80.0)
	e.SetThreshold("mem", 70.0, 90.0)

	alerts1 := e.Evaluate(&pb.AggregatedMetric{MetricName: "cpu", Avg: 60.0})
	if len(alerts1) != 1 {
		t.Fatal("expected alert for cpu")
	}

	alerts2 := e.Evaluate(&pb.AggregatedMetric{MetricName: "mem", Avg: 95.0})
	if len(alerts2) != 1 {
		t.Fatal("expected alert for mem")
	}
	if alerts2[0].Severity != SeverityCritical {
		t.Fatalf("expected CRITICAL for mem, got %s", alerts2[0].Severity)
	}

	alerts3 := e.Evaluate(&pb.AggregatedMetric{MetricName: "disk", Avg: 50.0})
	if alerts3 != nil {
		t.Fatal("expected no alert for disk (no threshold)")
	}
}

func TestRecentAlerts(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu", 10.0, 50.0)

	// Generate several alerts
	for i := 0; i < 10; i++ {
		e.Evaluate(&pb.AggregatedMetric{MetricName: "cpu", Avg: 30.0})
	}

	recent := e.RecentAlerts(3)
	if len(recent) != 3 {
		t.Fatalf("expected 3 recent alerts, got %d", len(recent))
	}

	// Most recent should be the last one triggered
	all := e.RecentAlerts(100)
	if len(all) != 10 {
		t.Fatalf("expected all 10 alerts, got %d", len(all))
	}
}

func TestRecentAlerts_Empty(t *testing.T) {
	e := NewEngine(100)
	recent := e.RecentAlerts(5)
	if len(recent) != 0 {
		t.Fatal("expected no alerts when none triggered")
	}
}

func TestRecentAlerts_RequestMoreThanAvailable(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu", 10.0, 50.0)
	e.Evaluate(&pb.AggregatedMetric{MetricName: "cpu", Avg: 30.0})

	recent := e.RecentAlerts(100)
	if len(recent) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(recent))
	}
}

func TestMaxAlertsCap(t *testing.T) {
	e := NewEngine(5)
	e.SetThreshold("cpu", 10.0, 50.0)

	// Generate 10 alerts — only last 5 should be kept
	for i := 0; i < 10; i++ {
		e.Evaluate(&pb.AggregatedMetric{MetricName: "cpu", Avg: 30.0})
	}

	all := e.RecentAlerts(100)
	if len(all) != 5 {
		t.Fatalf("expected 5 alerts (capped), got %d", len(all))
	}
}

func TestEvaluate_UpdatesThreshold(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu", 50.0, 80.0)

	// Below old threshold, above new one
	snap := &pb.AggregatedMetric{MetricName: "cpu", Avg: 40.0}
	if alerts := e.Evaluate(snap); alerts != nil {
		t.Fatal("expected no alert with 40.0 < 50.0")
	}

	// Update threshold lower
	e.SetThreshold("cpu", 30.0, 60.0)
	if alerts := e.Evaluate(snap); len(alerts) != 1 {
		t.Fatal("expected alert after lowering threshold to 30")
	}
}

func TestAlertIdIsUnique(t *testing.T) {
	e := NewEngine(100)
	e.SetThreshold("cpu", 10.0, 50.0)

	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		alerts := e.Evaluate(&pb.AggregatedMetric{MetricName: "cpu", Avg: 30.0})
		if ids[alerts[0].Id] {
			t.Fatal("expected unique alert IDs")
		}
		ids[alerts[0].Id] = true
	}
}
