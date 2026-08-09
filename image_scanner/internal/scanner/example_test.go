package scanner

import (
	"context"
	"fmt"

	grypePkg "github.com/anchore/grype/grype/pkg"
	"github.com/anchore/syft/syft/pkg"
	"github.com/anchore/syft/syft/sbom"
	"github.com/anchore/syft/syft/source"
)

// ExampleRun demonstrates the full scan pipeline with deterministic, verified
// output: an SBOM is built in memory, normalized into a ScanTarget, and scanned
// by the policy scanner.
//
// The output is deterministic because no Linux distribution is present (so the
// time-dependent EOL check is skipped) and the only finding comes from the
// package baseline in custom.go. `go test` verifies the printed output against
// the `// Output:` comment below.
func ExampleRun() {
	collection := pkg.NewCollection(
		pkg.Package{Name: "lodash", Version: "2.4.2", Type: pkg.NpmPkg, PURL: "pkg:npm/lodash@2.4.2"},
		pkg.Package{Name: "lodash", Version: "4.17.21", Type: pkg.NpmPkg, PURL: "pkg:npm/lodash@4.17.21"},
		pkg.Package{Name: "express", Version: "4.18.2", Type: pkg.NpmPkg, PURL: "pkg:npm/express@4.18.2"},
	)
	s := &sbom.SBOM{
		Artifacts: sbom.Artifacts{Packages: collection},
	}

	target := NewTarget(s, source.Description{})
	findings, err := Run(context.Background(), target, []Scanner{NewCustomScanner()})
	if err != nil {
		panic(err)
	}

	for _, f := range findings {
		fmt.Printf("%s: %s %s@%s\n", f.ID, f.Severity, f.Package, f.Version)
	}

	// Output:
	// OUTDATED-PACKAGE: medium lodash@2.4.2
}

// nameBlockScanner is a minimal Scanner used by ExampleScanner. It flags any
// package whose name appears on a small blocklist.
type nameBlockScanner struct{}

// Name implements Scanner.
func (nameBlockScanner) Name() string { return "name-blocklist" }

// Scan implements Scanner.
func (nameBlockScanner) Scan(_ context.Context, target *ScanTarget) ([]Finding, error) {
	var out []Finding
	for _, p := range target.Packages {
		if p.Name == "debug-tool" {
			out = append(out, Finding{
				ID:       "BLOCKED-PACKAGE",
				Scanner:  "name-blocklist",
				Severity: "high",
				Package:  p.Name,
				Version:  p.Version,
				Details:  "package is blocked by organization policy",
			})
		}
	}
	return out, nil
}

// ExampleScanner shows how to implement the Scanner interface to add your own
// scanner to the pipeline. Its output is fully deterministic.
func ExampleScanner() {
	target := &ScanTarget{Packages: []grypePkg.Package{
		{Name: "debug-tool", Version: "1.0.0"},
		{Name: "app", Version: "2.0.0"},
	}}

	findings, err := Run(context.Background(), target, []Scanner{nameBlockScanner{}})
	if err != nil {
		panic(err)
	}

	for _, f := range findings {
		fmt.Printf("%s: %s %s@%s\n", f.ID, f.Severity, f.Package, f.Version)
	}

	// Output:
	// BLOCKED-PACKAGE: high debug-tool@1.0.0
}
