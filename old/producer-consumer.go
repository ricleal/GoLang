package main

/* producer-consumer problem in Go */

import (
	"fmt"
	"time"
)

var done = make(chan bool)
var msgs = make(chan int)

func produce() {
	for i := 0; i < 4; i++ {
		fmt.Println("Producer: sending:", i)
		msgs <- i
		fmt.Println("Producer: sent!")
	}
	fmt.Println("Producer: Before closing channel")
	close(msgs)
	fmt.Println("Producer: Before passing true to done")
	done <- true
}

func consume() {
	for msg := range msgs {
		fmt.Println("Consumer: got:", msg)
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	go produce()
	go consume()
	<-done
}
