package scanner

import (
	"testing"

	"github.com/anchore/grype/grype/match"
	"github.com/anchore/grype/grype/vulnerability"
)

func TestSeverityOf(t *testing.T) {
	tests := []struct {
		name string
		meta *vulnerability.Metadata
		want string
	}{
		{"severity present", &vulnerability.Metadata{Severity: "High"}, "High"},
		{"empty severity string", &vulnerability.Metadata{Severity: ""}, "unknown"},
		{"nil metadata", nil, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := match.Match{Vulnerability: vulnerability.Vulnerability{Metadata: tt.meta}}
			if got := severityOf(m); got != tt.want {
				t.Fatalf("severityOf() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFixVersions(t *testing.T) {
	withFix := match.Match{Vulnerability: vulnerability.Vulnerability{
		Fix: vulnerability.Fix{Versions: []string{"1.2.3", "1.2.4"}},
	}}
	if got := fixVersions(withFix); got != "1.2.3, 1.2.4" {
		t.Fatalf("fixVersions() = %q, want %q", got, "1.2.3, 1.2.4")
	}

	noFix := match.Match{Vulnerability: vulnerability.Vulnerability{}}
	if got := fixVersions(noFix); got != "" {
		t.Fatalf("fixVersions() = %q, want empty", got)
	}
}
