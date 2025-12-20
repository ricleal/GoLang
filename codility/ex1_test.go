package codility_test

import (
	"testing"

	"exp/codility"
)

func TestSolution(t *testing.T) {
	// table driven tests
	tests := map[string]struct {
		A    []int
		want int
	}{
		"example 1": {
			A:    []int{1, 3, 6, 4, 1, 2},
			want: 5,
		},
		"example 2": {
			A:    []int{1, 2, 3},
			want: 4,
		},
		"example 3": {
			A:    []int{-1, -3},
			want: 1,
		},
		"example 4": {
			A:    []int{1, 2, 3, 4, 5, 6},
			want: 7,
		},
		"example 5": {
			A:    []int{},
			want: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := codility.Solution(tt.A); got != tt.want {
				t.Errorf("Solution() = %v, want %v", got, tt.want)
			}
		})
	}
}
