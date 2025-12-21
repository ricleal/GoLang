package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Test to see if when the channel is full, the function filling the channel
// blocks
// Answer: No, the function does not block. It continues to run

func f() int {
	r := rand.Intn(5)
	fmt.Println("f started, sleeping", r, "seconds")
	time.Sleep(time.Duration(r) * 300 * time.Millisecond)
	fmt.Println("f finished")
	return r
}

func main() {
	var wg sync.WaitGroup
	ch := make(chan int)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch <- f()
		}()
	}

	// Read the channel
	wdRead := sync.WaitGroup{}
	wdRead.Add(1)
	go func() {
		defer wdRead.Done()
		for v := range ch {
			fmt.Println("\tread value", v, "from ch")
		}
	}()

	fmt.Println("waiting for goroutines to finish")
	wg.Wait()
	close(ch)
	wdRead.Wait()
	fmt.Println("all goroutines finished")
}
