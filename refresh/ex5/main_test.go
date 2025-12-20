package main

import "testing"

func TestSum(t *testing.T) {
	tests := map[string]struct {
		a        int
		b        int
		expected int
	}{
		"t1": {
			a:        1,
			b:        2,
			expected: 3,
		},
		"t2": {
			a:        2,
			b:        2,
			expected: 4,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := Sum(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("Error %d != %d", got, tt.expected)
			}
		})
	}
}
