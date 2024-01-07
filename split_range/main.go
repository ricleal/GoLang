package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"golang.org/x/sync/semaphore"
)

const (
	workersCount = 10
)

func main() {
	// dataSize is a random number
	dataSize := 100000 + rand.Intn(1000)

	data := make([]int, dataSize)
	offset := dataSize / workersCount // Calculate offset for each worker

	var m sync.Mutex

	sem := semaphore.NewWeighted(int64(workersCount)) // Create semaphore for concurrency control

	var wg sync.WaitGroup
	wg.Add(dataSize)

	fmt.Println("Starting workers: dataSize =", dataSize, "offset =", offset, "workersCount =", workersCount)
	for i := 0; i < dataSize; i++ {
		index := (i % workersCount) * offset
		if index >= dataSize {
			break
		}

		go func(index int) {
			defer sem.Release(1) // Release the permit
			defer wg.Done()
			sem.Acquire(context.Background(), 1) // Acquire a permit
			if (index % 10000) == 0 {
				fmt.Println("Worker", index, "started")
			}
			m.Lock()
			data[index] = index
			m.Unlock()
		}(i)
	}
	fmt.Println("Waiting for workers to finish...")
	wg.Wait() // Wait for all workers to finish
	// Check if data is correct
	fmt.Print("Checking data:: ")
	for i := 0; i < dataSize; i++ {
		if data[i] != i {
			fmt.Println("❌ Error at index:", i)
			break
		}
	}
	fmt.Println("✅")
	fmt.Println("Done!")
}
