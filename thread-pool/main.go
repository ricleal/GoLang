package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

const (
	numJobs    = 10
	numWorkers = 3
)

// This code demonstrates two approaches to implementing a **worker pool pattern** in Go:

// 1. **Traditional Thread Pool**: Uses buffered channels and goroutines to process jobs concurrently with a fixed number of workers
// 2. **Semaphore-based Pool**: Uses `golang.org/x/sync/semaphore` to control concurrent worker access

// Both approaches limit concurrency to 3 workers processing 10 jobs, preventing resource exhaustion.

func worker(id int, jobs <-chan int, results chan<- int) {
	for j := range jobs {
		fmt.Printf("Worker %d started  job %d\n", id, j)

		// Simulate work with random delay
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

		fmt.Printf("Worker %d finished job %d\n", id, j)
		results <- j * 2
	}
}

// threadPool demonstrates the classic worker pool pattern using channels
func threadPool() {
	fmt.Println("\n=== Thread Pool Pattern (Classic Channels) ===")

	// Buffered channels to hold jobs and results
	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	// Start workers
	for w := 1; w <= numWorkers; w++ {
		go worker(w, jobs, results)
	}

	// Send jobs
	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs)

	// Collect results
	fmt.Println("Waiting for results...")
	for a := 1; a <= numJobs; a++ {
		result := <-results
		fmt.Printf("  Result: %d\n", result)
	}
	close(results)
	fmt.Println("✅ All results received. Done!")
}

// semaphoreExample demonstrates worker pool using semaphore for concurrency control
func semaphoreExample() {
	fmt.Println("\n=== Semaphore-Based Pool ===")

	ctx := context.Background()

	// Buffered channels
	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	// Semaphore to limit concurrent workers
	sem := semaphore.NewWeighted(numWorkers)

	// WaitGroup to wait for all workers to complete
	var wg sync.WaitGroup

	// Send jobs first (before starting workers)
	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs)

	// Start workers with semaphore control
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Acquire semaphore slot
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Printf("Failed to acquire semaphore: %v", err)
				return
			}
			defer sem.Release(1)

			// Process jobs
			worker(id, jobs, results)
		}(w)
	}

	// Close results channel after all workers complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	fmt.Println("Waiting for results...")
	for result := range results {
		fmt.Printf("  Result: %d\n", result)
	}
	fmt.Println("✅ All results received. Done!")
}

func main() {
	// Note: As of Go 1.20+, the global random generator is automatically seeded
	// No need to call rand.Seed() anymore

	threadPool()
	semaphoreExample()

	fmt.Println("\n=== Summary ===")
	fmt.Printf("Total jobs: %d\n", numJobs)
	fmt.Printf("Worker pool size: %d\n", numWorkers)
	fmt.Println("Both patterns successfully limited concurrency!")
}
