package main

import "fmt"

// factorial(n) = n * factorial(n-1)
func factorial(n int) int {
	if n == 0 {
		return 1
	}
	return n * factorial(n-1)
}

// factorialIterative(n) = n * (n-1) * (n-2) * ... * 1
func factorialIterative(n int) int {
	result := 1
	for i := 1; i <= n; i++ {
		result *= i
	}
	return result
}

func main() {
	fmt.Println("Start...")
	fmt.Printf("factorial(5) = %d\n", factorial(5))
	fmt.Printf("factorial_iterative(5) = %d\n", factorialIterative(5))
	fmt.Println("End...")
}
