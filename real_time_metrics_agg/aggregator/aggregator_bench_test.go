package aggregator

import (
	"testing"
)

func BenchmarkRecord(b *testing.B) {
	a := New()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			a.Record("cpu", 50.0)
		}
	})
}

func BenchmarkRecordAndCheck(b *testing.B) {
	a := New()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			a.Record("cpu", 50.0)
			a.CheckAlerts()
		}
	})
}

func BenchmarkRecordAndAverages(b *testing.B) {
	a := New()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			a.Record("cpu", 50.0)
			a.Averages()
		}
	})
}

func BenchmarkMixedWorkload(b *testing.B) {
	a := New()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			a.Record("cpu", 50.0)
			a.Record("network", 500_000)
			a.Record("requests", 75.0)
			a.Averages()
			a.CheckAlerts()
		}
	})
}
