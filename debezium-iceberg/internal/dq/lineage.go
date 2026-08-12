package dq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Lineage event constants mirror the Python openlineage-python emitter.
const (
	producer       = "debezium_stream/dq-runner"
	jobNamespace   = "cdc-stand"
	pgNamespace    = "postgres://postgres-source:5432"
	icebergNS      = "iceberg://lakehouse"
	lineagePath    = "/api/v1/lineage"
	runEventStart  = "START"
	runEventFinish = "COMPLETE"
	runEventFail   = "FAIL"
)

type runEvent struct {
	EventType string        `json:"eventType"`
	EventTime string        `json:"eventTime"`
	Run       lineageRun    `json:"run"`
	Job       lineageJob    `json:"job"`
	Producer  string        `json:"producer"`
	Inputs    []lineageNode `json:"inputs,omitempty"`
	Outputs   []lineageNode `json:"outputs,omitempty"`
}

type lineageRun struct {
	RunID string `json:"runId"`
}

type lineageJob struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type lineageNode struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// LineageEmitter posts OpenLineage START/COMPLETE|FAIL events to Marquez for
// every DQ cycle. A missing or malformed OPENLINEAGE_URL disables lineage
// instead of killing the loop: lineage must never break the DQ metrics.
type LineageEmitter struct {
	client    *http.Client
	endpoint  string
	disabled  bool
	pgTables  []string
	iceTables []string
}

// NewLineageEmitter builds the emitter. It never returns an error: a bad URL
// only disables lineage (with a logged notice).
func NewLineageEmitter(rawURL string) *LineageEmitter {
	if rawURL == "" {
		fmt.Println("lineage disabled (OPENLINEAGE_URL not set)")
		return &LineageEmitter{disabled: true}
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		fmt.Printf("lineage disabled (%v)\n", err)
		return &LineageEmitter{disabled: true}
	}
	return &LineageEmitter{
		client:   &http.Client{Timeout: 10 * time.Second},
		endpoint: strings.TrimRight(u.String(), "/") + lineagePath,
	}
}

// Configure sets the input/output dataset lists for subsequent cycles.
func (l *LineageEmitter) Configure(pgTables, icebergTables []string) {
	l.pgTables = pgTables
	l.iceTables = icebergTables
}

// EmitCycle posts START then COMPLETE/FAIL for one DQ cycle.
func (l *LineageEmitter) EmitCycle(ctx context.Context, ok bool) {
	if l.disabled {
		return
	}
	runID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	finish := runEventFinish
	if !ok {
		finish = runEventFail
	}
	if err := l.post(ctx, runEvent{
		EventType: runEventStart,
		EventTime: now,
		Run:       lineageRun{runID},
		Job:       lineageJob{jobNamespace, "dq_check"},
		Producer:  producer,
	}); err != nil {
		fmt.Printf("lineage emit failed: %v\n", err)
		return
	}
	inputs := make([]lineageNode, 0, len(l.pgTables))
	for _, t := range l.pgTables {
		inputs = append(inputs, lineageNode{pgNamespace, t})
	}
	outputs := make([]lineageNode, 0, len(l.iceTables))
	for _, t := range l.iceTables {
		outputs = append(outputs, lineageNode{icebergNS, t})
	}
	if err := l.post(ctx, runEvent{
		EventType: finish,
		EventTime: now,
		Run:       lineageRun{runID},
		Job:       lineageJob{jobNamespace, "dq_check"},
		Producer:  producer,
		Inputs:    inputs,
		Outputs:   outputs,
	}); err != nil {
		fmt.Printf("lineage emit failed: %v\n", err)
	}
}

func (l *LineageEmitter) post(ctx context.Context, ev runEvent) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	// Marquez (as deployed in this stand) rejects the OpenLineage media types
	// (application/vnd.openlineage+json) with 415 and accepts plain
	// application/json; the body is a spec-compliant RunEvent either way.
	req.Header.Set("Content-Type", "application/json")
	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("marquez replied %s", resp.Status)
	}
	return nil
}
