package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitedClient wraps http.Client with rate limiting
type RateLimitedClient struct {
	client  *http.Client
	limiter *rate.Limiter
}

// NewRateLimitedClient creates a new rate-limited HTTP client
// rps: requests per second, burst: maximum burst size
func NewRateLimitedClient(rps int, burst int) *RateLimitedClient {
	return &RateLimitedClient{
		client:  &http.Client{Timeout: 10 * time.Second},
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

// makeRequest performs a single HTTP request with metrics
func makeRequest(ctx context.Context, client *RateLimitedClient, id int, url string, results chan<- RequestResult) {
	totalStart := time.Now()

	// Wait for rate limiter
	err := client.limiter.Wait(ctx)
	if err != nil {
		results <- RequestResult{
			ID:       id,
			Duration: time.Since(totalStart),
			Success:  false,
			Error:    fmt.Sprintf("rate limiter wait failed: %v", err),
		}
		return
	}

	// Now make the actual HTTP request
	httpStart := time.Now()
	resp, err := client.client.Get(url)
	httpDuration := time.Since(httpStart)
	totalDuration := time.Since(totalStart)

	result := RequestResult{
		ID:           id,
		Duration:     totalDuration,
		HTTPDuration: httpDuration,
		Success:      err == nil,
	}

	if err != nil {
		result.Error = err.Error()
		results <- result
		return
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Read response body
	body, _ := io.ReadAll(resp.Body)
	result.BodySize = len(body)

	results <- result
}

// RequestResult holds the result of a single request
type RequestResult struct {
	ID           int
	Duration     time.Duration // Total time including rate limit wait
	HTTPDuration time.Duration // Actual HTTP request time
	StatusCode   int
	BodySize     int
	Success      bool
	Error        string
}

// printProgress displays real-time progress
func printProgress(completed, total int, startTime time.Time) {
	elapsed := time.Since(startTime)
	rps := float64(completed) / elapsed.Seconds()
	fmt.Printf("\r[Progress] %d/%d requests | %.2f req/s | Elapsed: %v",
		completed, total, rps, elapsed.Round(time.Millisecond))
}

func main() {
	fmt.Println("=== Outgoing HTTP Rate Limiter Demo ===")
	fmt.Println("Simulating AWS API rate limits")
	fmt.Println()

	// Configuration (simulating AWS API limits)
	const (
		targetURL     = "http://localhost:8080/get" // httpbin endpoint
		totalRequests = 50                          // Total requests to make
		rateLimit     = 5                           // Requests per second (AWS typical limit)
		burstSize     = 10                          // Burst capacity
		numWorkers    = 20                          // Parallel workers
	)

	fmt.Printf("Configuration:\n")
	fmt.Printf("  • Rate Limit: %d requests/second\n", rateLimit)
	fmt.Printf("  • Burst Size: %d requests\n", burstSize)
	fmt.Printf("  • Total Requests: %d\n", totalRequests)
	fmt.Printf("  • Parallel Workers: %d\n", numWorkers)
	fmt.Printf("  • Target: %s\n\n", targetURL)

	// Create rate-limited HTTP client
	client := NewRateLimitedClient(rateLimit, burstSize)

	// Channel for work distribution
	jobs := make(chan int, totalRequests)
	results := make(chan RequestResult, totalRequests)

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start workers
	var wg sync.WaitGroup
	fmt.Printf("Starting %d workers...\n", numWorkers)

	startTime := time.Now()

	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for id := range jobs {
				makeRequest(ctx, client, id, targetURL, results)
			}
		}(w)
	}

	// Send jobs
	for i := 1; i <= totalRequests; i++ {
		jobs <- i
	}
	close(jobs)

	// Collect results with progress monitoring
	var allResults []RequestResult
	var completedCount int
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Close results channel when workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results and show progress
	lastPrint := time.Now()
	for result := range results {
		allResults = append(allResults, result)
		completedCount++

		// Print progress every 100ms
		if time.Since(lastPrint) >= 100*time.Millisecond {
			printProgress(completedCount, totalRequests, startTime)
			lastPrint = time.Now()
		}
	}

	// Final progress print
	printProgress(completedCount, totalRequests, startTime)
	fmt.Println()

	// Analyze results
	var successCount, failCount int
	var totalHTTPDuration time.Duration
	statusCodes := make(map[int]int)

	for _, result := range allResults {
		if result.Success {
			successCount++
			totalHTTPDuration += result.HTTPDuration
			statusCodes[result.StatusCode]++
		} else {
			failCount++
			log.Printf("Request %d failed: %s", result.ID, result.Error)
		}
	}

	elapsed := time.Since(startTime)
	actualRPS := float64(successCount) / elapsed.Seconds()

	// Print summary
	fmt.Println("=== Results Summary ===")
	fmt.Printf("Total Time: %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Successful: %d/%d (%.1f%%)\n", successCount, totalRequests,
		float64(successCount)/float64(totalRequests)*100)
	fmt.Printf("Failed: %d\n", failCount)
	fmt.Printf("Actual Rate: %.2f req/s (limit: %d req/s)\n", actualRPS, rateLimit)

	if successCount > 0 {
		avgHTTPTime := totalHTTPDuration / time.Duration(successCount)
		fmt.Printf("Average HTTP Response Time: %v\n", avgHTTPTime.Round(time.Millisecond))
	} else {
		fmt.Println("Average HTTP Response Time: N/A (no successful requests)")
	}

	fmt.Println("\nStatus Codes:")
	for code, count := range statusCodes {
		fmt.Printf("  • %d: %d requests\n", code, count)
	}

	fmt.Println("\n=== Rate Limiter Analysis ===")
	// Calculate expected time accounting for burst:
	// First 'burst' requests are immediate, remaining requests throttled at rate limit
	var expectedTime float64
	if totalRequests <= burstSize {
		expectedTime = 0.1 // Nearly instant for requests within burst
	} else {
		// Burst requests are instant, remaining requests take (remaining / rate) seconds
		remainingRequests := totalRequests - burstSize
		expectedTime = float64(remainingRequests) / float64(rateLimit)
	}

	fmt.Printf("Expected time (with burst): %.2f seconds\n", expectedTime)
	fmt.Printf("Actual time: %.2f seconds\n", elapsed.Seconds())
	fmt.Printf("Burst allowance: first %d requests\n", burstSize)

	// Check if rate limiter is working correctly
	// The actual time should be >= expected time (within small tolerance for overhead)
	// Note: actualRPS can exceed rateLimit due to burst (first N requests are instant)
	tolerancePercent := 0.95
	isTimeCorrect := elapsed.Seconds() >= expectedTime*tolerancePercent

	if isTimeCorrect {
		fmt.Println("✅ Rate limiter working correctly!")
		fmt.Printf("   • Burst allowed %d immediate requests\n", burstSize)
		fmt.Printf("   • Remaining %d requests throttled at %d req/s\n", totalRequests-burstSize, rateLimit)
		fmt.Printf("   • Overall rate (%.2f req/s) > limit due to burst effect\n", actualRPS)
		fmt.Printf("   • This is expected: (burst + throttled) / total_time = higher average\n")
	} else {
		fmt.Println("⚠️  Rate limiter behavior unexpected")
		fmt.Printf("   • Completed in %.2fs, expected at least %.2fs\n", elapsed.Seconds(), expectedTime)
		fmt.Println("   • Rate limit may not be enforced properly")
	}

	fmt.Println("\n=== Key Takeaways ===")
	fmt.Println("• Rate limiter prevents overwhelming external APIs")
	fmt.Println("• Tokens are refilled at specified rate (5/sec)")
	fmt.Println("• Burst allows short bursts of traffic")
	fmt.Println("• Workers automatically throttled by rate limiter")
	fmt.Println("• Similar to AWS SDK rate limiting behavior")
}
