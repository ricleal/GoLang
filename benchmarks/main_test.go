package main

import (
	"fmt"
	"testing"
)

// test beanchmark of factorial
var N []int = []int{1, 10, 100, 1000, 10000}

func BenchmarkFactorial(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, n := range N {
			b.Run(fmt.Sprintf("factorial(%d)", n), func(b *testing.B) {
				factorial(n)
			})
		}
	}
}

func BenchmarkFactorialIterative(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, n := range N {
			b.Run(fmt.Sprintf("factorial(%d)", n), func(b *testing.B) {
				factorialIterative(n)
			})
		}
	}
}
