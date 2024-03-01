package hw

import (
	"fmt"
	"time"
)

// Some function that does work
func HardWork() {
	start := time.Now()
	fmt.Printf("Starting hard work...")

	// Memory
	a := memWork()
	fmt.Printf("Memory work done in %v (len: %v)\n", time.Since(start), len(a))

	// CPU
	cpuWork()

	fmt.Printf("CPU work done in %v\n", time.Since(start))
}

func cpuWork() {
	for i := 0; i < 100000000; i++ {
		_ = i * i
	}
}

func memWork() []string {
	a := []string{}
	for i := 0; i < 5000000; i++ {
		a = append(a, "some string")
	}
	return a
}
