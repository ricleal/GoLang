package fibs_test

// Run as:
//  go test -bench=. -benchmem ./fun/q1/fibs

import (
	"fmt"
	"testing"

	"exp/fun/q1/fibs"
)

var inputs = []int{5, 10, 20}

func BenchmarkFib(b *testing.B) {
	for _, input := range inputs {
		b.Run(fmt.Sprintf("Fib input=%d", input), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				fibs.Fib(input)
			}
		})
	}
}

func BenchmarkFib2(b *testing.B) {
	for _, input := range inputs {
		b.Run(fmt.Sprintf("Fib2 input=%d", input), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				fibs.Fib2(input)
			}
		})
	}
}

func BenchmarkFib3(b *testing.B) {
	for _, input := range inputs {
		b.Run(fmt.Sprintf("Fib3 input=%d", input), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				fibs.Fib3(input)
			}
		})
	}
}

func BenchmarkFib4(b *testing.B) {
	b.Log("BenchmarkFib4")
	for _, input := range inputs {
		b.Run(fmt.Sprintf("Fib4 input=%d", input), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				fibs.Fib4(input)
			}
		})
	}
}
