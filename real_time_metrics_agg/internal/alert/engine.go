package alert

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ricleal/GoLang/real_time_metrics_agg/pb"
)

// Severity levels.
const (
	SeverityWarn     = "WARN"
	SeverityCritical = "CRITICAL"
)

// Threshold defines alert thresholds for a metric.
type Threshold struct {
	MetricName string
	Warn       float64 // value above this triggers WARN
	Critical   float64 // value above this triggers CRITICAL
}

// Engine checks aggregated metrics against thresholds and produces alerts.
type Engine struct {
	mu         sync.RWMutex
	thresholds map[string]*Threshold
	alerts     []*pb.Alert
	maxAlerts  int
}

// NewEngine creates an alert engine.
func NewEngine(maxAlerts int) *Engine {
	return &Engine{
		thresholds: make(map[string]*Threshold),
		alerts:     make([]*pb.Alert, 0, maxAlerts),
		maxAlerts:  maxAlerts,
	}
}

// SetThreshold sets or updates a threshold for a metric.
func (e *Engine) SetThreshold(metricName string, warn, critical float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.thresholds[metricName] = &Threshold{
		MetricName: metricName,
		Warn:       warn,
		Critical:   critical,
	}
}

// RemoveThreshold removes a threshold.
func (e *Engine) RemoveThreshold(metricName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.thresholds, metricName)
}

// GetThreshold returns the threshold for a metric.
func (e *Engine) GetThreshold(metricName string) *Threshold {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.thresholds[metricName]
}

// Evaluate checks an aggregated metric against thresholds, returning any new alerts.
func (e *Engine) Evaluate(snap *pb.AggregatedMetric) []*pb.Alert {
	e.mu.RLock()
	t, ok := e.thresholds[snap.MetricName]
	e.mu.RUnlock()
	if !ok {
		return nil
	}

	var alerts []*pb.Alert

	if snap.Avg >= t.Critical {
		a := e.newAlert(snap.MetricName, t.Critical, snap.Avg, SeverityCritical)
		alerts = append(alerts, a)
	} else if snap.Avg >= t.Warn {
		a := e.newAlert(snap.MetricName, t.Warn, snap.Avg, SeverityWarn)
		alerts = append(alerts, a)
	}

	e.mu.Lock()
	e.alerts = append(e.alerts, alerts...)
	if len(e.alerts) > e.maxAlerts {
		overflow := len(e.alerts) - e.maxAlerts
		e.alerts = e.alerts[overflow:]
	}
	e.mu.Unlock()

	return alerts
}

func (e *Engine) newAlert(metricName string, threshold, current float64, severity string) *pb.Alert {
	return &pb.Alert{
		Id:           fmt.Sprintf("alert-%d-%d", time.Now().UnixNano(), rand.Intn(1000)),
		MetricName:   metricName,
		Threshold:    threshold,
		CurrentValue: current,
		Severity:     severity,
		TriggeredAt:  time.Now().UnixNano(),
		Message:      fmt.Sprintf("[%s] %s avg (%.2f) crossed threshold (%.2f)", severity, metricName, current, threshold),
	}
}

// RecentAlerts returns the most recent alerts.
func (e *Engine) RecentAlerts(n int) []*pb.Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if n > len(e.alerts) {
		n = len(e.alerts)
	}
	result := make([]*pb.Alert, n)
	copy(result, e.alerts[len(e.alerts)-n:])
	return result
}
