package main

import (
	"fmt"
	"math/rand"
	"testing"
)

// TypePackage represents the type of package.
type TypePackage int

func (t TypePackage) String() string {
	return [...]string{"STANDARD", "SPECIAL", "REJECTED"}[t]
}

const (
	// standard packages (those that are not bulky or heavy) can be handled normally.
	Standard TypePackage = iota
	// packages that are either heavy or bulky can't be handled automatically.
	Special
	// packages that are both heavy and bulky are rejected.
	Rejected
)

const (
	// 1_000_000 cm³.
	MaxVolume = 1_000_000
	// 150 cm.
	MaxDimension = 150
	// 20 kg.
	MaxMass = 20
)

// isBulky returns true if the package has a volume (Width x Height x Length) greater than
// or equal to 1,000,000 cm³ or when one of its dimensions is greater or equal to 150 cm.
// units are centimeters.
func isBulky(width, height, length int) bool {
	return width >= MaxDimension ||
		height >= MaxDimension ||
		length >= MaxDimension ||
		width*height*length >= MaxVolume
}

// isHeavy returns true if the package has a mass greater or equal to 20 kg.
// units are kilograms.
func isHeavy(mass int) bool {
	return mass >= MaxMass
}

// Sort returns the name of the stack where the package should go.
// The package should go to the STANDARD stack if it is not bulky or heavy.
// The package should go to the SPECIAL stack if it is either heavy or bulky.
// The package should go to the REJECTED stack if it is both heavy and bulky.
// units are centimeters for the dimensions and kilograms for the mass.
func Sort(width, height, length, mass int) string {
	bulky := isBulky(width, height, length)
	heavy := isHeavy(mass)

	if heavy && bulky {
		return Rejected.String()
	}

	if heavy || bulky {
		return Special.String()
	}

	return Standard.String()
}

// Replit runs the tests in a directory without a main package. Unfortunely this needs to be here...
func TestSort(t *testing.T) {
	tests := []struct {
		width, height, length, mass int
		expected                    TypePackage
	}{
		{99, 100, 100, 10, Standard},  // total volume = 990_000 cm³.
		{100, 100, 100, 10, Special},  // total volume = 1_000_000 cm³.
		{10, 10, 10, 100, Special},    // mass > 20 kg.
		{150, 100, 100, 10, Special},  // width >= 150 cm.
		{100, 150, 100, 30, Rejected}, // height >= 150 cm and mass > 20 kg.
	}

	for _, tt := range tests {
		result := Sort(tt.width, tt.height, tt.length, tt.mass)
		if result != tt.expected.String() {
			t.Errorf("Sort(%d, %d, %d, %d) = %s; want %s", tt.width, tt.height, tt.length, tt.mass, result, tt.expected)
		}
	}
}

func main() {
	// random values for the dimensions and mass.
	width, height, length, mass := rand.Intn(200), rand.Intn(200), rand.Intn(200), rand.Intn(40)
	res := Sort(width, height, length, mass)
	fmt.Printf("Sort(%d, %d, %d, %d) = %s\n", width, height, length, mass, res)

	// Run the tests programatically.
	fmt.Println("Running tests...")
	testing.Main(
		nil,
		[]testing.InternalTest{
			{"TestSort", TestSort},
		},
		nil, nil,
	)
}
