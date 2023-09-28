package codility_test

import (
	"exp/codility"
	"testing"
)

func TestSolution3(t *testing.T) {
	// Table driven tests
	tests := []struct {
		name string
		N    int
		want int
	}{
		{
			name: "example 1",
			N:    24,
			want: 3,
		},
		{
			name: "example 2",
			N:    16,
			want: 4,
		},
		{
			name: "example 3",
			N:    1,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codility.Solution3(tt.N); got != tt.want {
				t.Errorf("Solution() = %v, want %v", got, tt.want)
			}
		})
	}
}
