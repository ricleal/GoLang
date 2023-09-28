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

func f(wg *sync.WaitGroup) int {
	defer wg.Done()
	r := rand.Intn(5)
	fmt.Println("f started, sleeping", r, "seconds")
	time.Sleep(time.Duration(r) * time.Second)
	fmt.Println("f finished")
	return r
}

func main() {

	wg := sync.WaitGroup{}
	ch := make(chan int)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			ch <- f(&wg)
		}()
	}

	// Read the channel
	go func() {
		for v := range ch {
			fmt.Println("read value", v, "from ch")
		}
	}()

	fmt.Println("waiting for goroutines to finish")
	wg.Wait()
	close(ch)
	fmt.Println("all goroutines finished")
}
