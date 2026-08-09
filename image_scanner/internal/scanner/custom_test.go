package scanner

import (
	"testing"

	"github.com/anchore/grype/grype/pkg"
)

func TestCheckEOLDistro(t *testing.T) {
	s := NewCustomScanner()
	tests := []struct {
		name   string
		distro string
		want   int
	}{
		{"EOL alpine 3.16 (past end-of-life)", "alpine 3.16.9", 1},
		{"EOL alpine 3.19 (past end-of-life)", "alpine 3.19.9", 1},
		{"release not in EOL table", "alpine 3.21.0", 0},
		{"distro not in EOL table", "ubuntu 22.04", 0},
		{"no distro detected", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := s.checkEOLDistro(&ScanTarget{Distro: tt.distro})
			if len(findings) != tt.want {
				t.Fatalf("expected %d findings, got %d: %#v", tt.want, len(findings), findings)
			}
		})
	}
}

func TestCheckDenyList(t *testing.T) {
	// The deny list is empty by default; seed it for the test and restore after.
	original := deniedPackages
	defer func() { deniedPackages = original }()
	deniedPackages = []struct {
		name          string
		versionPrefix string
	}{
		{name: "openssl", versionPrefix: "1.0."},
	}

	s := NewCustomScanner()
	target := &ScanTarget{Packages: []pkg.Package{
		{Name: "openssl", Version: "1.0.2"},
		{Name: "openssl", Version: "3.0.13"},
		{Name: "busybox", Version: "1.36.1"},
	}}

	findings := s.checkDenyList(target)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %#v", len(findings), findings)
	}
	if findings[0].ID != "DENIED-PACKAGE" || findings[0].Package != "openssl" || findings[0].Version != "1.0.2" {
		t.Fatalf("unexpected finding: %#v", findings[0])
	}
}

func TestCheckOutdatedPackages(t *testing.T) {
	// NOTE: this asserts against the example data in requiredMinVersions
	// (lodash baseline = 4.17.21). Adjust if the baseline feed changes.
	s := NewCustomScanner()
	target := &ScanTarget{Packages: []pkg.Package{
		{Name: "lodash", Version: "2.4.2"},   // below baseline -> finding
		{Name: "lodash", Version: "4.17.21"}, // at baseline -> no finding
		{Name: "lodash", Version: "4.20.0"},  // above baseline -> no finding
		{Name: "not-baselined", Version: "1.0"},
	}}

	findings := s.checkOutdatedPackages(target)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %#v", len(findings), findings)
	}
	if findings[0].ID != "OUTDATED-PACKAGE" || findings[0].Package != "lodash" || findings[0].Version != "2.4.2" {
		t.Fatalf("unexpected finding: %#v", findings[0])
	}
}

func TestParseDistro(t *testing.T) {
	tests := []struct {
		in, wantID, wantVer string
	}{
		{"alpine 3.19.9", "alpine", "3.19.9"},
		{"debian 12 (bookworm)", "debian", "12"},
		{"alpine", "alpine", ""},
		{"", "", ""},
	}
	for _, tt := range tests {
		id, ver := parseDistro(tt.in)
		if id != tt.wantID || ver != tt.wantVer {
			t.Errorf("parseDistro(%q) = (%q, %q), want (%q, %q)", tt.in, id, ver, tt.wantID, tt.wantVer)
		}
	}
}

func TestVersionMatches(t *testing.T) {
	tests := []struct {
		version, release string
		want             bool
	}{
		{"3.19.9", "3.19", true},
		{"3.19", "3.19", true},
		{"3.20.1", "3.19", false},
		{"3.1.9", "3.19", false},
		{"", "3.19", false},
	}
	for _, tt := range tests {
		if got := versionMatches(tt.version, tt.release); got != tt.want {
			t.Errorf("versionMatches(%q, %q) = %v, want %v", tt.version, tt.release, got, tt.want)
		}
	}
}
