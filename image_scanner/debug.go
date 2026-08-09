package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/anchore/syft/syft/pkg"
	"github.com/anchore/syft/syft/sbom"
)

// printSBOMStructure prints a readable overview of the SBOM's structure so the
// contents of an image can be inspected. It is triggered by the -debug flag.
func printSBOMStructure(s *sbom.SBOM) {
	fmt.Println("\n===== SBOM structure (debug) =====")

	// Descriptor
	fmt.Printf("descriptor : name=%q version=%q\n", s.Descriptor.Name, s.Descriptor.Version)

	// Source
	src := s.Source
	fmt.Printf("source     : id=%q name=%q version=%q supplier=%q metadata=%T\n",
		src.ID, src.Name, src.Version, src.Supplier, src.Metadata)

	// OS / distro
	if ld := s.Artifacts.LinuxDistribution; ld != nil {
		fmt.Printf("distro     : %s %s (id=%q idLike=%v)\n", ld.PrettyName, ld.VersionID, ld.ID, ld.IDLike)
	} else {
		fmt.Println("distro     : none detected")
	}

	// Packages
	pkgs := s.Artifacts.Packages.Sorted()
	fmt.Printf("packages   : %d total\n", len(pkgs))
	fmt.Printf("  by type    : %s\n", groupCounts(pkgs, func(p pkg.Package) string { return string(p.Type) }))
	fmt.Printf("  by language: %s\n", groupCounts(pkgs, func(p pkg.Package) string { return string(p.Language) }))

	// Relationships (dependency edges)
	fmt.Printf("relations  : %d total\n", len(s.Relationships))
	if len(s.Relationships) > 0 {
		counts := map[string]int{}
		for _, r := range s.Relationships {
			counts[string(r.Type)]++
		}
		fmt.Printf("  by type    : %s\n", renderCounts(counts))
	}

	// Files
	fmt.Printf("files      : metadata=%d digests=%d contents=%d licenses=%d executables=%d unknowns=%d\n",
		len(s.Artifacts.FileMetadata), len(s.Artifacts.FileDigests), len(s.Artifacts.FileContents),
		len(s.Artifacts.FileLicenses), len(s.Artifacts.Executables), len(s.Artifacts.Unknowns))

	// Sample packages
	const sample = 5
	fmt.Printf("\nsample packages (first %d):\n", min(sample, len(pkgs)))
	for i, p := range pkgs {
		if i >= sample {
			break
		}
		printPackage(p)
	}

	fmt.Println("===== end SBOM structure =====")
}

// printPackage prints the key attributes of a single package.
func printPackage(p pkg.Package) {
	licenses := make([]string, 0, len(p.Licenses.ToSlice()))
	for _, l := range p.Licenses.ToSlice() {
		if l.Value != "" {
			licenses = append(licenses, l.Value)
		}
	}

	locations := make([]string, 0, len(p.Locations.ToSlice()))
	for _, l := range p.Locations.ToSlice() {
		locations = append(locations, l.RealPath)
	}

	fmt.Printf("  %s@%s\n", p.Name, p.Version)
	fmt.Printf("    type=%s language=%s foundBy=%q\n", p.Type, p.Language, p.FoundBy)
	fmt.Printf("    purl=%s\n", p.PURL)
	fmt.Printf("    cpes=%d licenses=%v\n", len(p.CPEs), licenses)
	fmt.Printf("    metadata=%T locations=%v\n", p.Metadata, truncate(locations, 3))
}

// groupCounts returns a "count by key" summary for a list of packages,
// e.g. "npm=120 deb=14".
func groupCounts(pkgs []pkg.Package, key func(pkg.Package) string) string {
	counts := map[string]int{}
	for _, p := range pkgs {
		k := key(p)
		if k == "" {
			k = "unknown"
		}
		counts[k]++
	}
	return renderCounts(counts)
}

// renderCounts renders a map of counts sorted by key, e.g. "deb=14 npm=120".
func renderCounts(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%s=%d", k, counts[k])
	}
	return sb.String()
}

// truncate limits a slice to at most n items, noting how many were omitted.
func truncate(items []string, n int) []string {
	if len(items) <= n {
		return items
	}
	out := append([]string{}, items[:n]...)
	return append(out, fmt.Sprintf("+%d more", len(items)-n))
}
