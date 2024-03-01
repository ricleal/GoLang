package hw_test

import (
	"exp/profile/hw"
	"testing"
)

func BenchmarkHardWork(b *testing.B) {
	for i := 0; i < b.N; i++ {
		hw.HardWork()
	}
}
