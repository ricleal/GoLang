// Package integration contains the stand-level tests run by `make verify`
// inside the compose `verify` service. They need the full stand up
// (postgres-source, minio, lakekeeper, debezium, dq-runner, prometheus,
// grafana, marquez); services that are down make the test skip rather than
// fail, matching the Python test_core.py / test_observability.py behaviour.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/table"
	"github.com/jackc/pgx/v5"

	"debezium-iceberg/internal/dq"
)

var (
	ctx        = context.Background()
	cfg        = dq.LoadConfig()
	ns         = table.Identifier{cfg.Namespace}
	httpClient = &http.Client{Timeout: 5 * time.Second}
)

func httpGet(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	resp, err := httpClient.Get(url)
	if err != nil {
		t.Skipf("%s unreachable - the owning service is not up (%v)", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

func waitUntil(t *testing.T, timeout time.Duration, desc string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		ok := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					lastErr = fmt.Errorf("%v", r)
				}
			}()
			ok = fn()
		}()
		if ok {
			return
		}
		time.Sleep(3 * time.Second)
	}
	if lastErr != nil {
		t.Fatalf("timeout waiting for: %s; last error: %v", desc, lastErr)
	}
	t.Fatalf("timeout waiting for: %s", desc)
}

// lakeTables returns the set of table names in the CDC namespace.
func lakeTables(t *testing.T, cat catalog.Catalog) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	for ident, err := range cat.ListTables(ctx, ns) {
		if err != nil {
			continue
		}
		names[ident[len(ident)-1]] = true
	}
	return names
}

func catalogOrSkip(t *testing.T) catalog.Catalog {
	t.Helper()
	cat, err := dq.LoadCatalog(ctx, cfg)
	if err != nil {
		t.Skipf("catalog unreachable: %v", err)
	}
	return cat
}

// ---- core ----

func TestLakekeeperWarehouseConfigured(t *testing.T) {
	resp, _ := httpGet(t, fmt.Sprintf("http://lakekeeper:8181/catalog/v1/config?warehouse=%s", cfg.IcebergWh))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestIcebergTablesCreated(t *testing.T) {
	cat := catalogOrSkip(t)
	expected := map[string]bool{
		"dbz_inventory_customers":        true,
		"dbz_inventory_products":         true,
		"dbz_inventory_orders":           true,
		"dbz_inventory_products_on_hand": true,
	}
	waitUntil(t, 300*time.Second, "debezium creates iceberg tables", func() bool {
		got := map[string]bool{}
		for ident, err := range cat.ListTables(ctx, ns) {
			if err != nil {
				return false
			}
			got[ident[len(ident)-1]] = true
		}
		for name := range expected {
			if !got[name] {
				return false
			}
		}
		return true
	})
}

func TestOffsetsStoredInIceberg(t *testing.T) {
	cat := catalogOrSkip(t)
	waitUntil(t, 120*time.Second, "debezium offset table in iceberg", func() bool {
		for ident, err := range cat.ListTables(ctx, ns) {
			if err != nil {
				return false
			}
			if strings.Contains(ident[len(ident)-1], "offset") {
				return true
			}
		}
		return false
	})
}

func TestRowcountsReconcile(t *testing.T) {
	cat := catalogOrSkip(t)
	conn, err := pgx.Connect(ctx, cfg.PGDSN)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	defer conn.Close(ctx)

	tables := []string{"customers", "products", "orders", "products_on_hand"}
	waitUntil(t, 180*time.Second, "rowcounts reconcile within tolerance", func() bool {
		for _, name := range tables {
			var src int
			if err := conn.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM inventory.%s", name)).Scan(&src); err != nil {
				return false
			}
			live, err := dq.LiveCount(ctx, cfg, cat, "inventory."+name)
			if err != nil {
				return false
			}
			if abs(src-live) > cfg.Tolerance {
				return false
			}
		}
		return true
	})
}

func TestInsertPropagates(t *testing.T) {
	cat := catalogOrSkip(t)
	conn, err := pgx.Connect(ctx, cfg.PGDSN)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	defer conn.Close(ctx)

	email := fmt.Sprintf("verify-%x@example.com", time.Now().UnixNano())
	if _, err := conn.Exec(ctx,
		"INSERT INTO inventory.customers (first_name, last_name, email) VALUES ('Ver', 'Ify', $1)", email); err != nil {
		t.Fatalf("insert: %v", err)
	}
	t.Cleanup(func() {
		// every `make verify` used to leave one more verify-* customer behind
		_, _ = conn.Exec(ctx, "DELETE FROM inventory.customers WHERE email = $1", email)
	})

	waitUntil(t, 120*time.Second, "insert visible in iceberg", func() bool {
		vals, err := dq.LiveValues(ctx, cfg, cat, "inventory.customers", "email")
		if err != nil {
			return false
		}
		for _, v := range vals {
			if v == email {
				return true
			}
		}
		return false
	})
}

// ---- observability ----

func TestPrometheusTargetsUp(t *testing.T) {
	waitUntil(t, 90*time.Second, "prometheus targets healthy", func() bool {
		resp, body := httpGet(t, "http://prometheus:9090/api/v1/targets")
		if resp.StatusCode != 200 {
			return false
		}
		var payload struct {
			Data struct {
				ActiveTargets []struct {
					Labels map[string]string `json:"labels"`
					Health string            `json:"health"`
				} `json:"activeTargets"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			return false
		}
		health := map[string]string{}
		for _, tgt := range payload.Data.ActiveTargets {
			health[tgt.Labels["job"]] = tgt.Health
		}
		return health["dq-runner"] == "up" && health["debezium-connector"] == "up" && health["debezium-quarkus"] == "up"
	})
}

func TestDQMetricsExposed(t *testing.T) {
	// require an actual labelled sample, not just the #HELP/#TYPE banner
	waitUntil(t, 90*time.Second, "dq-runner reports a rowcount sample", func() bool {
		resp, body := httpGet(t, "http://dq-runner:8000/metrics")
		if resp.StatusCode != 200 {
			return false
		}
		return strings.Contains(body, "dq_rowcount_diff{")
	})
}

func TestDQKeyCheckReported(t *testing.T) {
	// the keys check is the only one that sees a row missing from the lake
	waitUntil(t, 90*time.Second, "dq-runner reports the keys check", func() bool {
		resp, body := httpGet(t, "http://dq-runner:8000/metrics")
		if resp.StatusCode != 200 {
			return false
		}
		return strings.Contains(body, `dq_check_ok{check="keys"`)
	})
}

func TestGrafanaDashboardProvisioned(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://grafana:3000/api/dashboards/uid/cdc-stand", nil)
	req.SetBasicAuth("admin", "admin")
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Skipf("grafana unreachable: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMarquezHasDQJob(t *testing.T) {
	waitUntil(t, 120*time.Second, "dq_check job appears in marquez", func() bool {
		resp, body := httpGet(t, "http://marquez:5000/api/v1/namespaces/cdc-stand/jobs")
		if resp.StatusCode != 200 {
			return false
		}
		var payload struct {
			Jobs []struct {
				Name string `json:"name"`
			} `json:"jobs"`
		}
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			return false
		}
		for _, j := range payload.Jobs {
			if j.Name == "dq_check" {
				return true
			}
		}
		return false
	})
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
