package scanner

import (
	"github.com/anchore/grype/grype/distro"
	"github.com/anchore/grype/grype/pkg"
	"github.com/anchore/syft/syft/sbom"
	"github.com/anchore/syft/syft/source"
	"github.com/charmbracelet/log"
)

// NewTarget converts a Syft SBOM into the normalized ScanTarget that every
// scanner consumes. It is created once per image so scanners never re-parse
// the image layers.
func NewTarget(s *sbom.SBOM, srcDesc source.Description) *ScanTarget {
	// Convert the SBOM package collection into Grype's package representation.
	pkgPtrs := pkg.FromCollection(s.Artifacts.Packages, s.Relationships, pkg.SynthesisConfig{})
	packages := make([]pkg.Package, 0, len(pkgPtrs))
	for _, p := range pkgPtrs {
		packages = append(packages, *p)
	}

	// Detect the OS distro from the SBOM to build the package context.
	d := distro.FromRelease(s.Artifacts.LinuxDistribution, nil)
	pkgCtx := pkg.Context{
		Source:                &srcDesc,
		Distro:                d,
		DistroDetectionFailed: s.Artifacts.LinuxDistribution != nil && d == nil,
	}

	// Mirror grype's pkg.Provide behavior: attach the detected distro to each package.
	if d != nil {
		for i := range packages {
			if packages[i].Distro == nil {
				packages[i].Distro = d
			}
		}
	}

	distroStr := ""
	if d != nil {
		distroStr = d.String()
	}

	log.Debug("Built scan target", "packages", len(packages), "distro", distroStr)

	return &ScanTarget{
		SBOM:       s,
		Packages:   packages,
		PkgContext: pkgCtx,
		Distro:     distroStr,
	}
}
