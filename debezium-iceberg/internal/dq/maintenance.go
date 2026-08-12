package dq

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/table"
)

// MetadataProps are idempotently ensured on every table:
//  1. enable auto-trimming of old metadata.json files
//  2. cap how many metadata versions are kept
//
// Physical bin-packing of small parquet files and row-level deduplication are
// intentionally out of scope (as in the Python lab) - in production that is an
// offline Trino/Spark job.
var MetadataProps = iceberg.Properties{
	"write.metadata.delete-after-commit.enabled": "true",
	"write.metadata.previous-versions-max":       "25",
}

// Maintain expires old snapshots on every table of the configured namespace
// and ensures the metadata retention properties. It is the `make maintain`
// job; sink commits run concurrently, so a conflict on one table is skipped,
// not fatal.
func Maintain(ctx context.Context, cfg *Config) error {
	cat, err := LoadCatalog(ctx, cfg)
	if err != nil {
		return err
	}
	for ident, err := range cat.ListTables(ctx, table.Identifier{cfg.Namespace}) {
		if err != nil {
			log.Printf("list tables: %v", err)
			continue
		}
		if err := maintainTable(ctx, cat, ident, cfg); err != nil {
			log.Printf("%s: skipped (%v)", strings.Join(ident, "."), err)
		}
	}
	return nil
}

func maintainTable(ctx context.Context, cat catalog.Catalog, ident table.Identifier, cfg *Config) error {
	tbl, err := cat.LoadTable(ctx, ident)
	if err != nil {
		return err
	}
	before := len(tbl.Metadata().Snapshots())

	txn := tbl.NewTransaction()
	missing := iceberg.Properties{}
	for k, v := range MetadataProps {
		if tbl.Properties()[k] != v {
			missing[k] = v
		}
	}
	if len(missing) > 0 {
		if err := txn.SetProperties(missing); err != nil {
			return err
		}
	}
	// older than MAINT_MAX_AGE_HOURS, but the last MAINT_RETAIN_LAST snapshots
	// are never touched (the current snapshot and refs are protected by the
	// expire logic itself).
	if err := txn.ExpireSnapshots(
		table.WithRetainLast(cfg.MaintRetainLast),
		table.WithOlderThan(time.Duration(cfg.MaintMaxAgeHours*float64(time.Hour))),
	); err != nil {
		return err
	}
	if _, err := txn.Commit(ctx); err != nil {
		return err
	}

	reloaded, err := cat.LoadTable(ctx, ident)
	if err != nil {
		return err
	}
	after := len(reloaded.Metadata().Snapshots())
	fmt.Printf("%s: snapshots %d -> %d\n", strings.Join(ident, "."), before, after)
	return nil
}
