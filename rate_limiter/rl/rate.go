package main

// Rate limiter using golang.org/x/time/rate
// This is a simple rate limiter that limits the number of requests per second.
// The rate limiter is implemented using the golang.org/x/time/rate package.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RL is a rate limiter struct
type RL struct {
	client  *http.Client
	limiter *rate.Limiter
}

func NewRL(r, b int) *RL {
	return &RL{
		client:  &http.Client{},
		limiter: rate.NewLimiter(rate.Limit(r), b),
	}
}

func (rl *RL) Get(ctx context.Context, url string) map[string]interface{} {
	if !rl.limiter.Allow() {
		err := rl.limiter.Wait(ctx)
		if err != nil {
			panic(err)
		}
	}

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		panic(err)
	}
	resp, err := rl.client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// convert response to map[string]interface{}
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		panic(err)
	}
	return response
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// 10 requests per second
	rl := NewRL(10, 1)

	start := time.Now()
	wg := sync.WaitGroup{}
	for i := 0; i <= 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp := rl.Get(ctx, "https://httpbin.org/get")

			fmt.Printf("Message %3d - %5dms -> %s\n", i, time.Since(start).Milliseconds(), resp["origin"])
		}(i)
	}
	wg.Wait()
	fmt.Println("Done")
}
