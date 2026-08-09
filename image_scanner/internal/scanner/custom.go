package scanner

import (
	"context"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/charmbracelet/log"
)

// CustomScanner applies organization-specific policy checks to the scan target.
// It is the template for adding your own scanners: implement Scan, then register
// it in main.go alongside the Grype scanner.
//
// It currently performs three checks (all backed by EXAMPLE data — replace with
// your own feeds):
//  1. EOL distro detection: flags images based on an OS release that is past
//     its end-of-life date.
//  2. Deny-list check: flags packages whose version matches a denied prefix.
//  3. Outdated-package check: flags packages below a required minimum version.
type CustomScanner struct{}

// NewCustomScanner returns a ready-to-use CustomScanner.
func NewCustomScanner() *CustomScanner { return &CustomScanner{} }

// Name implements Scanner.
func (s *CustomScanner) Name() string { return "policy" }

// eolDistros maps distro ID -> release -> end-of-life date (YYYY-MM-DD).
// This is illustrative data; wire it to a real EOL feed (e.g. endoflife.date)
// for production use.
var eolDistros = map[string]map[string]string{
	"alpine": {
		"3.16": "2024-05-23",
		"3.17": "2024-11-22",
		"3.18": "2025-05-09",
		"3.19": "2025-11-01",
		"3.20": "2026-04-01",
	},
	"ubuntu": {
		"18.04": "2028-05-31", // extended security maintenance
	},
	"debian": {
		"11": "2026-08-14", // LTS
	},
}

// deniedPackages lists package@version-prefix combinations that are not allowed
// in production images. Example entry only — extend with your own policy.
var deniedPackages = []struct {
	name          string
	versionPrefix string
}{
	// {name: "openssl", versionPrefix: "1.0."},
}

// requiredMinVersions defines the minimum acceptable version for security-
// sensitive packages (a "security baseline").
//
// EXAMPLE data covering the npm packages of a typical Node app — replace with
// your organization's enforced baseline (or a feed such as NVD/OSV).
var requiredMinVersions = map[string]string{
	"lodash":          "4.17.21",
	"jsonwebtoken":    "9.0.0",
	"tar":             "7.5.19",
	"ws":              "8.21.0",
	"sanitize-html":   "2.17.5",
	"socket.io":       "4.6.2",
	"moment":          "2.30.1",
	"js-yaml":         "4.1.0",
	"crypto-js":       "4.2.0",
	"uuid":            "11.1.1",
	"express-jwt":     "6.0.0",
	"multer":          "2.0.2",
	"ip-address":      "10.3.1",
	"minimatch":       "10.2.3",
	"brace-expansion": "5.0.9",
	"node":            "24.18.1",
}

// Scan implements Scanner.
func (s *CustomScanner) Scan(_ context.Context, target *ScanTarget) ([]Finding, error) {
	log.Debug("Running policy checks", "packages", len(target.Packages), "distro", target.Distro)

	var findings []Finding

	findings = append(findings, s.checkEOLDistro(target)...)
	findings = append(findings, s.checkDenyList(target)...)
	findings = append(findings, s.checkOutdatedPackages(target)...)

	log.Debug("Policy checks complete", "findings", len(findings))
	return findings, nil
}

// checkEOLDistro flags the image if its OS release is past end-of-life.
func (s *CustomScanner) checkEOLDistro(target *ScanTarget) []Finding {
	id, version := parseDistro(target.Distro)
	if id == "" {
		return nil
	}

	for release, eol := range eolDistros[id] {
		if !versionMatches(version, release) {
			continue
		}
		eolDate, err := time.Parse("2006-01-02", eol)
		if err != nil {
			continue
		}
		if time.Now().After(eolDate) {
			return []Finding{{
				ID:       "EOL-DISTRO",
				Scanner:  s.Name(),
				Severity: "high",
				Package:  id,
				Version:  version,
				Details:  "operating system release reached end-of-life on " + eol,
			}}
		}
	}
	return nil
}

// checkDenyList flags packages whose version matches a denied prefix.
func (s *CustomScanner) checkDenyList(target *ScanTarget) []Finding {
	var findings []Finding
	for _, p := range target.Packages {
		for _, denied := range deniedPackages {
			if p.Name == denied.name && strings.HasPrefix(p.Version, denied.versionPrefix) {
				findings = append(findings, Finding{
					ID:       "DENIED-PACKAGE",
					Scanner:  s.Name(),
					Severity: "high",
					Package:  p.Name,
					Version:  p.Version,
					Details:  "package version is on the organization deny-list",
				})
			}
		}
	}
	return findings
}

// checkOutdatedPackages flags packages whose installed version is below the
// required security baseline.
func (s *CustomScanner) checkOutdatedPackages(target *ScanTarget) []Finding {
	var findings []Finding
	for _, p := range target.Packages {
		required, ok := requiredMinVersions[p.Name]
		if !ok {
			continue
		}

		installed, err := semver.NewVersion(p.Version)
		if err != nil {
			continue // ignore versions we cannot parse
		}
		min, err := semver.NewVersion(required)
		if err != nil {
			continue
		}
		if installed.LessThan(min) {
			findings = append(findings, Finding{
				ID:         "OUTDATED-PACKAGE",
				Scanner:    s.Name(),
				Severity:   "medium",
				Package:    p.Name,
				Version:    p.Version,
				FixVersion: required,
				Details:    "package version is below the required security baseline",
			})
		}
	}
	return findings
}

// parseDistro splits a distro string like "alpine 3.19.1" into its ID and version.
func parseDistro(distro string) (id, version string) {
	parts := strings.Fields(distro)
	if len(parts) == 0 {
		return "", ""
	}
	id = parts[0]
	if len(parts) > 1 {
		version = parts[1]
	}
	return id, version
}

// versionMatches reports whether a version (e.g. "3.19.1") matches a release
// key (e.g. "3.19"), tolerating patch-level differences.
func versionMatches(version, release string) bool {
	return version == release || strings.HasPrefix(version, release+".")
}
