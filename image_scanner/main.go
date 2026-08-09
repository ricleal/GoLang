package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/anchore/syft/syft"
	"github.com/anchore/syft/syft/sbom"
	"github.com/anchore/syft/syft/source"
	"github.com/charmbracelet/log"

	"scanner/internal/scanner"
)

func main() {
	debug, imageRef := parseFlags()
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	ctx := context.Background()

	log.Debug("Starting image scanner", "image", imageRef, "debug", debug)
	log.Info("[1/3] Building SBOM", "image", imageRef)
	s, srcDesc, err := buildSBOM(ctx, imageRef)
	if err != nil {
		exitWithError("Failed to build SBOM", err)
	}
	log.Info("Discovered software packages", "count", len(s.Artifacts.Packages.Sorted()))

	// -debug: dump the SBOM structure so the image contents can be inspected.
	if debug {
		log.Debug("Dumping SBOM structure")
		printSBOMStructure(s)
	}

	// Normalize the SBOM once into a target that every scanner consumes.
	target := scanner.NewTarget(s, srcDesc)
	log.Debug("Normalized SBOM into scan target", "packages", len(target.Packages), "distro", target.Distro)

	// Register the scanners to run: the built-in Grype matcher plus your custom one.
	grypeScanner, err := scanner.NewGrypeScanner()
	if err != nil {
		exitWithError("Failed to load vulnerability database", err)
	}
	defer grypeScanner.Close()

	scanners := []scanner.Scanner{
		grypeScanner,
		scanner.NewCustomScanner(),
	}

	log.Info("[2/3] Running vulnerability scanners", "count", len(scanners))
	findings, err := scanner.Run(ctx, target, scanners)
	if err != nil {
		exitWithError("Failed to run scanners", err)
	}
	log.Debug("Scanners finished", "findings", len(findings))

	log.Info("[3/3] Rendering results", "findings", len(findings))
	renderResults(findings)
}

// parseFlags parses the command line with the standard flag package:
//   - -debug: enable debug-level logging and print the SBOM structure
//   - -image: image reference to scan (defaults to alpine:3.19)
//
// -h prints usage for free.
func parseFlags() (debug bool, imageRef string) {
	fs := flag.NewFlagSet("image-scanner", flag.ExitOnError)
	fs.BoolVar(&debug, "debug", false, "enable debug-level logging and print the SBOM structure")
	fs.StringVar(&imageRef, "image", "alpine:3.19", "image reference to scan")
	_ = fs.Parse(os.Args[1:])
	return debug, imageRef
}

// buildSBOM resolves an image reference and catalogs its packages into an SBOM.
// It also returns the source description captured before the source is closed.
func buildSBOM(ctx context.Context, imageRef string) (*sbom.SBOM, source.Description, error) {
	log.Debug("Resolving image source", "image", imageRef)
	src, err := syft.GetSource(ctx, imageRef, nil)
	if err != nil {
		return nil, source.Description{}, fmt.Errorf("get image source: %w", err)
	}
	defer src.Close()

	log.Debug("Cataloging packages from image source", "image", imageRef)
	s, err := syft.CreateSBOM(ctx, src, nil)
	if err != nil {
		return nil, source.Description{}, fmt.Errorf("create SBOM: %w", err)
	}

	log.Debug("SBOM built", "packages", len(s.Artifacts.Packages.Sorted()))
	return s, src.Describe(), nil
}

// renderResults prints the merged findings from all scanners as a table
// followed by a severity summary.
func renderResults(findings []scanner.Finding) {
	if len(findings) == 0 {
		fmt.Println("\nNo vulnerabilities or policy findings detected!")
		return
	}

	fmt.Println("\nScan Results:")

	// tabwriter aligns columns to their content, so long package names,
	// versions, and fix lists stay readable instead of overflowing.
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 1, ' ', 0)
	fmt.Fprintln(w, "SCANNER\t|\tVULN ID\t|\tSEVERITY\t|\tPACKAGE\t|\tVERSION\t|\tFIX VERSIONS")

	severityCounts := make(map[string]int)
	for _, f := range findings {
		severityCounts[f.Severity]++

		fix := f.FixVersion
		if fix == "" {
			fix = "None"
		}
		fmt.Fprintf(w, "%s\t|\t%s\t|\t%s\t|\t%s\t|\t%s\t|\t%s\n",
			f.Scanner, f.ID, f.Severity, f.Package, f.Version, fix)
	}
	_ = w.Flush()
	log.Debug("Rendered findings", "by_severity", severityCounts)

	fmt.Println("\nVulnerability Summary:")
	for sev, count := range severityCounts {
		fmt.Printf("  %-10s: %d\n", sev, count)
	}
}

// exitWithError logs a message and terminates the process with a non-zero exit code.
func exitWithError(prefix string, err error) {
	log.Error(prefix, "err", err)
	os.Exit(1)
}
