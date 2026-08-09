package scanner

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/anchore/syft/syft/linux"
	"github.com/anchore/syft/syft/pkg"
	"github.com/anchore/syft/syft/sbom"
	"github.com/anchore/syft/syft/source"
)

// fakeScanner is a Scanner implementation used in tests. It mimics real
// scanners by stamping its name onto every finding it returns.
type fakeScanner struct {
	name     string
	findings []Finding
	err      error
}

func (f *fakeScanner) Name() string { return f.name }
func (f *fakeScanner) Scan(context.Context, *ScanTarget) ([]Finding, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]Finding, len(f.findings))
	for i, fd := range f.findings {
		fd.Scanner = f.name
		out[i] = fd
	}
	return out, nil
}

func TestRunMergesNormalizesAndSortsFindings(t *testing.T) {
	scanners := []Scanner{
		&fakeScanner{name: "a", findings: []Finding{
			{ID: "Z", Severity: "HIGH", Package: "p", Version: "1"},
			{ID: "Z", Severity: "High", Package: "p", Version: "1"}, // exact dup of the above (same scanner)
			{ID: "A", Severity: "low", Package: "p", Version: "1"},
		}},
		&fakeScanner{name: "b", findings: []Finding{
			{ID: "Z", Severity: "high", Package: "p", Version: "1"}, // same vuln from another scanner: kept
			{ID: "M", Severity: "critical", Package: "p", Version: "1"},
		}},
	}

	got, err := Run(context.Background(), &ScanTarget{}, scanners)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(got) != 4 {
		t.Fatalf("expected 4 unique findings, got %d: %#v", len(got), got)
	}

	// Severities must be canonicalized to lowercase.
	for _, f := range got {
		if f.Severity != strings.ToLower(f.Severity) {
			t.Fatalf("severity not normalized: %q", f.Severity)
		}
	}

	// Sorted by severity (desc) then ID.
	want := []Finding{
		{ID: "M", Scanner: "b", Severity: "critical"},
		{ID: "Z", Scanner: "a", Severity: "high"},
		{ID: "Z", Scanner: "b", Severity: "high"},
		{ID: "A", Scanner: "a", Severity: "low"},
	}
	for i, w := range want {
		if got[i].ID != w.ID || got[i].Scanner != w.Scanner || got[i].Severity != w.Severity {
			t.Fatalf("item %d = %#v, want %#v", i, got[i], w)
		}
	}
}

func TestRunReturnsScannerError(t *testing.T) {
	boom := errors.New("boom")
	scanners := []Scanner{
		&fakeScanner{name: "ok", findings: []Finding{{ID: "A"}}},
		&fakeScanner{name: "bad", err: boom},
	}

	_, err := Run(context.Background(), &ScanTarget{}, scanners)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestSeverityRank(t *testing.T) {
	tests := []struct {
		sev  string
		want int
	}{
		{"critical", 5},
		{"high", 4},
		{"medium", 3},
		{"low", 2},
		{"negligible", 1},
		{"unknown", 0},
		{"", 0},
	}
	for _, tt := range tests {
		if got := severityRank(tt.sev); got != tt.want {
			t.Errorf("severityRank(%q) = %d, want %d", tt.sev, got, tt.want)
		}
	}
}

func TestNewTarget(t *testing.T) {
	collection := pkg.NewCollection(
		pkg.Package{Name: "musl", Version: "1.2.4", Type: pkg.ApkPkg, PURL: "pkg:apk/alpine/musl@1.2.4"},
		pkg.Package{Name: "lodash", Version: "4.17.21", Type: pkg.NpmPkg, PURL: "pkg:npm/lodash@4.17.21"},
	)
	s := &sbom.SBOM{
		Artifacts: sbom.Artifacts{
			Packages: collection,
			LinuxDistribution: &linux.Release{
				PrettyName: "Alpine Linux v3.19",
				ID:         "alpine",
				VersionID:  "3.19.9",
			},
		},
	}

	target := NewTarget(s, source.Description{})
	if target == nil {
		t.Fatal("NewTarget returned nil")
	}
	if target.SBOM != s {
		t.Fatal("target should reference the source SBOM")
	}
	if len(target.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(target.Packages))
	}
	if target.Distro != "alpine 3.19.9" {
		t.Fatalf("expected distro %q, got %q", "alpine 3.19.9", target.Distro)
	}
	if target.PkgContext.Distro == nil {
		t.Fatal("expected package context distro to be set")
	}
	for _, p := range target.Packages {
		if p.Distro == nil {
			t.Fatalf("expected package %s to have its distro attached", p.Name)
		}
	}
}
