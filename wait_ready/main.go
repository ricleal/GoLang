package main

import (
	"context"
	"log"
	"math/rand"
	"time"
)

const readyTimeout = 3500 * time.Millisecond

func waitUntilReady(ctx context.Context) error {
	r := rand.Intn(7)
	startDelay := time.Duration(r) * time.Second
	start := time.Now()

	log.Println("trying to be ready in", startDelay)

	pollingInterval := 1 * time.Second
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			log.Println("Polling for readiness every", pollingInterval)
			if time.Since(start) >= startDelay {
				return nil
			}
		}
	}
}

func main() {
	ctx := context.Background()

	waitCtx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()

	log.Println("waiting to be ready in less than", readyTimeout)
	err := waitUntilReady(waitCtx)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Ready")
}
