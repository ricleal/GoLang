package codility_test

import (
	"exp/codility"
	"testing"
)

func TestSolution(t *testing.T) {
	// table driven tests
	tests := []struct {
		name string
		A    []int
		want int
	}{
		{
			name: "example 1",
			A:    []int{1, 3, 6, 4, 1, 2},
			want: 5,
		},
		{
			name: "example 2",
			A:    []int{1, 2, 3},
			want: 4,
		},
		{
			name: "example 3",
			A:    []int{-1, -3},
			want: 1,
		},
		{
			name: "example 4",
			A:    []int{1, 2, 3, 4, 5, 6},
			want: 7,
		},
		{
			name: "example 5",
			A:    []int{},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codility.Solution(tt.A); got != tt.want {
				t.Errorf("Solution() = %v, want %v", got, tt.want)
			}
		})
	}
}
