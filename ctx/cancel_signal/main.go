package main

import (
	"context"
	"log"
	"math/rand"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Example of graceful shutdown using a context and a WaitGroup
// Consumer launches a number of workers and waits for a signal to shutdown

func Consumer(ctx context.Context) {

	var wg sync.WaitGroup
	for workerID := 0; workerID < 5; workerID++ {
		wg.Add(1)
		go worker(ctx, &wg, workerID)
	}

	// block until a signal is received
	<-ctx.Done()

	log.Printf("Signal received: '%v'. Waiting for workers. Shutting down...\n", ctx.Err())
	// give workers time to finish
	wg.Wait()
	log.Printf("Shutdown complete ✅")
}

func worker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()
	for {
		log.Printf("Worker %d: Waiting for task...\n", workerID)
		processTask(ctx, workerID)

		// Check if context is cancelled, exit the goroutine
		select {
		case <-ctx.Done():
			// Context cancelled, exit the goroutine
			log.Printf("Worker %d: Goroutine cancelled 👋\n", workerID)
			return
		default:
			// Continue
		}
	}
}

func processTask(ctx context.Context, workerID int) error {
	log.Printf("Worker %d: Processing task 👨‍🏭 ...\n", workerID)
	r := rand.Intn(1000)
	time.Sleep(time.Duration(r) * time.Millisecond)
	return nil
}

func main() {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()
	Consumer(ctx)
}
