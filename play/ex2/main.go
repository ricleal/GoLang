package main

import (
	"fmt"
	"sync"
)

// The Problem:
// Write a function that takes an arbitrary number of read-only channels (<-chan Message) and merges them into a single output channel.
// The output channel must close cleanly when all input channels are exhausted.

type Message struct {
	Data string
}

func mergeChannels(channels ...<-chan Message) <-chan Message {
	out := make(chan Message)
	var wg sync.WaitGroup

	for _, ch := range channels {
		wg.Go(func() {
			for msg := range ch {
				out <- msg
			}
		})
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func main() {
	var channels [10]<-chan Message
	for i := range channels {
		ch := make(chan Message, 1)
		ch <- Message{Data: fmt.Sprintf("Ch %d", i)}
		close(ch)
		channels[i] = ch
	}

	merged := mergeChannels(channels[:]...)
	for msg := range merged {
		println(msg.Data)
	}
}
