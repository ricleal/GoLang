// Package dq implements the data-quality loop of the CDC stand:
// Postgres vs Iceberg checks exported as Prometheus gauges, optional
// OpenLineage lineage to Marquez, and Iceberg snapshot maintenance.
//
// It is the Go port of the Python `dq/` module of debezium-iceberg-lab.
package dq

import (
	"os"
	"strconv"
	"strings"
)

// Config holds every knob the DQ loop and the maintenance job read from the
// environment. The variable names intentionally mirror the Python lab
// (DQ_*, MAINT_*, OPENLINEAGE_URL) so the docker-compose/.env stay portable.
type Config struct {
	PGDSN        string
	Tables       []string
	Namespace    string
	TopicPrefix  string
	IntervalSec  float64
	Tolerance    int
	FreshnessMax float64
	PGTimeoutSec int

	NullChecks  map[string]string // table -> column
	PKOverrides map[string]string // table -> primary-key column
	IcebergURI  string
	IcebergWh   string
	S3Endpoint  string
	S3Key       string
	S3Secret    string
	S3Region    string
	HTTPTimeout int // per-table scan timeout in ms
	LineageURL  string

	MaintMaxAgeHours float64
	MaintRetainLast  int
}

func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envFloatOr(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// splitPairs parses "a:b,c:d" into a map, skipping empty entries.
func splitPairs(raw string) map[string]string {
	out := map[string]string{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 {
			continue
		}
		out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return out
}

// LoadConfig reads the DQ configuration from the environment.
func LoadConfig() *Config {
	tables := []string{}
	for _, t := range strings.Split(envOr("DQ_TABLES", ""), ",") {
		if t = strings.TrimSpace(t); t != "" {
			tables = append(tables, t)
		}
	}
	return &Config{
		PGDSN:            os.Getenv("DQ_PG_DSN"),
		Tables:           tables,
		Namespace:        envOr("DQ_ICEBERG_NAMESPACE", "cdc"),
		TopicPrefix:      envOr("DQ_TOPIC_PREFIX", "dbz"),
		IntervalSec:      envFloatOr("DQ_INTERVAL_SECONDS", 30),
		Tolerance:        envIntOr("DQ_ROWCOUNT_TOLERANCE", 25),
		FreshnessMax:     envFloatOr("DQ_FRESHNESS_MAX_SECONDS", 120),
		PGTimeoutSec:     envIntOr("DQ_PG_CONNECT_TIMEOUT", 10),
		NullChecks:       splitPairs(envOr("DQ_NULL_CHECKS", "")),
		PKOverrides:      splitPairs(envOr("DQ_PK_OVERRIDES", "inventory.products_on_hand:product_id")),
		IcebergURI:       os.Getenv("DQ_ICEBERG_URI"),
		IcebergWh:        envOr("DQ_ICEBERG_WAREHOUSE", "lakehouse"),
		S3Endpoint:       envOr("DQ_S3_ENDPOINT", ""),
		S3Key:            os.Getenv("DQ_S3_KEY"),
		S3Secret:         os.Getenv("DQ_S3_SECRET"),
		S3Region:         envOr("DQ_S3_REGION", "local-01"),
		HTTPTimeout:      envIntOr("DQ_HTTP_TIMEOUT_MS", 15000),
		LineageURL:       os.Getenv("OPENLINEAGE_URL"),
		MaintMaxAgeHours: envFloatOr("MAINT_MAX_AGE_HOURS", 2),
		MaintRetainLast:  envIntOr("MAINT_RETAIN_LAST", 20),
	}
}

// ChecksFor returns the checks a table actually runs. A check a table never
// runs must not be reset on error, otherwise its gauge would sit at 0 with no
// way to get back to a real value.
func (c *Config) ChecksFor(pgTable string) []string {
	checks := []string{"rowcount", "freshness", "duplicates", "keys"}
	if _, ok := c.NullChecks[pgTable]; ok {
		checks = append(checks, "nulls")
	}
	return checks
}
