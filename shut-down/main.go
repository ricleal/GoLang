package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// shutdown a goroutine gracefully

func main() {
	// Create a channel to receive termination signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Create a channel to signal the goroutine to stop
	stopCh := make(chan struct{})

	// Use a wait group to wait for goroutines to finish
	var wg sync.WaitGroup
	wg.Add(1)

	// Start your goroutine
	go myGoroutine(stopCh, &wg)

	// Wait for termination signal
	<-sigCh

	// Signal the goroutine to stop
	close(stopCh)

	// Wait for the goroutine to finish
	wg.Wait()

	fmt.Println("Graceful shutdown gracefully completed")
}

// myGoroutine is a dummy goroutine that does some work
// it runs forever until it receives a stop signal
func myGoroutine(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-stopCh:
			fmt.Println("Goroutine received stop signal")
			return
		default:
			// Do some work
			fmt.Println("Goroutine is running normally:", time.Now())
			time.Sleep(1 * time.Second)
		}
	}
}
