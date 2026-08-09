// Package scanner defines a pluggable scanning framework for container images.
//
// Every scanner (the built-in Grype CVE matcher, or your own custom scanners)
// implements the Scanner interface and returns a uniform set of Finding values.
// This decouples scanners from the Syft/Grype internals and lets you mix and
// match vulnerability databases, policy checks, secret scans, etc.
package scanner

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/anchore/grype/grype/pkg"
	"github.com/anchore/syft/syft/sbom"
	"github.com/charmbracelet/log"
)

// Finding is a single vulnerability or policy finding reported by any scanner.
type Finding struct {
	ID         string // e.g. CVE-2024-1234, or a custom rule ID like "EOL-DISTRO"
	Scanner    string // name of the scanner that produced this finding
	Severity   string // critical | high | medium | low | unknown
	Package    string // affected package name ("" for image-level findings)
	Version    string // affected package version
	FixVersion string // version that fixes the issue, or "" when unfixed
	Details    string // human-readable explanation
}

// ScanTarget is the normalized model every scanner consumes. It is built once
// from the SBOM so scanners never need to re-parse the image.
type ScanTarget struct {
	SBOM       *sbom.SBOM        // the raw SBOM for scanners that need it
	Packages   []pkg.Package     // packages in Grype's representation
	PkgContext pkg.Context       // distro + source context used by the matcher
	Distro     string            // human-friendly distro, e.g. "alpine 3.19.1"
	Files      map[string]string // path -> contents (populated by scanners that need file access)
}

// Scanner is implemented by any vulnerability or policy scanner.
type Scanner interface {
	// Name returns a short identifier for the scanner (used in Finding.Scanner).
	Name() string
	// Scan inspects the target and returns zero or more findings.
	Scan(ctx context.Context, target *ScanTarget) ([]Finding, error)
}

// Run executes every scanner in order against the target and merges their
// findings, removing exact duplicates and normalizing severities. Scanners run
// sequentially so they can safely share state (e.g. the Grype DB); parallelize
// later if needed.
func Run(ctx context.Context, target *ScanTarget, scanners []Scanner) ([]Finding, error) {
	var all []Finding
	for _, sc := range scanners {
		log.Debug("Running scanner", "name", sc.Name(), "packages", len(target.Packages))
		findings, err := sc.Scan(ctx, target)
		if err != nil {
			return nil, fmt.Errorf("scanner %q failed: %w", sc.Name(), err)
		}
		// Canonicalize severity (e.g. "High" -> "high") so findings from
		// different scanners aggregate consistently in reports.
		for i := range findings {
			findings[i].Severity = strings.ToLower(findings[i].Severity)
		}
		log.Debug("Scanner finished", "name", sc.Name(), "findings", len(findings))
		all = append(all, findings...)
	}
	return dedupe(all), nil
}

// dedupe removes exact duplicate findings across scanners.
func dedupe(findings []Finding) []Finding {
	seen := make(map[string]bool, len(findings))
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		key := f.Scanner + "|" + f.ID + "|" + f.Package + "|" + f.Version
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, f)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return severityRank(out[i].Severity) > severityRank(out[j].Severity)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// severityRank orders severities so the most severe findings sort first.
func severityRank(s string) int {
	switch s {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "negligible":
		return 1
	default:
		return 0
	}
}
