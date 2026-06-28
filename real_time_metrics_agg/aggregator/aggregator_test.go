package aggregator

import (
	"math/rand"
	"sync"
	"testing"
)

// ── smoke tests ──────────────────────────────────────────────────────────────

func TestNew(t *testing.T) {
	a := New()
	if a == nil {
		t.Fatal("New() returned nil")
	}
	// Verify default thresholds.
	th := a.Thresholds()
	for _, k := range []string{"cpu", "network", "requests", "default"} {
		if _, ok := th[k]; !ok {
			t.Errorf("missing default threshold for %q", k)
		}
	}
}

func TestRecordAndAverage(t *testing.T) {
	a := New()
	a.Record("cpu", 50)
	a.Record("cpu", 60)

	avgs := a.Averages()
	avg, ok := avgs["cpu"]
	if !ok {
		t.Fatal("expected cpu metric in Averages()")
	}
	if avg != 55.0 {
		t.Errorf("expected average 55.0, got %.2f", avg)
	}
}

func TestMultipleMetrics(t *testing.T) {
	a := New()
	a.Record("cpu", 80)
	a.Record("network", 500_000)
	a.Record("requests", 50)

	avgs := a.Averages()
	if _, ok := avgs["cpu"]; !ok {
		t.Error("expected cpu")
	}
	if _, ok := avgs["network"]; !ok {
		t.Error("expected network")
	}
	if _, ok := avgs["requests"]; !ok {
		t.Error("expected requests")
	}
	if len(avgs) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(avgs))
	}
}

func TestZeroValues(t *testing.T) {
	a := New()
	a.Record("cpu", 0)
	a.Record("cpu", 0)

	avgs := a.Averages()
	avg, ok := avgs["cpu"]
	if !ok {
		t.Fatal("expected cpu metric")
	}
	if avg != 0.0 {
		t.Errorf("expected average 0.0, got %.2f", avg)
	}
}

// ── alert tests ──────────────────────────────────────────────────────────────

func TestAlertFiresWhenThresholdExceeded(t *testing.T) {
	a := New()
	// cpu threshold is 80 — push well above it.
	for i := 0; i < 100; i++ {
		a.Record("cpu", 90.0)
	}

	alerts := a.CheckAlerts()
	if len(alerts) == 0 {
		t.Fatal("expected at least one alert when cpu exceeds threshold")
	}
	for _, alert := range alerts {
		if alert.MetricName == "cpu" {
			if alert.Average <= alert.Threshold {
				t.Errorf("alert average %.2f should exceed threshold %.2f", alert.Average, alert.Threshold)
			}
			return
		}
	}
	t.Error("no alert found for cpu metric")
}

func TestNoAlertBelowThreshold(t *testing.T) {
	a := New()
	a.Record("cpu", 10.0)
	a.Record("cpu", 20.0)

	alerts := a.CheckAlerts()
	for _, alert := range alerts {
		if alert.MetricName == "cpu" {
			t.Error("unexpected alert for cpu — average is below threshold")
		}
	}
}

func TestAlertCooldown(t *testing.T) {
	a := New()
	// Fill the window with values above the cpu threshold.
	for i := 0; i < 100; i++ {
		a.Record("cpu", 90.0)
	}

	// First check → must fire.
	alerts1 := a.CheckAlerts()
	if len(alerts1) == 0 {
		t.Fatal("first CheckAlert() should fire — threshold exceeded")
	}

	// Immediate second check → must NOT fire (within 5 s cooldown).
	alerts2 := a.CheckAlerts()
	for _, a2 := range alerts2 {
		if a2.MetricName == "cpu" {
			t.Fatal("second immediate CheckAlert() should be suppressed by cooldown")
		}
	}
}

func TestAlertPerMetricIsIndependent(t *testing.T) {
	a := New()
	// Only cpu exceeds threshold.
	for i := 0; i < 100; i++ {
		a.Record("cpu", 90.0)
	}
	a.Record("network", 1.0)

	alerts := a.CheckAlerts()
	alertedMetrics := make(map[string]bool)
	for _, al := range alerts {
		alertedMetrics[al.MetricName] = true
	}
	if !alertedMetrics["cpu"] {
		t.Error("expected cpu alert")
	}
	if alertedMetrics["network"] {
		t.Error("unexpected network alert — value is below threshold")
	}
}

// ── thresholds ───────────────────────────────────────────────────────────────

func TestThresholdsReturnsCopy(t *testing.T) {
	a := New()
	th := a.Thresholds()
	// Modify the returned map.
	th["new"] = 999
	// Original should be unaffected.
	orig := a.Thresholds()
	if _, ok := orig["new"]; ok {
		t.Error("Thresholds() should return a copy, not a reference")
	}
}

// ── custom metric threshold ──────────────────────────────────────────────────

func TestCustomMetricThreshold(t *testing.T) {
	a := New()
	// "disk" is not in the predefined thresholds map, so it gets "default" (100).
	// With average = 50, no alert should fire.
	a.Record("disk", 50.0)
	alerts := a.CheckAlerts()
	for _, al := range alerts {
		if al.MetricName == "disk" {
			t.Error("unexpected disk alert — 50 < default threshold 100")
		}
	}
}

// ── concurrent safety (run with -race) ────────────────────────────────────────

func TestConcurrentAccess(t *testing.T) {
	a := New()
	var wg sync.WaitGroup

	// Multiple goroutines recording, reading averages, and checking alerts.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				a.Record("cpu", float64(j))
				a.Record("network", float64(j*1000))
				_ = a.Averages()
				_ = a.CheckAlerts()
				_ = a.Thresholds()
			}
		}(i)
	}
	wg.Wait()

	// After all concurrent writes, the aggregator should still produce sensible
	// results (no panic, no data corruption).
	avgs := a.Averages()
	if _, ok := avgs["cpu"]; !ok {
		t.Error("cpu metric should exist after concurrent writes")
	}
	if _, ok := avgs["network"]; !ok {
		t.Error("network metric should exist after concurrent writes")
	}
}

// ── double-checked locking correctness ────────────────────────────────────────

func TestConcurrentNewMetricCreation(t *testing.T) {
	a := New()
	var wg sync.WaitGroup

	// Multiple goroutines racing to create the same new metric.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.Record("mem", float64(rand.Intn(100)))
		}()
	}
	wg.Wait()

	avgs := a.Averages()
	if _, ok := avgs["mem"]; !ok {
		t.Error("mem metric should exist after concurrent creation")
	}
}
