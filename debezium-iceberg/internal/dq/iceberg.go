package dq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	_ "github.com/apache/iceberg-go/catalog/rest" // registers the "rest" catalog type via init()
	_ "github.com/apache/iceberg-go/io/gocloud"   // registers the S3/GCS/Azure file-IO schemes
	"github.com/apache/iceberg-go/table"
)

// NO_SNAPSHOT is the freshness sentinel: the table has no snapshot yet. +Inf
// would pin every max() panel forever, so -1 signals "none".
const NO_SNAPSHOT = -1.0

// CatalogProps maps the DQ env config onto the iceberg-go REST catalog
// properties. Path-style addressing is implied automatically whenever
// s3.endpoint is set (MinIO), so no extra flag is needed.
func CatalogProps(cfg *Config) iceberg.Properties {
	return iceberg.Properties{
		"type":                 "rest",
		"uri":                  cfg.IcebergURI,
		"warehouse":            cfg.IcebergWh,
		"s3.endpoint":          cfg.S3Endpoint,
		"s3.access-key-id":     cfg.S3Key,
		"s3.secret-access-key": cfg.S3Secret,
		"s3.region":            cfg.S3Region,
	}
}

// LoadCatalog returns a REST catalog pointing at Lakekeeper.
func LoadCatalog(ctx context.Context, cfg *Config) (catalog.Catalog, error) {
	return catalog.Load(ctx, "lakekeeper", CatalogProps(cfg))
}

// IcebergIdent maps a Postgres table name to its Iceberg table identifier,
// e.g. "inventory.customers" -> {cdc, dbz_inventory_customers}.
func IcebergIdent(cfg *Config, pgTable string) table.Identifier {
	name := fmt.Sprintf("%s_%s", cfg.TopicPrefix, strings.ReplaceAll(pgTable, ".", "_"))
	return table.Identifier{cfg.Namespace, name}
}

// ScanRows scans the lake table projecting only the columns the DQ loop needs
// (soft-delete flag, primary key, and optionally the null-checked column).
// iceberg-go applies equality and positional deletes during the scan, so each
// returned row is the live version for that key.
func ScanRows(ctx context.Context, cfg *Config, cat catalog.Catalog, pgTable, pkCol, nullCol string) ([]Row, error) {
	tbl, err := cat.LoadTable(ctx, IcebergIdent(cfg, pgTable))
	if err != nil {
		return nil, err
	}
	wanted := []string{"__deleted", pkCol}
	hasNull := nullCol != "" && nullCol != pkCol
	if hasNull {
		wanted = append(wanted, nullCol)
	}
	at, err := tbl.Scan(table.WithSelectedFields(wanted...)).ToArrowTable(ctx)
	if err != nil {
		return nil, err
	}
	defer at.Release()
	return rowsFromArrow(at, pkCol, nullCol, hasNull)
}

func rowsFromArrow(at arrow.Table, pkCol, nullCol string, hasNull bool) ([]Row, error) {
	schema := at.Schema()
	deletedIdx := fieldIndex(schema, "__deleted")
	if deletedIdx < 0 {
		return nil, fmt.Errorf("lake table has no __deleted column")
	}
	pkIdx := fieldIndex(schema, pkCol)
	if pkIdx < 0 {
		return nil, fmt.Errorf("lake table has no %q column", pkCol)
	}
	nullIdx := -1
	if hasNull {
		nullIdx = fieldIndex(schema, nullCol)
		if nullIdx < 0 {
			nullIdx = -1
			hasNull = false
		}
	}

	tr := array.NewTableReader(at, 8192)
	defer tr.Release()

	var rows []Row
	for tr.Next() {
		rec := tr.Record()
		delArr := rec.Column(deletedIdx)
		pkArr := rec.Column(pkIdx)
		var nullArr arrow.Array
		if hasNull {
			nullArr = rec.Column(nullIdx)
		}
		for i := 0; i < int(rec.NumRows()); i++ {
			rows = append(rows, Row{
				Deleted: deletedAt(delArr, i),
				Key:     valueAt(pkArr, i),
				Null:    hasNull && nullArr.IsNull(i),
			})
		}
	}
	if err := tr.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

func fieldIndex(schema *arrow.Schema, name string) int {
	for i, f := range schema.Fields() {
		if f.Name == name {
			return i
		}
	}
	return -1
}

// deletedAt reads the soft-delete flag from an arrow array. The sink may write
// it as boolean or as "true"/"false" string depending on the schema.
func deletedAt(a arrow.Array, i int) bool {
	if a.IsNull(i) {
		return false
	}
	switch arr := a.(type) {
	case *array.Boolean:
		return arr.Value(i)
	case *array.String:
		return arr.Value(i) == "true"
	case *array.LargeString:
		return arr.Value(i) == "true"
	case *array.FixedSizeBinary:
		return string(arr.Value(i)) == "true"
	}
	return false
}

// valueAt reads a single cell as a plain Go value (nil for NULL). Only the
// types that show up in the inventory tables are handled; anything exotic is
// rendered through its string form.
func valueAt(a arrow.Array, i int) any {
	if a.IsNull(i) {
		return nil
	}
	switch arr := a.(type) {
	case *array.Boolean:
		return arr.Value(i)
	case *array.Int8:
		return int64(arr.Value(i))
	case *array.Int16:
		return int64(arr.Value(i))
	case *array.Int32:
		return int64(arr.Value(i))
	case *array.Int64:
		return arr.Value(i)
	case *array.Uint8:
		return int64(arr.Value(i))
	case *array.Uint16:
		return int64(arr.Value(i))
	case *array.Uint32:
		return int64(arr.Value(i))
	case *array.Uint64:
		return int64(arr.Value(i))
	case *array.Float32:
		return float64(arr.Value(i))
	case *array.Float64:
		return arr.Value(i)
	case *array.String:
		return arr.Value(i)
	case *array.LargeString:
		return arr.Value(i)
	case *array.Date32:
		return int64(arr.Value(i))
	case *array.Date64:
		return int64(arr.Value(i))
	case *array.Timestamp:
		return arr.Value(i)
	case *array.Decimal128:
		return arr.Value(i)
	}
	return a.ValueStr(i)
}

// LastSnapshotMs returns the current snapshot timestamp (ms) of a lake table.
// ok is false when the table has no snapshot yet.
func LastSnapshotMs(ctx context.Context, cfg *Config, cat catalog.Catalog, pgTable string) (ms int64, ok bool, err error) {
	tbl, err := cat.LoadTable(ctx, IcebergIdent(cfg, pgTable))
	if err != nil {
		return 0, false, err
	}
	snap := tbl.CurrentSnapshot()
	if snap == nil {
		return 0, false, nil
	}
	return snap.TimestampMs, true, nil
}

// CheckTimeout returns a per-table context: a hung (not refused) MinIO or
// Lakekeeper must not stall the DQ loop forever while the metrics endpoint
// keeps serving the last good values.
func CheckTimeout(parent context.Context, cfg *Config) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, time.Duration(cfg.HTTPTimeout)*time.Millisecond)
}

// LiveCount returns the number of live (non soft-deleted) rows of a table.
// Used by the integration tests. The pk honours the DQ_PK_OVERRIDES config so
// tables whose primary key is not `id` (e.g. products_on_hand) scan cleanly.
func LiveCount(ctx context.Context, cfg *Config, cat catalog.Catalog, pgTable string) (int, error) {
	pk := cfg.PKOverrides[pgTable]
	if pk == "" {
		pk = "id"
	}
	rows, err := ScanRows(ctx, cfg, cat, pgTable, pk, "")
	if err != nil {
		return 0, err
	}
	return len(LiveRows(rows)), nil
}

// LiveValues returns the values of a column for the live rows of a table.
// Used by the integration tests (e.g. to confirm a just-inserted email row
// has propagated to the lake). The pk is projected too: the scan must apply
// equality-delete files, which requires the delete column (the pk) to be part
// of the projection.
func LiveValues(ctx context.Context, cfg *Config, cat catalog.Catalog, pgTable, column string) ([]any, error) {
	tbl, err := cat.LoadTable(ctx, IcebergIdent(cfg, pgTable))
	if err != nil {
		return nil, err
	}
	pk := cfg.PKOverrides[pgTable]
	if pk == "" {
		pk = "id"
	}
	sel := []string{"__deleted", pk}
	if column != pk {
		sel = append(sel, column)
	}
	at, err := tbl.Scan(table.WithSelectedFields(sel...)).ToArrowTable(ctx)
	if err != nil {
		return nil, err
	}
	defer at.Release()
	return valuesFromArrow(at, column)
}

func valuesFromArrow(at arrow.Table, column string) ([]any, error) {
	schema := at.Schema()
	colIdx := fieldIndex(schema, column)
	if colIdx < 0 {
		return nil, fmt.Errorf("lake table has no %q column", column)
	}
	deletedIdx := fieldIndex(schema, "__deleted") // may be absent; then all rows are live

	tr := array.NewTableReader(at, 8192)
	defer tr.Release()
	var out []any
	for tr.Next() {
		rec := tr.Record()
		col := rec.Column(colIdx)
		var del arrow.Array
		if deletedIdx >= 0 {
			del = rec.Column(deletedIdx)
		}
		for i := 0; i < int(rec.NumRows()); i++ {
			if del == nil || !deletedAt(del, i) {
				out = append(out, valueAt(col, i))
			}
		}
	}
	return out, tr.Err()
}
