# Outgoing HTTP Rate Limiter

Demonstration of rate limiting for outgoing HTTP requests, simulating AWS API rate limits.

## Overview

This example shows how to implement rate limiting for outgoing HTTP requests using `golang.org/x/time/rate`. This is crucial when interacting with external APIs that impose rate limits (e.g., AWS, Stripe, GitHub).

## Architecture

```
┌─────────────┐
│   Workers   │ (20 parallel goroutines)
│  (Goroutines)│
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│  Rate Limiter   │ ← Token Bucket Algorithm
│  (5 req/s)      │   • Refills at 5 tokens/sec
│  Burst: 10      │   • Max 10 tokens
└────────┬────────┘
         │
         ▼
   ┌─────────┐
   │ httpbin │ (Docker container)
   └─────────┘
```

## Features

- **Token Bucket Rate Limiter**: Uses `golang.org/x/time/rate`
- **Configurable Limits**: Set requests per second and burst size
- **Parallel Workers**: 20 concurrent goroutines making requests
- **Real-time Progress**: Live monitoring of request throughput
- **Context Support**: Proper cancellation and timeouts
- **Detailed Metrics**: Success rate, actual RPS, response times

## Setup

### 1. Start httpbin Service

```bash
docker-compose up -d
```

Wait for the service to be healthy:
```bash
docker-compose ps
```

### 2. Install Dependencies

```bash
go mod tidy
```

### 3. Run the Rate Limiter

```bash
go run main.go
```

## Configuration

```go
const (
    rateLimit     = 5   // Requests per second (AWS typical limit)
    burstSize     = 10  // Maximum burst capacity
    totalRequests = 50  // Total requests to make
    numWorkers    = 20  // Parallel workers
)
```

## Expected Output

```
=== Outgoing HTTP Rate Limiter Demo ===
Simulating AWS API rate limits

Configuration:
  • Rate Limit: 5 requests/second
  • Burst Size: 10 requests
  • Total Requests: 50
  • Parallel Workers: 20
  • Target: http://localhost:8080/get

Starting 20 workers...
[Progress] 50/50 requests | 5.03 req/s | Elapsed: 9.94s

=== Results Summary ===
Total Time: 9.94s
Successful: 50/50 (100.0%)
Failed: 0
Actual Rate: 5.03 req/s (limit: 5 req/s)
Average Response Time: 12ms

Status Codes:
  • 200: 50 requests

=== Rate Limiter Analysis ===
Expected minimum time: 10.00 seconds
Actual time: 9.94 seconds
✅ Rate limiter working correctly - stayed within limits!

=== Key Takeaways ===
• Rate limiter prevents overwhelming external APIs
• Tokens are refilled at specified rate (5/sec)
• Burst allows short bursts of traffic
• Workers automatically throttled by rate limiter
• Similar to AWS SDK rate limiting behavior
```

## How It Works

### Token Bucket Algorithm

1. **Bucket Capacity**: Holds `burstSize` tokens (10)
2. **Refill Rate**: Adds tokens at `rateLimit` per second (5/s)
3. **Request Handling**:
   - Each request consumes 1 token
   - If tokens available → request proceeds immediately
   - If no tokens → waits until token available
   - Maximum wait determined by context timeout

### Flow

```
Initial burst of 10 requests → Immediate (burst capacity)
Next 40 requests → Throttled at 5 req/s
Total time ≈ 10 seconds (50 requests / 5 req/s)
```

## Real-World Use Cases

### AWS API Rate Limits

```go
// AWS S3: 3,500 PUT/COPY/POST/DELETE per second per prefix
client := NewRateLimitedClient(3500, 5000)

// AWS DynamoDB: 40,000 read capacity units per second
client := NewRateLimitedClient(40000, 50000)
```

### GitHub API

```go
// GitHub API: 5,000 requests per hour (≈1.4 req/s)
client := NewRateLimitedClient(1, 5) // Conservative
```

### Stripe API

```go
// Stripe: 100 requests per second
client := NewRateLimitedClient(100, 150)
```

## Testing Different Scenarios

### High Burst, Low Steady Rate

```go
// Allow quick bursts but throttle sustained traffic
client := NewRateLimitedClient(5, 50)
```

### Strict Rate Limit

```go
// No burst, strict enforcement
client := NewRateLimitedClient(5, 1)
```

### Very Restrictive

```go
// One request per second, no burst
client := NewRateLimitedClient(1, 1)
```

## Cleanup

```bash
docker-compose down
```

## Key Concepts

1. **Rate Limiter**: Controls the rate of operations
2. **Burst**: Allows short-term spikes above steady rate
3. **Token Bucket**: Algorithm that refills tokens over time
4. **Wait vs Allow**:
   - `Wait()`: Blocks until token available
   - `Allow()`: Returns false immediately if no token

## Comparison with Other Approaches

### Without Rate Limiter

```go
// ❌ Can overwhelm API, get 429 errors
for i := 0; i < 1000; i++ {
    go makeRequest()
}
```

### With Rate Limiter

```go
// ✅ Respects API limits, smooth traffic
limiter := rate.NewLimiter(5, 10)
for i := 0; i < 1000; i++ {
    limiter.Wait(ctx)
    go makeRequest()
}
```

## Monitoring

The implementation includes:
- Real-time progress display
- Success/failure tracking
- Actual vs expected rate comparison
- Response time metrics
- HTTP status code distribution

## Best Practices

1. **Set Conservative Limits**: Start below API limits
2. **Add Burst Capacity**: Handle temporary spikes
3. **Use Context**: Enable cancellation and timeouts
4. **Monitor Metrics**: Track actual vs configured rates
5. **Handle Errors**: Implement retry logic for rate limit errors
6. **Per-Resource Limits**: Different limiters for different endpoints

## References

- [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate)
- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
- [AWS API Rate Limits](https://docs.aws.amazon.com/general/latest/gr/api-retries.html)
