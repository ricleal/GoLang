package codility_test

import (
	"exp/codility"
	"testing"
)

func TestSolution4(t *testing.T) {
	// Table driven tests
	tests := []struct {
		name string
		A    []int
		want int
	}{
		{
			name: "example 1",
			A:    []int{-1, 6, 3, 4, 7, 4},
			want: 4,
		},
		{
			name: "example 2",
			A:    []int{1, 2, 3},
			want: 0,
		},
		{
			name: "example 3",
			A:    []int{-1, -3},
			want: 1,
		},
		{
			name: "example 4",
			A:    []int{1, 2, 3, 4, 5, 6},
			want: 0,
		},
		{
			name: "example 5",
			A:    []int{6, 5, 4, 3, 2, 1},
			want: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codility.Solution4(tt.A); got != tt.want {
				t.Errorf("Solution() = %v, want %v", got, tt.want)
			}
		})
	}
}
