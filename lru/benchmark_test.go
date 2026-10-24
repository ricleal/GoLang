package main

import (
	"strconv"
	"testing"
)

func BenchmarkLRU(b *testing.B) {
	const capacity = 1024
	const keyCount = capacity * 2

	keys := make([]string, keyCount)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}
	keyMask := keyCount - 1

	b.Run("container-list", func(b *testing.B) {
		cache := New(capacity)
		for i := 0; i < capacity; i++ {
			cache.Add(keys[i], i)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.Add(keys[i&keyMask], i)
			cache.Get(keys[(i+capacity-1)&keyMask])
		}
	})

	b.Run("map-doubly-linked", func(b *testing.B) {
		cache := NewEnhanced(capacity)
		for i := 0; i < capacity; i++ {
			cache.Add(keys[i], i)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.Add(keys[i&keyMask], i)
			cache.Get(keys[(i+capacity-1)&keyMask])
		}
	})
}
