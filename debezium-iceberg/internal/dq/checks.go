package dq

import (
	"fmt"
	"math"
)

// Row is one scanned lake row projected to the fields the DQ loop needs.
// Deleted carries the soft-delete flag (`__deleted`); Key is the primary-key
// value (nil when NULL); Null is true when the null-checked column is NULL
// (only meaningful when a null column was requested).
type Row struct {
	Deleted bool
	Key     any
	Null    bool
}

// RowcountResult mirrors checks.rowcount_check in the Python lab.
type RowcountResult struct {
	Source int
	Target int
	Diff   int
	OK     bool
}

// RowcountCheck compares source and lake row counts within a tolerance.
func RowcountCheck(source, target, tolerance int) RowcountResult {
	diff := int(math.Abs(float64(source - target)))
	return RowcountResult{source, target, diff, diff <= tolerance}
}

// LiveRows drops soft-deleted rows (`__deleted` = true), mirroring
// `SELECT * FROM t WHERE COALESCE(CAST(__deleted AS VARCHAR),'false') <> 'true'`.
func LiveRows(rows []Row) []Row {
	live := make([]Row, 0, len(rows))
	for _, r := range rows {
		if !r.Deleted {
			live = append(live, r)
		}
	}
	return live
}

// DuplicatePkRows counts live rows that share a primary key beyond the first
// (the sink writes upserts with equality deletes, so >1 live row with the same
// key means a lost equality-delete file).
func DuplicatePkRows(live []Row) int {
	seen := make(map[string]struct{})
	extra := 0
	for _, r := range live {
		if r.Key == nil {
			continue
		}
		k := keyString(r.Key)
		if _, ok := seen[k]; ok {
			extra++
		} else {
			seen[k] = struct{}{}
		}
	}
	return extra
}

// KeyDiffResult mirrors checks.KeyDiffResult in the Python lab.
type KeyDiffResult struct {
	Missing int // live in the source, absent from the lake
	Extra   int // live in the lake, absent from the source
	OK      bool
}

// KeyDiff compares key sets, not counts. A row that vanishes from the lake
// while alive in the source moves rowcount by one, which hides inside the
// tolerance; comparing keys catches it exactly.
//
// Keys above the highest key the lake has seen are excluded: those are rows
// the sink has simply not committed yet, so counting them would make the check
// fire on ordinary batching lag instead of on real divergence.
func KeyDiff(sourceKeys, lakeKeys []any) KeyDiffResult {
	sourceInt, allSourceInt := toInt64Set(sourceKeys)
	lakeInt, allLakeInt := toInt64Set(lakeKeys)
	if allSourceInt && allLakeInt {
		return keyDiffInt(sourceInt, lakeInt)
	}
	return keyDiffStr(toStringSet(sourceKeys), toStringSet(lakeKeys))
}

func keyDiffInt(source, lake map[int64]struct{}) KeyDiffResult {
	if len(lake) == 0 {
		return KeyDiffResult{0, 0, true}
	}
	horizon := int64(math.MinInt64)
	for k := range lake {
		if k > horizon {
			horizon = k
		}
	}
	missing := 0
	for k := range source {
		if k <= horizon {
			if _, ok := lake[k]; !ok {
				missing++
			}
		}
	}
	extra := 0
	for k := range lake {
		if _, ok := source[k]; !ok {
			extra++
		}
	}
	return KeyDiffResult{missing, extra, missing == 0}
}

func keyDiffStr(source, lake map[string]struct{}) KeyDiffResult {
	if len(lake) == 0 {
		return KeyDiffResult{0, 0, true}
	}
	horizon := ""
	for k := range lake {
		if k > horizon {
			horizon = k
		}
	}
	missing := 0
	for k := range source {
		if k <= horizon {
			if _, ok := lake[k]; !ok {
				missing++
			}
		}
	}
	extra := 0
	for k := range lake {
		if _, ok := source[k]; !ok {
			extra++
		}
	}
	return KeyDiffResult{missing, extra, missing == 0}
}

// NullRate is the fraction of live rows whose watched column is NULL.
func NullRate(live []Row) float64 {
	if len(live) == 0 {
		return 0.0
	}
	nulls := 0
	for _, r := range live {
		if r.Null {
			nulls++
		}
	}
	return float64(nulls) / float64(len(live))
}

// FreshnessSeconds is the age of the last iceberg snapshot, clamped to >= 0.
func FreshnessSeconds(lastSnapshotTsMs, nowMs int64) float64 {
	return math.Max(0, float64(nowMs-lastSnapshotTsMs)/1000.0)
}

// ---- key helpers ----

func toInt64Set(keys []any) (map[int64]struct{}, bool) {
	out := make(map[int64]struct{}, len(keys))
	for _, k := range keys {
		if k == nil {
			continue
		}
		v, ok := keyInt64(k)
		if !ok {
			return nil, false
		}
		out[v] = struct{}{}
	}
	return out, true
}

func toStringSet(keys []any) map[string]struct{} {
	out := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if k == nil {
			continue
		}
		out[keyString(k)] = struct{}{}
	}
	return out
}

// keyInt64 normalizes integer-like Go values to int64.
func keyInt64(v any) (int64, bool) {
	switch k := v.(type) {
	case int:
		return int64(k), true
	case int8:
		return int64(k), true
	case int16:
		return int64(k), true
	case int32:
		return int64(k), true
	case int64:
		return k, true
	case uint:
		return int64(k), true
	case uint8:
		return int64(k), true
	case uint16:
		return int64(k), true
	case uint32:
		return int64(k), true
	case uint64:
		return int64(k), true
	case float32:
		f := float64(k)
		if f == math.Trunc(f) {
			return int64(f), true
		}
	case float64:
		if k == math.Trunc(k) {
			return int64(k), true
		}
	}
	return 0, false
}

// keyString renders a key as a stable comparable string.
func keyString(v any) string {
	switch k := v.(type) {
	case string:
		return k
	default:
		return fmt.Sprint(k)
	}
}
