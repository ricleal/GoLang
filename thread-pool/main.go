package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/sync/semaphore"
)

const numJobs = 10
const numWorkers = 3

func worker(id int, jobs <-chan int, results chan<- int) {

	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	for j := range jobs {
		fmt.Println("worker", id, "started  job", j)
		time.Sleep(time.Duration(r.Intn(1000)) * time.Millisecond)
		fmt.Println("worker", id, "finished job", j)
		results <- j * 2
	}
}

func threadPool() {

	// Buffered channels
	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	for w := 1; w <= numWorkers; w++ {
		go worker(w, jobs, results)
	}

	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs)

	fmt.Println("Waiting for results")
	for a := 1; a <= numJobs; a++ {
		<-results
	}
	close(results)
	fmt.Println("All results received. Done!")
}

func semaphoreExample() {

	ctx := context.Background()

	// Buffered channels
	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	// Semaphore
	var s = semaphore.NewWeighted(numWorkers)

	for w := 1; w <= numWorkers; w++ {
		if err := s.Acquire(ctx, 1); err != nil {
			fmt.Println("Failed to acquire semaphore: %w", err)
			break
		}
		go func(w int, jobs <-chan int, results chan<- int) {
			defer s.Release(1)
			worker(w, jobs, results)
		}(w, jobs, results)
	}

	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs)

	fmt.Println("Waiting for results")
	for a := 1; a <= numJobs; a++ {
		<-results
	}
	fmt.Println("All results received. Done!")

}

func main() {
	threadPool()
	semaphoreExample()
}
