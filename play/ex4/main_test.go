package main

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var errAPI = errors.New("api error")

func succeed() error { return nil }
func fail() error    { return errAPI }

// getState reads the breaker state under the lock (safe from tests).
func getState(cb *CircuitBreaker) State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// ── Closed state ─────────────────────────────────────────────────────────────

func TestClosed_FnIsCalledOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker()
	called := false
	err := cb.Call(func() error { called = true; return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("fn was not called in Closed state")
	}
	if getState(cb) != Closed {
		t.Fatalf("expected Closed, got %v", getState(cb))
	}
}

func TestClosed_FailuresBeforeNDoNotTrip(t *testing.T) {
	cb := NewCircuitBreaker()
	for i := 0; i < N-1; i++ {
		if err := cb.Call(fail); !errors.Is(err, errAPI) {
			t.Fatalf("call %d: expected errAPI, got %v", i+1, err)
		}
	}
	if getState(cb) != Closed {
		t.Fatalf("expected Closed after %d failures, got %v", N-1, getState(cb))
	}
}

func TestClosed_TripsToOpenAfterNConsecutiveFailures(t *testing.T) {
	cb := NewCircuitBreaker()
	for i := 0; i < N; i++ {
		cb.Call(fail) //nolint:errcheck
	}
	if getState(cb) != Open {
		t.Fatalf("expected Open after %d failures, got %v", N, getState(cb))
	}
}

func TestClosed_SuccessResetsErrorCounter(t *testing.T) {
	cb := NewCircuitBreaker()
	// Build up N-1 failures.
	for i := 0; i < N-1; i++ {
		cb.Call(fail) //nolint:errcheck
	}
	// A success must reset the consecutive-failure counter.
	cb.Call(succeed) //nolint:errcheck

	// N-1 more failures should still not trip the breaker.
	for i := 0; i < N-1; i++ {
		cb.Call(fail) //nolint:errcheck
	}
	if getState(cb) != Closed {
		t.Fatalf("expected Closed after counter reset, got %v", getState(cb))
	}
}

// ── Open state ───────────────────────────────────────────────────────────────

func TestOpen_FailsFastWithoutCallingFn(t *testing.T) {
	cb := NewCircuitBreaker()
	for i := 0; i < N; i++ {
		cb.Call(fail) //nolint:errcheck
	}

	called := false
	err := cb.Call(func() error { called = true; return nil })

	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
	if called {
		t.Fatal("fn must not be called while the circuit is Open")
	}
}

func TestOpen_ProbeSucceeds_ClosesBreaker(t *testing.T) {
	cb := NewCircuitBreaker()
	for i := 0; i < N; i++ {
		cb.Call(fail) //nolint:errcheck
	}
	// Fast-forward past the cooldown window.
	cb.openedAt = time.Now().Add(-(X + time.Second))

	if err := cb.Call(succeed); err != nil {
		t.Fatalf("probe should succeed, got %v", err)
	}
	if getState(cb) != Closed {
		t.Fatalf("expected Closed after successful probe, got %v", getState(cb))
	}
}

func TestOpen_ProbeFails_ReopensWithFreshCooldown(t *testing.T) {
	cb := NewCircuitBreaker()
	for i := 0; i < N; i++ {
		cb.Call(fail) //nolint:errcheck
	}
	cb.openedAt = time.Now().Add(-(X + time.Second))

	before := time.Now()
	if err := cb.Call(fail); !errors.Is(err, errAPI) {
		t.Fatalf("expected errAPI from failed probe, got %v", err)
	}
	if getState(cb) != Open {
		t.Fatalf("expected Open after failed probe, got %v", getState(cb))
	}
	cb.mu.Lock()
	refreshed := cb.openedAt.After(before)
	cb.mu.Unlock()
	if !refreshed {
		t.Fatal("openedAt must be refreshed after a failed probe")
	}
}

// ── HalfOpen state ───────────────────────────────────────────────────────────

func TestHalfOpen_ConcurrentCallersRejectedImmediately(t *testing.T) {
	cb := NewCircuitBreaker()
	// Simulate a probe already in flight: breaker is in HalfOpen.
	cb.state = HalfOpen

	err := cb.Call(succeed)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen from concurrent caller, got %v", err)
	}
}

// ── Concurrency ──────────────────────────────────────────────────────────────

// TestConcurrency_NoRace is designed to be run with `go test -race`.
// It hammers the circuit breaker from many goroutines to surface any data races.
func TestConcurrency_NoRace(t *testing.T) {
	cb := NewCircuitBreaker()

	var count atomic.Int32
	api := func() error {
		if count.Add(1) <= int32(N) {
			return errAPI
		}
		return nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				cb.Call(api) //nolint:errcheck
			}
		}()
	}
	wg.Wait()
}
