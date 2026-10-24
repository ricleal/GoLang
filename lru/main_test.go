package main

import "testing"

func TestAddUpdatesValueAndRecency(t *testing.T) {
	cache := New(2)
	cache.Add("a", 1)
	cache.Add("b", 2)
	cache.Add("a", 3)
	cache.Add("c", 4)

	if got := cache.Get("a"); got != 3 {
		t.Fatalf("Get(a) = %v, want 3", got)
	}
	if got := cache.Get("b"); got != nil {
		t.Fatalf("Get(b) = %v, want nil after eviction", got)
	}
	if got := cache.Get("c"); got != 4 {
		t.Fatalf("Get(c) = %v, want 4", got)
	}
}

func TestAddWithNonPositiveCapacity(t *testing.T) {
	for _, capacity := range []int{0, -1} {
		cache := New(capacity)
		cache.Add("a", 1)
		if got := cache.Get("a"); got != nil {
			t.Fatalf("capacity %d: Get(a) = %v, want nil", capacity, got)
		}
	}
}

func TestEnhancedAddUpdatesValueAndRecency(t *testing.T) {
	cache := NewEnhanced(2)
	cache.Add("a", 1)
	cache.Add("b", 2)
	cache.Add("a", 3)
	cache.Add("c", 4)

	if got := cache.Get("a"); got != 3 {
		t.Fatalf("Get(a) = %v, want 3", got)
	}
	if got := cache.Get("b"); got != nil {
		t.Fatalf("Get(b) = %v, want nil after eviction", got)
	}
	if got := cache.Get("c"); got != 4 {
		t.Fatalf("Get(c) = %v, want 4", got)
	}
}

func TestEnhancedAddWithNonPositiveCapacity(t *testing.T) {
	for _, capacity := range []int{0, -1} {
		cache := NewEnhanced(capacity)
		cache.Add("a", 1)
		if got := cache.Get("a"); got != nil {
			t.Fatalf("capacity %d: Get(a) = %v, want nil", capacity, got)
		}
	}
}

func ExampleNew() {
	cache := New(2)
	cache.Add("a", 1)
	cache.Add("b", 2)
	cache.Get("a")
	cache.Add("c", 3)
	cache.Print()
	// Output:
	// --------------------
	// a : 1
	// c : 3
}

func ExampleNewEnhanced() {
	cache := NewEnhanced(2)
	cache.Add("a", 1)
	cache.Add("b", 2)
	cache.Get("a")
	cache.Add("c", 3)
	cache.Print()
	// Output:
	// --------------------
	// a : 1
	// c : 3
}
