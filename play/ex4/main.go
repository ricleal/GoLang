package main

// If fn fails $N$ times consecutively, the breaker flips to an Open state and
// immediately fails all subsequent calls for a cooldown duration of $X$ seconds without invoking fn.
// After $X$ seconds, it enters a Half-Open state where it allows one trial call.
// If it succeeds, the breaker closes; if it fails, it opens again.

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	N = 3
	X = time.Duration(3) * time.Second
)

type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

func (s State) String() string {
	return [...]string{"Closed", "Open", "HalfOpen"}[s]
}

var ErrCircuitOpen = errors.New("circuit breaker is open")

type CircuitBreaker struct {
	mu       sync.Mutex
	state    State
	nErrors  int
	openedAt time.Time
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{state: Closed}
}

func (c *CircuitBreaker) Call(fn func() error) error {
	c.mu.Lock()

	switch c.state {
	case Closed:
		c.mu.Unlock()
		err := fn()
		c.mu.Lock()
		if err != nil {
			c.nErrors++
			if c.nErrors >= N {
				c.state = Open
				c.openedAt = time.Now()
			}
			c.mu.Unlock()
			return err
		}
		// Success resets the consecutive-failure counter.
		c.nErrors = 0
		c.mu.Unlock()
		return nil

	case Open:
		if time.Since(c.openedAt) < X {
			// Still within cooldown — fail fast without calling fn.
			c.mu.Unlock()
			return ErrCircuitOpen
		}
		// Cooldown expired: this goroutine becomes the sole probe caller.
		// Setting state to HalfOpen *before* unlocking causes any concurrent
		// caller to hit the HalfOpen case below and fail fast immediately,
		// preventing multiple simultaneous probes.
		c.state = HalfOpen
		c.mu.Unlock()
		err := fn()
		c.mu.Lock()
		if err != nil {
			c.state = Open
			c.openedAt = time.Now()
			c.mu.Unlock()
			return err
		}
		c.state = Closed
		c.nErrors = 0
		c.mu.Unlock()
		return nil

	case HalfOpen:
		// A probe is already in flight — reject immediately.
		c.mu.Unlock()
		return ErrCircuitOpen

	default:
		c.mu.Unlock()
		return fmt.Errorf("unknown circuit breaker state: %v", c.state)
	}
}

// ── demo ─────────────────────────────────────────────────────────────────────

func main() {
	const (
		nGoroutines       = 4
		callsPerGoroutine = 12
		callInterval      = 600 * time.Millisecond
	)

	cb := NewCircuitBreaker()

	// Simulated AI inference API: times out on exactly the first N actual
	// invocations, then recovers — mirroring a brief upstream outage.
	var callCount atomic.Int32
	api := func() error {
		n := callCount.Add(1)
		time.Sleep(20 * time.Millisecond) // simulate network latency
		if n <= N {
			return fmt.Errorf("inference timeout (real call #%d)", n)
		}
		return nil
	}

	var (
		wg      sync.WaitGroup
		printMu sync.Mutex
		start   = time.Now()
	)

	logf := func(gid, attempt int, err error) {
		elapsed := time.Since(start).Truncate(time.Millisecond)
		printMu.Lock()
		defer printMu.Unlock()
		if err != nil {
			fmt.Printf("[+%-8v] G%d call %02d  FAIL  %v\n", elapsed, gid, attempt, err)
		} else {
			fmt.Printf("[+%-8v] G%d call %02d  OK\n", elapsed, gid, attempt)
		}
	}

	for g := 0; g < nGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Stagger goroutine starts to reduce initial contention.
			time.Sleep(time.Duration(id) * 50 * time.Millisecond)
			for i := 1; i <= callsPerGoroutine; i++ {
				err := cb.Call(api)
				logf(id, i, err)
				time.Sleep(callInterval)
			}
		}(g)
	}

	wg.Wait()
	fmt.Println("\ndone.")
}
