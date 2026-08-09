package scanner

import (
	"context"
	"fmt"
	"strings"

	"github.com/anchore/clio"
	"github.com/anchore/grype/grype"
	"github.com/anchore/grype/grype/db/v6/distribution"
	"github.com/anchore/grype/grype/db/v6/installation"
	"github.com/anchore/grype/grype/match"
	"github.com/anchore/grype/grype/vulnerability"
	"github.com/charmbracelet/log"
)

// GrypeScanner finds known vulnerabilities by matching the image's packages
// against the Grype vulnerability database.
type GrypeScanner struct {
	provider vulnerability.Provider
}

// NewGrypeScanner loads (updating when needed) the Grype vulnerability database
// and returns a scanner backed by it.
func NewGrypeScanner() (*GrypeScanner, error) {
	log.Debug("Loading Grype vulnerability database")
	dbConfig := distribution.DefaultConfig()
	installConfig := installation.DefaultConfig(clio.Identification{
		Name:    "image-scanner",
		Version: "dev",
	})

	provider, _, err := grype.LoadVulnerabilityDB(dbConfig, installConfig, true)
	if err != nil {
		return nil, fmt.Errorf("load vulnerability database: %w", err)
	}
	log.Debug("Grype vulnerability database loaded")
	return &GrypeScanner{provider: provider}, nil
}

// Name implements Scanner.
func (g *GrypeScanner) Name() string { return "grype" }

// Scan implements Scanner: it runs Grype's matcher over the target's packages.
func (g *GrypeScanner) Scan(ctx context.Context, target *ScanTarget) ([]Finding, error) {
	log.Debug("Matching packages against Grype database", "packages", len(target.Packages))
	matcher := grype.VulnerabilityMatcher{
		VulnerabilityProvider: g.provider,
	}

	matches, _, err := matcher.FindMatchesContext(ctx, target.Packages, target.PkgContext)
	if err != nil {
		return nil, fmt.Errorf("match vulnerabilities: %w", err)
	}

	matchList := matches.Sorted()
	findings := make([]Finding, 0, len(matchList))
	for _, m := range matchList {
		findings = append(findings, Finding{
			ID:         m.Vulnerability.ID,
			Scanner:    g.Name(),
			Severity:   severityOf(m),
			Package:    m.Package.Name,
			Version:    m.Package.Version,
			FixVersion: fixVersions(m),
			Details:    m.Vulnerability.Namespace,
		})
	}
	log.Debug("Grype matching complete", "matches", len(findings))
	return findings, nil
}

// Close releases the underlying vulnerability database.
func (g *GrypeScanner) Close() error {
	if g.provider == nil {
		return nil
	}
	return g.provider.Close()
}

// severityOf returns the severity string for a match, defaulting to "unknown".
func severityOf(m match.Match) string {
	if m.Vulnerability.Metadata != nil && m.Vulnerability.Metadata.Severity != "" {
		return m.Vulnerability.Metadata.Severity
	}
	return "unknown"
}

// fixVersions returns the comma-joined list of fixed versions, or "" when unfixed.
func fixVersions(m match.Match) string {
	if len(m.Vulnerability.Fix.Versions) > 0 {
		return strings.Join(m.Vulnerability.Fix.Versions, ", ")
	}
	return ""
}
