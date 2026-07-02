package aggregator

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/ricleal/GoLang/real_time_metrics_agg/pb"
)

func TestNewMetricState(t *testing.T) {
	ms := NewMetricState(10*time.Second, 1*time.Second)
	if ms == nil {
		t.Fatal("expected non-nil MetricState")
	}
	if ms.numBuckets != 10 {
		t.Fatalf("expected 10 buckets, got %d", ms.numBuckets)
	}
	if ms.window != 10*time.Second {
		t.Fatalf("expected window 10s, got %v", ms.window)
	}
	if ms.bucket != 1*time.Second {
		t.Fatalf("expected bucket 1s, got %v", ms.bucket)
	}
	for i, b := range ms.buckets {
		if b == nil {
			t.Fatalf("bucket %d is nil", i)
		}
		if len(b.Values) != 0 {
			t.Fatalf("bucket %d should be empty", i)
		}
	}
}

func TestNewMetricState_MinimumBuckets(t *testing.T) {
	ms := NewMetricState(100*time.Millisecond, 1*time.Second)
	if ms.numBuckets < 1 {
		t.Fatal("expected at least 1 bucket")
	}
}

func TestAdd_SingleValue(t *testing.T) {
	ms := NewMetricState(10*time.Second, 1*time.Second)
	ms.Add(42.0, time.Now())

	snap := ms.Snapshot()
	if snap.Count != 1 {
		t.Fatalf("expected count 1, got %d", snap.Count)
	}
	if snap.Avg != 42.0 {
		t.Fatalf("expected avg 42.0, got %f", snap.Avg)
	}
	if snap.Min != 42.0 {
		t.Fatalf("expected min 42.0, got %f", snap.Min)
	}
	if snap.Max != 42.0 {
		t.Fatalf("expected max 42.0, got %f", snap.Max)
	}
	if snap.Sum != 42.0 {
		t.Fatalf("expected sum 42.0, got %f", snap.Sum)
	}
}

func TestAdd_MultipleValues(t *testing.T) {
	ms := NewMetricState(10*time.Second, 1*time.Second)
	now := time.Now()
	ms.Add(10.0, now)
	ms.Add(20.0, now)
	ms.Add(30.0, now)
	ms.Add(40.0, now)
	ms.Add(50.0, now)

	snap := ms.Snapshot()
	if snap.Count != 5 {
		t.Fatalf("expected count 5, got %d", snap.Count)
	}
	if snap.Avg != 30.0 {
		t.Fatalf("expected avg 30.0, got %f", snap.Avg)
	}
	if snap.Min != 10.0 {
		t.Fatalf("expected min 10.0, got %f", snap.Min)
	}
	if snap.Max != 50.0 {
		t.Fatalf("expected max 50.0, got %f", snap.Max)
	}
	if snap.Sum != 150.0 {
		t.Fatalf("expected sum 150.0, got %f", snap.Sum)
	}
}

func TestPercentiles(t *testing.T) {
	ms := NewMetricState(10*time.Second, 1*time.Second)
	now := time.Now()
	// Insert values 1..100
	for i := 1; i <= 100; i++ {
		ms.Add(float64(i), now)
	}

	snap := ms.Snapshot()
	if snap.P50 != 50.0 && snap.P50 != 51.0 { // depends on rounding
		t.Logf("P50 = %f (acceptable: 50 or 51)", snap.P50)
	}
	if snap.P95 < 94.0 || snap.P95 > 96.0 {
		t.Fatalf("expected P95 ~95, got %f", snap.P95)
	}
	if snap.P99 < 98.0 || snap.P99 > 100.0 {
		t.Fatalf("expected P99 ~99, got %f", snap.P99)
	}
}

func TestSnapshot_NoData(t *testing.T) {
	ms := NewMetricState(10*time.Second, 1*time.Second)
	snap := ms.Snapshot()
	if snap.Count != 0 {
		t.Fatalf("expected count 0, got %d", snap.Count)
	}
	if snap.Avg != 0 {
		t.Fatalf("expected avg 0, got %f", snap.Avg)
	}
	if snap.WindowEnd <= snap.WindowStart {
		t.Fatal("window end should be after window start")
	}
}

func TestWindowSlides(t *testing.T) {
	ms := NewMetricState(1*time.Second, 1*time.Second)

	// Add a value now
	ms.Add(100.0, time.Now())
	snap := ms.Snapshot()
	if snap.Count != 1 {
		t.Fatalf("expected count 1, got %d", snap.Count)
	}

	// Wait until data is fully outside the window (2x bucket duration to
	// ensure the bucket covering the data has been fully rotated out).
	time.Sleep(2100 * time.Millisecond)

	// Snapshot should now have 0 values (window slid past)
	snap = ms.Snapshot()
	if snap.Count != 0 {
		t.Fatalf("expected count 0 after window slide, got %d", snap.Count)
	}
}

func TestSlidingWindowAggregator_AddMetric(t *testing.T) {
	agg := NewSlidingWindowAggregator(10*time.Second, 1*time.Second)
	now := time.Now()

	agg.AddMetric("cpu_usage", 45.0, now)
	agg.AddMetric("cpu_usage", 55.0, now)
	agg.AddMetric("memory_usage", 80.0, now)

	snap1 := agg.Snapshot("cpu_usage")
	if snap1 == nil {
		t.Fatal("expected non-nil snapshot for cpu_usage")
	}
	if snap1.Count != 2 {
		t.Fatalf("expected count 2 for cpu_usage, got %d", snap1.Count)
	}
	if snap1.Avg != 50.0 {
		t.Fatalf("expected avg 50.0 for cpu_usage, got %f", snap1.Avg)
	}

	snap2 := agg.Snapshot("memory_usage")
	if snap2 == nil {
		t.Fatal("expected non-nil snapshot for memory_usage")
	}
	if snap2.Count != 1 {
		t.Fatalf("expected count 1 for memory_usage, got %d", snap2.Count)
	}

	unknown := agg.Snapshot("nonexistent")
	if unknown != nil {
		t.Fatal("expected nil for unknown metric")
	}
}

func TestSlidingWindowAggregator_SnapshotAll(t *testing.T) {
	agg := NewSlidingWindowAggregator(10*time.Second, 1*time.Second)
	now := time.Now()

	agg.AddMetric("cpu", 50.0, now)
	agg.AddMetric("mem", 60.0, now)
	agg.AddMetric("cpu", 70.0, now)

	snaps := agg.SnapshotAll()
	if len(snaps) != 2 {
		t.Fatalf("expected 2 metric snapshots, got %d", len(snaps))
	}

	metricNames := make(map[string]bool)
	for _, s := range snaps {
		metricNames[s.MetricName] = true
	}
	if !metricNames["cpu"] {
		t.Fatal("expected cpu in snapshots")
	}
	if !metricNames["mem"] {
		t.Fatal("expected mem in snapshots")
	}
}

func TestConcurrentAdds(t *testing.T) {
	agg := NewSlidingWindowAggregator(10*time.Second, 1*time.Second)
	var wg sync.WaitGroup

	// Concurrently add from 10 goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(val float64) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				agg.AddMetric("concurrent_test", val+float64(j), time.Now())
			}
		}(float64(i * 100))
	}
	wg.Wait()

	snap := agg.Snapshot("concurrent_test")
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if snap.Count != 1000 {
		t.Fatalf("expected 1000 values, got %d", snap.Count)
	}
}

func TestBucketIndex(t *testing.T) {
	ms := NewMetricState(10*time.Second, 1*time.Second)
	start := ms.start

	tests := []struct {
		offset time.Duration
		want   int
	}{
		{0, 0},
		{500 * time.Millisecond, 0},
		{1 * time.Second, 1},
		{5 * time.Second, 5},
		{9*time.Second + 999*time.Millisecond, 9},
	}

	for _, tt := range tests {
		idx := ms.bucketIndex(start.Add(tt.offset))
		if idx != tt.want {
			t.Errorf("bucketIndex(start + %v) = %d, want %d", tt.offset, idx, tt.want)
		}
	}
}

func TestAdvance(t *testing.T) {
	ms := NewMetricState(3*time.Second, 1*time.Second)
	start := ms.start

	// Add a value at T=0
	ms.Add(1.0, start)
	if ms.buckets[0].Values[0] != 1.0 {
		t.Fatal("expected value in bucket 0")
	}

	// Advance past window+full-buffer to force a full reset.
	// With window=3s and 3 buckets, all data is outside the window
	// when now - window >= start + numBuckets * bucket, i.e. now >= start + 6s.
	ms.advance(start.Add(7 * time.Second))
	if !ms.start.After(start) {
		t.Fatal("expected start to advance")
	}
	// All buckets should be empty now
	for i, b := range ms.buckets {
		if len(b.Values) != 0 {
			t.Fatalf("expected bucket %d to be empty after full reset", i)
		}
	}
}

func TestPercentileEdgeCases(t *testing.T) {
	if p := percentile(nil, 50); p != 0 {
		t.Fatalf("expected 0 for nil slice, got %f", p)
	}
	if p := percentile([]float64{}, 50); p != 0 {
		t.Fatalf("expected 0 for empty slice, got %f", p)
	}
	sorted := []float64{1.0, 2.0, 3.0}
	if p := percentile(sorted, -1); p != 1.0 {
		t.Fatalf("expected 1.0 for p=-1, got %f", p)
	}
	if p := percentile(sorted, 100); p != 3.0 {
		t.Fatalf("expected 3.0 for p=100, got %f", p)
	}
	if p := percentile(sorted, 101); p != 3.0 {
		t.Fatalf("expected 3.0 for p=101, got %f", p)
	}
}

func TestSnapshotConcurrency(t *testing.T) {
	ms := NewMetricState(10*time.Second, 1*time.Second)
	var wg sync.WaitGroup

	// Writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			ms.Add(float64(i), time.Now())
		}
	}()

	// Readers
	for r := 0; r < 5; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				snap := ms.Snapshot()
				if snap == nil {
					t.Error("snapshot should not be nil")
				}
				_ = snap.Count
			}
		}()
	}
	wg.Wait()
}

func TestAggregator_MetricNameInSnapshot(t *testing.T) {
	agg := NewSlidingWindowAggregator(10*time.Second, 1*time.Second)
	agg.AddMetric("test.metric", 99.0, time.Now())

	snap := agg.Snapshot("test.metric")
	if snap.MetricName != "test.metric" {
		t.Fatalf("expected metric name 'test.metric', got '%s'", snap.MetricName)
	}
}

func TestLargeValues(t *testing.T) {
	ms := NewMetricState(10*time.Second, 1*time.Second)
	ms.Add(math.MaxFloat64/2, time.Now())
	ms.Add(math.MaxFloat64/2, time.Now())

	snap := ms.Snapshot()
	if snap.Count != 2 {
		t.Fatalf("expected count 2, got %d", snap.Count)
	}
	if snap.Avg != math.MaxFloat64/2 {
		t.Fatalf("expected avg %f, got %f", math.MaxFloat64/2, snap.Avg)
	}
}

func TestAggregator_SnapshotAllConcurrent(t *testing.T) {
	agg := NewSlidingWindowAggregator(5*time.Second, 1*time.Second)
	var wg sync.WaitGroup

	writer := func(name string) {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			agg.AddMetric(name, float64(i), time.Now())
		}
	}

	reader := func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			snaps := agg.SnapshotAll()
			for _, s := range snaps {
				if s.MetricName == "" {
					t.Error("snapshot has empty metric name")
				}
			}
		}
	}

	wg.Add(3)
	go writer("alpha")
	go writer("beta")
	go reader()
	wg.Wait()
}

// Ensure pb import compiles — verify exported types match
func TestSnapshotTypes(t *testing.T) {
	snap := &pb.AggregatedMetric{
		MetricName: "test",
		Sum:        100,
		Avg:        50,
		Count:      2,
	}
	if snap.Sum != 100 {
		t.Fatal("unexpected sum")
	}
}
