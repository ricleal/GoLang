package scanner

import (
	"context"
	"testing"

	"github.com/anchore/syft/syft/linux"
	"github.com/anchore/syft/syft/pkg"
	"github.com/anchore/syft/syft/sbom"
	"github.com/anchore/syft/syft/source"
)

// TestCustomPipelineEndToEnd exercises the full custom-scan path without any
// network access: SBOM -> NewTarget -> Run(custom scanner) -> findings.
//
// NOTE: it asserts against the example policy data (EOL table + baseline map)
// in custom.go; adjust if those feeds change.
func TestCustomPipelineEndToEnd(t *testing.T) {
	collection := pkg.NewCollection(
		pkg.Package{Name: "musl", Version: "1.2.4", Type: pkg.ApkPkg, PURL: "pkg:apk/alpine/musl@1.2.4"},
		pkg.Package{Name: "lodash", Version: "2.4.2", Type: pkg.NpmPkg, PURL: "pkg:npm/lodash@2.4.2"},
	)
	s := &sbom.SBOM{
		Artifacts: sbom.Artifacts{
			Packages: collection,
			LinuxDistribution: &linux.Release{
				ID:        "alpine",
				VersionID: "3.16.9",
			},
		},
	}

	target := NewTarget(s, source.Description{})
	findings, err := Run(context.Background(), target, []Scanner{NewCustomScanner()})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Expect: EOL-DISTRO (alpine 3.16 past EOL) + OUTDATED-PACKAGE (lodash 2.4.2 < baseline).
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d: %#v", len(findings), findings)
	}

	byID := map[string]Finding{}
	for _, f := range findings {
		if f.Scanner != "policy" {
			t.Fatalf("expected scanner %q, got %q", "policy", f.Scanner)
		}
		byID[f.ID] = f
	}

	if _, ok := byID["EOL-DISTRO"]; !ok {
		t.Fatalf("missing EOL-DISTRO finding: %#v", findings)
	}
	if f, ok := byID["OUTDATED-PACKAGE"]; !ok {
		t.Fatalf("missing OUTDATED-PACKAGE finding: %#v", findings)
	} else if f.Package != "lodash" || f.Version != "2.4.2" {
		t.Fatalf("unexpected OUTDATED-PACKAGE finding: %#v", f)
	}
}
