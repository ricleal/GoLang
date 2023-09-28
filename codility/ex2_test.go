package codility_test

import (
	"exp/codility"
	"testing"
)

func TestSolution2(t *testing.T) {
	// Table driven tests
	tests := []struct {
		name string
		A    []int
		want int
	}{
		// {
		// 	name: "example 1",
		// 	A:    []int{3, 2, 6, -1, 4, 5, -1, 2},
		// 	want: 17,
		// },
		{
			name: "example 2",
			A:    []int{3, 2, 6, -1, 4, 5, -1, 2},
			want: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codility.Solution2(tt.A); got != tt.want {
				t.Errorf("Solution() = %v, want %v", got, tt.want)
			}
		})
	}
}
