package main

import (
	"testing"
)

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
