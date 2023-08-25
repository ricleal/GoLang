package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

type CircularList struct {
	items []any
	index int
	m     sync.Mutex
}

func (cl *CircularList) Next() any {
	cl.m.Lock()
	defer cl.m.Unlock()
	if len(cl.items) == 0 {
		return nil
	}
	item := cl.items[cl.index]
	cl.index = (cl.index + 1) % len(cl.items)
	return item
}

func (cl *CircularList) Add(item any) {
	cl.m.Lock()
	defer cl.m.Unlock()
	cl.items = append(cl.items, item)
}

func (cl *CircularList) Remove(item any) {
	cl.m.Lock()
	defer cl.m.Unlock()
	for i, v := range cl.items {
		if v == item {
			cl.items = append(cl.items[:i], cl.items[i+1:]...)
			return
		}
	}
}

func (cl *CircularList) Len() int {
	cl.m.Lock()
	defer cl.m.Unlock()
	return len(cl.items)
}

type LoadBalancer struct {
	servers *CircularList
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		servers: &CircularList{},
	}
}

func (lb *LoadBalancer) AddServer(server any) {
	lb.servers.Add(server)
}

func (lb *LoadBalancer) RemoveServer(server any) {
	lb.servers.Remove(server)
}

func (lb *LoadBalancer) NextServer() any {
	return lb.servers.Next()
}

var maxWorkers = 3

func main() {
	ctx := context.Background()
	lb := NewLoadBalancer()
	lb.AddServer("server11")
	lb.AddServer("server12")
	lb.AddServer("server13")

	// simulate 10 requests in parallel with a max of 3 workers
	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(maxWorkers))
	for i := 0; i < 10; i++ {
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Printf("Failed to acquire semaphore: %v", err)
			break
		}
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()
			defer sem.Release(1)
			server := lb.NextServer()
			log.Printf("Request %d served by %s\n", requestID, server)
			// simulate request processing time with a random duration
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

			if requestID%3 == 0 {
				newServer := fmt.Sprintf("server%d", requestID)
				log.Printf("Adding new server %s\n", newServer)
				lb.AddServer(newServer)
				return
			}
		}(i)
	}
	wg.Wait()
	log.Println("Done")
}
