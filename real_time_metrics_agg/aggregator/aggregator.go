package aggregator

import (
	"sync"
	"time"
)

const (
	WindowSize    = 10
	BucketWidth   = time.Second
	AlertCooldown = 5 * time.Second
)

type bucket struct {
	sum   float64
	count int64
}

type metricWindow struct {
	mu          sync.RWMutex
	buckets     [WindowSize]bucket
	currentIdx  int
	lastRotated time.Time
	lastAlert   time.Time
	Threshold   float64
	Name        string
}

func (w *metricWindow) rotate() {
	now := time.Now()
	steps := int(now.Sub(w.lastRotated) / BucketWidth)
	if steps <= 0 {
		return
	}
	if steps > WindowSize {
		steps = WindowSize
	}
	for i := 0; i < steps; i++ {
		w.currentIdx = (w.currentIdx + 1) % WindowSize
		w.buckets[w.currentIdx] = bucket{}
	}
	w.lastRotated = now
}

func (w *metricWindow) Add(value float64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rotate()
	w.buckets[w.currentIdx].sum += value
	w.buckets[w.currentIdx].count++
}

func (w *metricWindow) average() (float64, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var total float64
	var n int64
	for _, b := range w.buckets {
		total += b.sum
		n += b.count
	}
	if n == 0 {
		return 0, false
	}
	return total / float64(n), true
}

func (w *metricWindow) CheckAlert() (avg float64, fire bool) {
	avg, ok := w.average()
	if !ok || avg <= w.Threshold {
		return avg, false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if time.Since(w.lastAlert) < AlertCooldown {
		return avg, false
	}
	w.lastAlert = time.Now()
	return avg, true
}

type AlertResult struct {
	MetricName string
	Average    float64
	Threshold  float64
}

type Aggregator struct {
	mu         sync.RWMutex
	windows    map[string]*metricWindow
	thresholds map[string]float64
}

func New() *Aggregator {
	return &Aggregator{
		windows: make(map[string]*metricWindow),
		thresholds: map[string]float64{
			"cpu":      80.0,
			"network":  1_000_000,
			"requests": 100.0,
			"default":  100.0,
		},
	}
}

func (a *Aggregator) Record(metricName string, value float64) {
	a.mu.RLock()
	w, ok := a.windows[metricName]
	a.mu.RUnlock()
	if !ok {
		a.mu.Lock()
		if w, ok = a.windows[metricName]; !ok {
			t := a.thresholds["default"]
			if v, found := a.thresholds[metricName]; found {
				t = v
			}
			w = &metricWindow{
				Name:        metricName,
				Threshold:   t,
				lastRotated: time.Now(),
			}
			a.windows[metricName] = w
		}
		a.mu.Unlock()
	}
	w.Add(value)
}

func (a *Aggregator) CheckAlerts() []AlertResult {
	a.mu.RLock()
	names := make([]string, 0, len(a.windows))
	for n := range a.windows {
		names = append(names, n)
	}
	a.mu.RUnlock()

	var out []AlertResult
	for _, name := range names {
		a.mu.RLock()
		w := a.windows[name]
		a.mu.RUnlock()
		if avg, fire := w.CheckAlert(); fire {
			out = append(out, AlertResult{
				MetricName: name,
				Average:    avg,
				Threshold:  w.Threshold,
			})
		}
	}
	return out
}

// Averages returns the current sliding-window average for every tracked metric.
// Metrics with no samples yet are omitted.
func (a *Aggregator) Averages() map[string]float64 {
	a.mu.RLock()
	names := make([]string, 0, len(a.windows))
	for n := range a.windows {
		names = append(names, n)
	}
	a.mu.RUnlock()

	out := make(map[string]float64, len(names))
	for _, name := range names {
		a.mu.RLock()
		w := a.windows[name]
		a.mu.RUnlock()
		if avg, ok := w.average(); ok {
			out[name] = avg
		}
	}
	return out
}

// Thresholds returns a copy of the per-metric threshold map.
func (a *Aggregator) Thresholds() map[string]float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make(map[string]float64, len(a.thresholds))
	for k, v := range a.thresholds {
		out[k] = v
	}
	return out
}
