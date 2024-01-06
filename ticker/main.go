package main

import (
	"context"
	"log"
	"time"
)

// If the context is not done, wait first, then check again.
func t(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second) // Adjust the duration as needed
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context is cancelled, return the error
			log.Printf("WaitForDbToComeOnline got canceled.")
			return ctx.Err()
		case <-ticker.C:
			// Attempt to ping the database to check if it's online
			log.Printf("Ping database...")
		}
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("Starting t()")
	if err := t(ctx); err != nil {
		log.Printf("t() returned an error: %v", err)
	}
	log.Printf("t() finished")
}
