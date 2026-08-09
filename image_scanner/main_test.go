package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"scanner/internal/scanner"
)

func TestParseFlags(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tests := []struct {
		name      string
		args      []string
		wantDebug bool
		wantImage string
	}{
		{"defaults", []string{"scanner"}, false, "alpine:3.19"},
		{"image flag", []string{"scanner", "-image", "nginx:latest"}, false, "nginx:latest"},
		{"debug and image", []string{"scanner", "-debug", "-image", "ubuntu:22.04"}, true, "ubuntu:22.04"},
		{"image then debug", []string{"scanner", "-image", "ubuntu:22.04", "--debug"}, true, "ubuntu:22.04"},
		{"debug only", []string{"scanner", "-debug"}, true, "alpine:3.19"},
		{"positional ignored", []string{"scanner", "nginx:latest"}, false, "alpine:3.19"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			debug, image := parseFlags()
			if debug != tt.wantDebug {
				t.Errorf("debug = %v, want %v", debug, tt.wantDebug)
			}
			if image != tt.wantImage {
				t.Errorf("image = %q, want %q", image, tt.wantImage)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate([]string{"a", "b", "c"}, 2); len(got) != 3 || got[2] != "+1 more" {
		t.Fatalf("truncate over limit = %#v", got)
	}
	if got := truncate([]string{"a", "b"}, 5); len(got) != 2 {
		t.Fatalf("truncate under limit = %#v", got)
	}
	if got := truncate(nil, 3); len(got) != 0 {
		t.Fatalf("truncate nil = %#v", got)
	}
}

// TestRenderResults captures stdout and checks that the rendered table contains
// the expected rows and summary.
func TestRenderResults(t *testing.T) {
	findings := []scanner.Finding{
		{ID: "CVE-2022-1", Scanner: "grype", Severity: "critical", Package: "busybox", Version: "1.36.1", FixVersion: ""},
		{ID: "POL-1", Scanner: "policy", Severity: "medium", Package: "lodash", Version: "2.4.2", FixVersion: "4.17.21"},
	}

	out := captureStdout(t, func() { renderResults(findings) })

	for _, want := range []string{
		"Scan Results:",
		"SCANNER", "VULN ID", "SEVERITY", "PACKAGE", "VERSION", "FIX VERSIONS",
		"CVE-2022-1", "grype", "critical", "busybox", "1.36.1", "None",
		"POL-1", "policy", "medium", "lodash", "4.17.21",
		"Vulnerability Summary:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what
// was written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	data, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(data)
}
