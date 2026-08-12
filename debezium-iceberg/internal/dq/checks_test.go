package dq

import (
	"math"
	"testing"
)

// Pure check functions only - no stand needed (the Python counterpart is
// tests/test_checks_unit.py, runnable with `go test ./internal/dq/`).

func TestRowcountWithinTolerance(t *testing.T) {
	res := RowcountCheck(100, 95, 10)
	if !res.OK {
		t.Fatalf("expected ok, got %+v", res)
	}
	if res.Diff != 5 {
		t.Fatalf("expected diff 5, got %d", res.Diff)
	}
}

func TestRowcountExceedsTolerance(t *testing.T) {
	if RowcountCheck(100, 50, 10).OK {
		t.Fatal("expected check to fail")
	}
}

func TestLiveRowsFiltersSoftDeletes(t *testing.T) {
	rows := []Row{
		{Deleted: false, Key: int64(1)},
		{Deleted: true, Key: int64(2)},
		{Deleted: false, Key: int64(3)},
	}
	if got := len(LiveRows(rows)); got != 2 {
		t.Fatalf("expected 2 live rows, got %d", got)
	}
}

func TestLiveRowsWithoutColumnIsPassthrough(t *testing.T) {
	rows := []Row{{Key: int64(1)}, {Key: int64(2)}}
	if got := len(LiveRows(rows)); got != 2 {
		t.Fatalf("expected 2 rows, got %d", got)
	}
}

func TestDuplicatePkRows(t *testing.T) {
	rows := []Row{{Key: int64(1)}, {Key: int64(1)}, {Key: int64(2)}, {Key: int64(3)}, {Key: int64(3)}, {Key: int64(3)}}
	if got := DuplicatePkRows(rows); got != 3 {
		t.Fatalf("expected 3 duplicate extras, got %d", got)
	}
}

func TestDuplicatePkRowsEmptyTable(t *testing.T) {
	if got := DuplicatePkRows(nil); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestDuplicatePkRowsMissingColumn(t *testing.T) {
	// a table whose PK is not `id` (and was not scanned) must not blow up
	rows := []Row{{Key: nil}, {Key: nil}}
	if got := DuplicatePkRows(rows); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestKeyDiffCatchesRowMissingFromLake(t *testing.T) {
	// id=2 exists in the source and not in the lake, and it is not in the tail:
	// exactly the delete-then-reinsert loss measured in notebook 05
	res := KeyDiff([]any{1, 2, 3}, []any{1, 3})
	if res.Missing != 1 {
		t.Fatalf("expected missing=1, got %+v", res)
	}
	if res.OK {
		t.Fatal("expected check to fail")
	}
}

func TestKeyDiffIgnoresUnreplicatedTail(t *testing.T) {
	// rows the sink has not committed yet are lag, not divergence
	res := KeyDiff([]any{1, 2, 3, 4, 5}, []any{1, 2, 3})
	if res.Missing != 0 {
		t.Fatalf("expected missing=0, got %+v", res)
	}
	if !res.OK {
		t.Fatal("expected check to pass")
	}
}

func TestKeyDiffReportsExtraKeysWithoutFailing(t *testing.T) {
	// a key deleted in the source but still live in the lake is ordinary lag
	res := KeyDiff([]any{1, 2}, []any{1, 2, 3})
	if res.Extra != 1 {
		t.Fatalf("expected extra=1, got %+v", res)
	}
	if !res.OK {
		t.Fatal("expected check to pass")
	}
}

func TestKeyDiffEmptyLakeIsNotAFailure(t *testing.T) {
	res := KeyDiff([]any{1, 2}, nil)
	if !res.OK {
		t.Fatalf("expected ok, got %+v", res)
	}
}

func TestKeyDiffInt32VsInt64(t *testing.T) {
	// pgx scans int4 as int32, the lake scan returns int64: both must compare
	res := KeyDiff([]any{int32(1), int32(2), int32(3)}, []any{int64(1), int64(3)})
	if res.Missing != 1 || res.OK {
		t.Fatalf("expected missing=1 not ok, got %+v", res)
	}
}

func TestKeyDiffStringKeys(t *testing.T) {
	res := KeyDiff([]any{"a", "b", "c"}, []any{"a", "c"})
	if res.Missing != 1 || res.OK {
		t.Fatalf("expected missing=1 not ok, got %+v", res)
	}
}

func TestNullRate(t *testing.T) {
	rows := []Row{{Null: false}, {Null: true}, {Null: false}, {Null: true}}
	rate := NullRate(rows)
	if math.Abs(rate-0.5) > 1e-9 {
		t.Fatalf("expected 0.5, got %v", rate)
	}
}

func TestNullRateMissingColumn(t *testing.T) {
	if got := NullRate([]Row{{Key: int64(1)}}); got != 0.0 {
		t.Fatalf("expected 0.0, got %v", got)
	}
}

func TestFreshnessSeconds(t *testing.T) {
	if got := FreshnessSeconds(1_000, 61_000); got != 60.0 {
		t.Fatalf("expected 60.0, got %v", got)
	}
}

func TestFreshnessClampsToZero(t *testing.T) {
	if got := FreshnessSeconds(61_000, 1_000); got != 0.0 {
		t.Fatalf("expected 0.0, got %v", got)
	}
}
