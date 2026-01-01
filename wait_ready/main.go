package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func waitUntilReady(ctx context.Context, name string, serviceStartDelay time.Duration) error {
	start := time.Now()

	log.Printf("[%s] Service will be ready in %v", name, serviceStartDelay)

	pollingInterval := 500 * time.Millisecond
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Context canceled after %v: %v", name, time.Since(start), ctx.Err())
			return ctx.Err()

		case <-ticker.C:
			elapsed := time.Since(start)
			log.Printf("[%s] Polling... (elapsed: %v)", name, elapsed.Round(time.Millisecond))

			// Service becomes ready after serviceStartDelay
			if elapsed >= serviceStartDelay {
				log.Printf("[%s] ✅ Service is READY! (took %v)", name, elapsed.Round(time.Millisecond))
				return nil
			}
		}
	}
}

func main() {
	ctx := context.Background()

	fmt.Println("=== Testing Wait Until Ready ===")
	fmt.Println()

	// Test 1: Context times out before service is ready (FAILS)
	fmt.Println("--- Test 1: Short timeout (will fail) ---")
	shortCtx, cancel1 := context.WithTimeout(ctx, 2*time.Second)
	defer cancel1()

	err := waitUntilReady(shortCtx, "FastTimeout", 5*time.Second)
	if err != nil {
		log.Printf("[FastTimeout] ❌ Failed: %v\n", err)
	}

	fmt.Println()
	time.Sleep(500 * time.Millisecond) // Brief pause between tests

	// Test 2: Context has enough time for service to be ready (SUCCESS)
	fmt.Println("--- Test 2: Long timeout (will succeed) ---")
	longCtx, cancel2 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel2()

	err = waitUntilReady(longCtx, "SlowTimeout", 3*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	fmt.Println("=== All tests completed ===")
}
