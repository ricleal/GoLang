package aggregator

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/ricleal/GoLang/real_time_metrics_agg/pb"
)

// Bucket holds metric values for a single time bucket.
type Bucket struct {
	Values []float64
}

// MetricState tracks the sliding window for a single metric name.
type MetricState struct {
	mu         sync.RWMutex
	buckets    []*Bucket
	start      time.Time
	window     time.Duration
	bucket     time.Duration
	numBuckets int
}

// NewMetricState creates a new sliding-window state.
func NewMetricState(window, bucketDuration time.Duration) *MetricState {
	n := int(window / bucketDuration)
	if n < 1 {
		n = 1
	}
	now := time.Now().Truncate(bucketDuration)
	buckets := make([]*Bucket, n)
	for i := range buckets {
		buckets[i] = &Bucket{Values: make([]float64, 0, 64)}
	}
	return &MetricState{
		buckets:    buckets,
		start:      now,
		window:     window,
		bucket:     bucketDuration,
		numBuckets: n,
	}
}

// advance moves the window forward to the given time, dropping only buckets
// that fall entirely before [now - window), preserving data still in range.
func (ms *MetricState) advance(now time.Time) {
	now = now.Truncate(ms.bucket)
	windowStart := now.Add(-ms.window)

	// Only drop buckets that are entirely before windowStart.
	drop := int(windowStart.Sub(ms.start) / ms.bucket)
	if drop < 0 {
		drop = 0
	}
	if drop >= ms.numBuckets {
		// Everything is outside the window — full reset.
		for i := range ms.buckets {
			ms.buckets[i] = &Bucket{Values: make([]float64, 0, 64)}
		}
		ms.start = windowStart
		return
	}
	if drop > 0 {
		copy(ms.buckets, ms.buckets[drop:])
		for i := ms.numBuckets - drop; i < ms.numBuckets; i++ {
			ms.buckets[i] = &Bucket{Values: make([]float64, 0, 64)}
		}
		ms.start = ms.start.Add(time.Duration(drop) * ms.bucket)
	}
}

// bucketIndex returns the index within the window for a timestamp.
func (ms *MetricState) bucketIndex(t time.Time) int {
	idx := int(t.Sub(ms.start) / ms.bucket)
	if idx < 0 {
		return 0
	}
	if idx >= ms.numBuckets {
		return ms.numBuckets - 1
	}
	return idx
}

// Add inserts a metric value into the sliding window.
func (ms *MetricState) Add(value float64, ts time.Time) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.advance(ts)
	idx := ms.bucketIndex(ts)
	ms.buckets[idx].Values = append(ms.buckets[idx].Values, value)
}

// Snapshot computes aggregated values over the current window.
// Note: acquires a full write lock because advance() mutates bucket state.
func (ms *MetricState) Snapshot() *pb.AggregatedMetric {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	ms.advance(now)

	var all []float64
	for _, b := range ms.buckets {
		all = append(all, b.Values...)
	}

	if len(all) == 0 {
		return &pb.AggregatedMetric{
			WindowStart: ms.start.UnixNano(),
			WindowEnd:   ms.start.Add(ms.window).UnixNano(),
		}
	}

	sum := 0.0
	min := math.MaxFloat64
	max := -math.MaxFloat64
	for _, v := range all {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	avg := sum / float64(len(all))
	count := uint64(len(all))

	sorted := make([]float64, len(all))
	copy(sorted, all)
	sort.Float64s(sorted)

	p50 := percentile(sorted, 50)
	p95 := percentile(sorted, 95)
	p99 := percentile(sorted, 99)

	return &pb.AggregatedMetric{
		Sum:         sum,
		Avg:         avg,
		Min:         min,
		Max:         max,
		Count:       count,
		P50:         p50,
		P95:         p95,
		P99:         p99,
		WindowStart: ms.start.UnixNano(),
		WindowEnd:   ms.start.Add(ms.window).UnixNano(),
	}
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p < 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := (p * len(sorted) / 100)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// SlidingWindowAggregator manages multiple metric states.
type SlidingWindowAggregator struct {
	mu      sync.RWMutex
	metrics map[string]*MetricState
	window  time.Duration
	bucket  time.Duration
}

// NewSlidingWindowAggregator creates a new aggregator.
func NewSlidingWindowAggregator(window, bucket time.Duration) *SlidingWindowAggregator {
	return &SlidingWindowAggregator{
		metrics: make(map[string]*MetricState),
		window:  window,
		bucket:  bucket,
	}
}

// AddMetric adds a data point to the named metric's sliding window.
func (a *SlidingWindowAggregator) AddMetric(name string, value float64, ts time.Time) {
	a.mu.RLock()
	state, ok := a.metrics[name]
	a.mu.RUnlock()
	if !ok {
		a.mu.Lock()
		// Double-check
		state, ok = a.metrics[name]
		if !ok {
			state = NewMetricState(a.window, a.bucket)
			a.metrics[name] = state
		}
		a.mu.Unlock()
	}
	state.Add(value, ts)
}

// SnapshotAll returns aggregated snapshots for all tracked metrics.
func (a *SlidingWindowAggregator) SnapshotAll() []*pb.AggregatedMetric {
	a.mu.RLock()
	names := make([]string, 0, len(a.metrics))
	for n := range a.metrics {
		names = append(names, n)
	}
	a.mu.RUnlock()

	result := make([]*pb.AggregatedMetric, 0, len(names))
	for _, n := range names {
		a.mu.RLock()
		state := a.metrics[n]
		a.mu.RUnlock()
		snap := state.Snapshot()
		snap.MetricName = n
		result = append(result, snap)
	}
	return result
}

// Snapshot returns the aggregated metric for a single metric name.
func (a *SlidingWindowAggregator) Snapshot(name string) *pb.AggregatedMetric {
	a.mu.RLock()
	state, ok := a.metrics[name]
	a.mu.RUnlock()
	if !ok {
		return nil
	}
	snap := state.Snapshot()
	snap.MetricName = name
	return snap
}
