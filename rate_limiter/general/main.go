package main

// Rate limiter using a custom implementation
// This is a simple rate limiter that limits the number of requests per second.
// The rate limiter is implemented using a custom implementation.

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type RateLimiter struct {
	maxMessages int
	windowInSec int
	mu          sync.Mutex

	messageCount int
	lastReset    time.Time
}

func NewRateLimiter(maxMessages int, windowInSec int) *RateLimiter {
	return &RateLimiter{
		maxMessages: maxMessages,
		windowInSec: windowInSec,
		lastReset:   time.Now(),
	}
}

func (r *RateLimiter) Allow() (bool, time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if r.messageCount < r.maxMessages {
		r.messageCount++
		return true, now
	}
	if now.Sub(r.lastReset) > time.Duration(r.windowInSec)*time.Second {
		r.messageCount -= r.maxMessages
		if r.messageCount < 0 {
			r.messageCount = 0
		}
		r.lastReset = now
	}
	// c is the number of windows that have passed since the last reset
	c := math.Ceil(float64(r.messageCount+1) / float64(r.maxMessages))

	t := r.lastReset.Add(time.Duration(float64(r.windowInSec)*c)*time.Second - time.Duration(r.windowInSec)*time.Second)

	r.messageCount++
	return false, t
}

func main() {
	// 100 messages per 10 seconds
	limiter := NewRateLimiter(100, 10)

	for i := 0; i <= 500; i++ {
		allow, nextBatchTime := limiter.Allow()
		fmt.Printf("Message %d (%v): %.0fs (%s)\n", i, allow, time.Until(nextBatchTime).Seconds(), nextBatchTime.Format("15:04:05"))
	}

	limiter = NewRateLimiter(10, 1)

	for i := 0; i <= 50; i++ {
		allow, nextBatchTime := limiter.Allow()
		fmt.Printf("Message %d (%v): %.0fs (%s)\n", i, allow, time.Until(nextBatchTime).Seconds(), nextBatchTime.Format("15:04:05"))
		if i%10 == 0 {
			time.Sleep(1 * time.Second)
		}
	}
}
