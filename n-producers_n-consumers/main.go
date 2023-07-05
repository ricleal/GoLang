package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var ch chan int
var quit chan struct{}
var wgProd, wgCons sync.WaitGroup

// Consumer
type Consumer struct {
	Name string
}

func (c *Consumer) consume(v int) {
	defer wgCons.Done()
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("%s consume <- %d\n", c.Name, v)
}

// Producer
type Producer struct {
	Name string
}

func (p *Producer) produce() {
	defer wgProd.Done()
	v := rand.Intn(1000)
	time.Sleep(10 * time.Millisecond)
	ch <- v
	fmt.Printf("%s produce -> %d\n", p.Name, v)
}

// Registry
type Registry struct {
	Consumers []*Consumer
}

func (r *Registry) Register(c *Consumer) {
	r.Consumers = append(r.Consumers, c)
}

func (r *Registry) FanOut(ch chan int, quit chan struct{}) {
	for {
		select {
		case v := <-ch:
			for _, c := range r.Consumers {
				wgCons.Add(1)
				go c.consume(v)
			}
		case <-quit:
			close(ch)
			return
		}
	}
}

// main
func main() {
	fmt.Println("Start...")

	ch = make(chan int)
	quit = make(chan struct{})

	// 5 producers
	for i := 0; i < 5; i++ {
		wgProd.Add(1)
		p := Producer{
			Name: fmt.Sprintf("producer-%d", i),
		}
		go p.produce()
	}

	// 5 consumers
	register := &Registry{}
	for i := 0; i < 5; i++ {
		register.Register(
			&Consumer{
				Name: fmt.Sprintf("consumer-%d", i),
			},
		)
	}
	go register.FanOut(ch, quit)

	fmt.Println("Wait for producers")
	wgProd.Wait()
	quit <- struct{}{}
	close(quit)
	fmt.Println("Wait for consumers")
	wgCons.Wait()
	fmt.Println("Done!")
}
