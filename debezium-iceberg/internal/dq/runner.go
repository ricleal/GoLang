package dq

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/apache/iceberg-go/catalog"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds the Prometheus collectors of the DQ loop. The namespace "dq"
// reproduces the Python metric names (dq_check_ok, dq_rowcount_source, ...).
type Metrics struct {
	reg        *prometheus.Registry
	checkOK    *prometheus.GaugeVec
	rowSource  *prometheus.GaugeVec
	rowIceberg *prometheus.GaugeVec
	rowDiff    *prometheus.GaugeVec
	fresh      *prometheus.GaugeVec
	dup        *prometheus.GaugeVec
	missing    *prometheus.GaugeVec
	extra      *prometheus.GaugeVec
	null       *prometheus.GaugeVec
	cycles     prometheus.Counter
	errors     prometheus.Counter
}

// NewMetrics builds the collector set on a private registry.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{reg: reg}
	m.checkOK = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: "dq", Name: "check_ok", Help: "1 if check passed"}, []string{"table", "check"})
	m.rowSource = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: "dq", Name: "rowcount_source", Help: "rows in source table"}, []string{"table"})
	m.rowIceberg = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: "dq", Name: "rowcount_iceberg", Help: "live rows in iceberg table"}, []string{"table"})
	m.rowDiff = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: "dq", Name: "rowcount_diff", Help: "abs(source - iceberg live)"}, []string{"table"})
	m.fresh = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: "dq", Name: "freshness_seconds", Help: fmt.Sprintf("age of last iceberg snapshot (%v = none yet)", NO_SNAPSHOT)}, []string{"table"})
	m.dup = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: "dq", Name: "duplicate_pk_rows", Help: "extra rows sharing a pk"}, []string{"table"})
	m.missing = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: "dq", Name: "missing_keys", Help: "keys live in source but absent from iceberg"}, []string{"table"})
	m.extra = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: "dq", Name: "extra_keys", Help: "keys live in iceberg but absent from source"}, []string{"table"})
	m.null = prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: "dq", Name: "null_rate", Help: "fraction of NULLs in column"}, []string{"table", "column"})
	m.cycles = prometheus.NewCounter(prometheus.CounterOpts{Namespace: "dq", Name: "cycles_total", Help: "completed dq cycles"})
	m.errors = prometheus.NewCounter(prometheus.CounterOpts{Namespace: "dq", Name: "cycle_errors_total", Help: "errors during dq cycles"})
	reg.MustRegister(m.checkOK, m.rowSource, m.rowIceberg, m.rowDiff, m.fresh, m.dup, m.missing, m.extra, m.null, m.cycles, m.errors)
	return m
}

// Handler returns the /metrics handler for this collector set.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

// Run starts the Prometheus endpoint on :8000 and the DQ loop. It returns when
// ctx is cancelled.
func Run(ctx context.Context, cfg *Config) error {
	m := NewMetrics()
	srv := &http.Server{Addr: ":8000", Handler: m.Handler()}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics server: %v", err)
		}
	}()
	defer srv.Close()

	lineage := NewLineageEmitter(cfg.LineageURL)
	iceTables := make([]string, len(cfg.Tables))
	for i, t := range cfg.Tables {
		iceTables[i] = strings.Join(IcebergIdent(cfg, t), ".")
	}
	lineage.Configure(cfg.Tables, iceTables)

	log.Printf("dq-runner up (interval %.0fs, tables %v)", cfg.IntervalSec, cfg.Tables)
	for {
		m.runCycle(ctx, cfg, lineage)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(cfg.IntervalSec * float64(time.Second))):
		}
	}
}

func (m *Metrics) runCycle(ctx context.Context, cfg *Config, lineage *LineageEmitter) {
	cycleOK := true

	cat, err := LoadCatalog(ctx, cfg)
	if err != nil {
		// the catalog itself is unreachable: nothing was checked this cycle.
		// Without this every dq_check_ok keeps its last green value and the
		// stand reports perfect health with the catalog down.
		m.errors.Inc()
		cycleOK = false
		m.zeroAllChecks(cfg)
		log.Printf("cycle error: catalog: %v", err)
	} else if conn, err := pgx.Connect(ctx, cfg.PGDSN); err != nil {
		m.errors.Inc()
		cycleOK = false
		m.zeroAllChecks(cfg)
		log.Printf("cycle error: postgres: %v", err)
	} else {
		defer conn.Close(ctx)
		for _, pgTable := range cfg.Tables {
			if err := m.checkTable(ctx, cfg, cat, conn, pgTable); err != nil {
				// every check for this table is now unknown, not just the one
				// that raised - leaving the others green is how a fully
				// unreadable table shows up as "1 failed check".
				m.errors.Inc()
				cycleOK = false
				for _, name := range cfg.ChecksFor(pgTable) {
					m.checkOK.WithLabelValues(pgTable, name).Set(0)
				}
				log.Printf("table %s: %v", pgTable, err)
			}
		}
	}

	m.cycles.Inc()
	lineage.EmitCycle(ctx, cycleOK)
}

func (m *Metrics) zeroAllChecks(cfg *Config) {
	for _, pgTable := range cfg.Tables {
		for _, name := range cfg.ChecksFor(pgTable) {
			m.checkOK.WithLabelValues(pgTable, name).Set(0)
		}
	}
}

// checkTable runs all checks for one table and writes the gauges.
func (m *Metrics) checkTable(ctx context.Context, cfg *Config, cat catalog.Catalog, conn *pgx.Conn, pgTable string) error {
	pk := cfg.PKOverrides[pgTable]
	if pk == "" {
		pk = "id"
	}
	column := cfg.NullChecks[pgTable]

	schema, tableName, found := strings.Cut(pgTable, ".")
	if !found {
		return fmt.Errorf("pg table %q must be schema.table", pgTable)
	}
	pkIdent := pgx.Identifier{pk}.Sanitize()
	tblIdent := pgx.Identifier{schema, tableName}.Sanitize()

	sourceKeys, err := queryKeys(ctx, conn, "SELECT "+pkIdent+" FROM "+tblIdent)
	if err != nil {
		return fmt.Errorf("source query: %w", err)
	}
	src := len(sourceKeys)

	scanCtx, cancel := CheckTimeout(ctx, cfg)
	defer cancel()
	lakeRows, err := ScanRows(scanCtx, cfg, cat, pgTable, pk, column)
	if err != nil {
		return fmt.Errorf("iceberg scan: %w", err)
	}
	live := LiveRows(lakeRows)

	// rowcount
	rc := RowcountCheck(src, len(live), cfg.Tolerance)
	m.rowSource.WithLabelValues(pgTable).Set(float64(rc.Source))
	m.rowIceberg.WithLabelValues(pgTable).Set(float64(rc.Target))
	m.rowDiff.WithLabelValues(pgTable).Set(float64(rc.Diff))
	m.checkOK.WithLabelValues(pgTable, "rowcount").Set(boolFloat(rc.OK))

	// freshness
	ms, has, err := LastSnapshotMs(ctx, cfg, cat, pgTable)
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}
	var fresh float64
	if has {
		fresh = FreshnessSeconds(ms, time.Now().UnixMilli())
	} else {
		fresh = NO_SNAPSHOT
	}
	m.fresh.WithLabelValues(pgTable).Set(fresh)
	freshOK := 0 <= fresh && fresh <= cfg.FreshnessMax
	m.checkOK.WithLabelValues(pgTable, "freshness").Set(boolFloat(freshOK))

	// duplicates (among live rows; should not occur in upsert mode)
	dup := DuplicatePkRows(live)
	m.dup.WithLabelValues(pgTable).Set(float64(dup))
	m.checkOK.WithLabelValues(pgTable, "duplicates").Set(boolFloat(dup == 0))

	// key-level reconciliation
	keys := KeyDiff(sourceKeys, liveKeys(live))
	m.missing.WithLabelValues(pgTable).Set(float64(keys.Missing))
	m.extra.WithLabelValues(pgTable).Set(float64(keys.Extra))
	m.checkOK.WithLabelValues(pgTable, "keys").Set(boolFloat(keys.OK))

	// null-rate
	if column != "" {
		rate := NullRate(live)
		m.null.WithLabelValues(pgTable, column).Set(rate)
		m.checkOK.WithLabelValues(pgTable, "nulls").Set(boolFloat(rate == 0))
	}
	return nil
}

func queryKeys(ctx context.Context, conn *pgx.Conn, sql string) ([]any, error) {
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []any
	for rows.Next() {
		var v any
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		keys = append(keys, v)
	}
	return keys, rows.Err()
}

func liveKeys(live []Row) []any {
	keys := make([]any, 0, len(live))
	for _, r := range live {
		if r.Key != nil {
			keys = append(keys, r.Key)
		}
	}
	return keys
}

func boolFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
