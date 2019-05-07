package main

import (
	"fmt"
	"runtime"
)

// represents one value and the respective fibonacci value
type fib_data struct {
	value     int
	fib_value int
}

func main() {
	input_channel := make(chan int, 100)
	output_channel := make(chan fib_data, 100)

	number_of_cores := runtime.NumCPU()
	N := 40 // number of fib to calculate

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
		fib_pair := <-output_channel
		fmt.Println(fib_pair.value, "->", fib_pair.fib_value)
	}
	close(output_channel)
}

func worker(input_channel <-chan int, output_channel chan<- fib_data) {
	// iterate over the values previously put in the input channel
	for n := range input_channel {
		output_channel <- fib_data{n, fib(n)}
	}
}

// inefficient fibonacci calculation
func fib(n int) int {
	if n <= 1 {
		return n
	}
	return fib(n-1) + fib(n-2)
}
