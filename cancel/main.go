package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func Executor() {
	// Create a context and cancel function
	ctx, cancel := context.WithCancel(context.Background())

	// WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Number of goroutines to launch
	numGoroutines := 5

	// Channel to communicate the result
	resultCh := make(chan int)

	// Launch the goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go performTask(ctx, &wg, i, resultCh)
	}

	// Wait for a result from any goroutine
	target := waitForResult(resultCh)

	// Cancel the context to signal other goroutines to stop
	cancel()

	// Wait for all goroutines to finish
	wg.Wait()

	fmt.Printf("Objective achieved by goroutine %d 🥳\n", target)
}

func performTask(ctx context.Context, wg *sync.WaitGroup, id int, resultCh chan<- int) {
	defer wg.Done()

	// Simulate some work
	fmt.Printf("%d - Doing some work...🤔\n", id)
	time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)

	// The first goroutine to finish sends the result

	select {
	case <-ctx.Done():
		// Context cancelled, exit the goroutine
		fmt.Printf("Goroutine %d cancelled (%v) 😭\n", id, ctx.Err())
	case resultCh <- id:
		fmt.Printf("Goroutine %d sent the result 👌\n", id)
		// Objective achieved, send the result
	}
}

func waitForResult(resultCh <-chan int) int {
	// block until a result is received
	result := <-resultCh
	return result
}

func main() {
	Executor()
}
