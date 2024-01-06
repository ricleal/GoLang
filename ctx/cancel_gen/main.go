package main

import (
	"context"
	"fmt"
	"sync"
)

// Example canceling a context propagated to a goroutine on exit.

func main() {
	// gen generates integers in a separate goroutine and
	// sends them to the returned channel.
	// The callers of gen need to cancel the context once
	// they are done consuming generated integers not to leak
	// the internal goroutine started by gen.
	gen := func(ctx context.Context, wg *sync.WaitGroup) <-chan int {
		dst := make(chan int)
		n := 1
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					fmt.Println("Go routine canceled")
					return // returning not to leak the goroutine
				case dst <- n:
					n++
				}
			}
		}()
		return dst
	}

	// Creating a WaitGroup to make sure the goroutine is finished
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel() // cancel when we are finished consuming integers
		wg.Wait()
		fmt.Println("All go routines finished. Main routine finished.")
	}()

	wg.Add(1)
	for n := range gen(ctx, &wg) {
		fmt.Println(n)
		if n == 5 {
			break
		}
	}
}
