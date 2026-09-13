package main

// Run as: `go test -bench=. -benchmem ./pool`

import (
	"bytes"
	"testing"
)

// warmPool populates the current P's local pool so the first measured Get
// does not pay for the New() allocation.
func warmPool() {
	p := logBufferPool.Get().(*bytes.Buffer)
	p.Reset()
	logBufferPool.Put(p)
}

func BenchmarkPooledReuse(b *testing.B) {
	warmPool()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// TODO: implement the pooled reuse logic
		buf := logBufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		// Simulate some work with the buffer
		buf.WriteString("benchmark")
		logBufferPool.Put(buf)
	}
}

func BenchmarkPooledReuseBufferUsage(b *testing.B) {
	warmPool()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := logBufferPool.Get().(*bytes.Buffer)

		buf.WriteString(`{"time":"2024-06-01T12:00:00Z","number":"1","duration":"100ms"}`)
		buf.WriteString("\n")
		buf.WriteString(`{"time":"2024-06-01T12:00:00Z","number":"2","duration":"100ms"}`)
		buf.WriteString("\n")
		buf.WriteString(`{"time":"2024-06-01T12:00:00Z","number":"3","duration":"100ms"}`)
		buf.WriteString("\n")

		// _ = buf.String() // converting to string is an allocation

		buf.Reset()
		// Simulate some work with the buffer
		buf.WriteString("benchmark")
		logBufferPool.Put(buf)
	}
}

func BenchmarkPooledReuseMain(b *testing.B) {
	warmPool()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_main()
	}
}
