package main

import (
	"fmt"
	"runtime"
)

func main() {
	input_channel := make(chan int, 100)
	output_channel := make(chan int, 100)

	number_of_cores := runtime.NumCPU()
	N := 45 // number of fib to calculate

	// Launch the workers
	for i := 0; i < number_of_cores-1; i++ {
		go worker(input_channel, output_channel)
	}
	// Fill the input chanel with values from 0 to N
	for i := 0; i < N; i++ {
		input_channel <- i
	}
	close(input_channel)
	// Get the results of the fib
	for j := 0; j < N; j++ {
		fmt.Println(<-output_channel)
	}
	close(output_channel)
}

func worker(input_channel <-chan int, output_channel chan<- int) {
	// iterate over the values previously put in the input channel
	for n := range input_channel {
		output_channel <- fib(n)
	}
}

// inefficient fibonnaci code
func fib(n int) int {
	if n <= 1 {
		return n
	}
	return fib(n-1) + fib(n-2)
}
