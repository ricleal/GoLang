package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Graceful shutdown demonstration with multiple approaches:
// 1. Using channels and WaitGroup (classic approach)
// 2. Using context.Context (modern approach)

func main() {
	fmt.Println("=== Graceful Shutdown Demo ===")
	fmt.Println("Press Ctrl+C to trigger graceful shutdown")
	fmt.Println()

	// Create a channel to receive termination signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Create a channel to signal goroutines to stop
	stopCh := make(chan struct{})

	// Create a context for timeout-based shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a wait group to wait for goroutines to finish
	var wg sync.WaitGroup

	// Start multiple goroutines to demonstrate coordinated shutdown
	wg.Add(3)
	go worker("Worker-1", stopCh, &wg)
	go worker("Worker-2", stopCh, &wg)
	go contextWorker("ContextWorker", ctx, &wg)

	// Wait for termination signal
	fmt.Println("Application running... waiting for signal")
	sig := <-sigCh
	fmt.Printf("\n✋ Received signal: %v\n", sig)
	fmt.Println("Initiating graceful shutdown...")

	// Start shutdown with timeout
	shutdownTimeout := 5 * time.Second
	shutdownStart := time.Now()

	// Signal all goroutines to stop
	close(stopCh)
	cancel() // Cancel context for context-based workers

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		elapsed := time.Since(shutdownStart)
		fmt.Printf("\n✅ Graceful shutdown completed successfully in %v\n", elapsed)
	case <-time.After(shutdownTimeout):
		fmt.Printf("\n⚠️  Shutdown timeout after %v - forcing exit\n", shutdownTimeout)
	}

	fmt.Println("Application exited.")
}

// worker demonstrates classic channel-based graceful shutdown
func worker(name string, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer fmt.Printf("[%s] Shutdown complete\n", name)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			fmt.Printf("[%s] Received stop signal, cleaning up...\n", name)

			// Simulate cleanup work
			time.Sleep(500 * time.Millisecond)
			return

		case <-ticker.C:
			fmt.Printf("[%s] Working... ⚙️\n", name)
		}
	}
}

// contextWorker demonstrates context-based graceful shutdown
func contextWorker(name string, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer fmt.Printf("[%s] Shutdown complete\n", name)

	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] Context cancelled: %v, cleaning up...\n", name, ctx.Err())

			// Simulate cleanup work
			time.Sleep(500 * time.Millisecond)
			return

		case <-ticker.C:
			fmt.Printf("[%s] Processing with context... 🔄\n", name)
		}
	}
}
