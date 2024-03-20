package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
)

const URL = "https://txmchallenge-json.netlify.app/0.json"

///

type ConcurrentSet struct {
	m  map[string]struct{}
	mu sync.Mutex
}

func NewConcurrentSet() *ConcurrentSet {
	return &ConcurrentSet{
		m: make(map[string]struct{}),
	}
}

func (c *ConcurrentSet) Add(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[s] = struct{}{}
}

func (c *ConcurrentSet) Contains(s string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.m[s]
	return ok
}

// /
type ConcurrentQueue struct {
	q  []string
	mu sync.Mutex
}

func NewConcurrentQueue() *ConcurrentQueue {
	return &ConcurrentQueue{
		q: make([]string, 0),
	}
}

func (c *ConcurrentQueue) Push(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.q = append(c.q, s)
}

func (c *ConcurrentQueue) PushAll(ss []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.q = append(c.q, ss...)
}

func (c *ConcurrentQueue) Pop() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.q) == 0 {
		return ""
	}
	s := c.q[0]
	c.q = c.q[1:]
	return s
}

///

type Crawler struct {
	visited *ConcurrentSet
}

func NewCrawler() *Crawler {
	return &Crawler{
		visited: NewConcurrentSet(),
	}
}

func (c *Crawler) getURLs(url string) ([]string, error) {
	// Get the content of the URL
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get the content of the URL: %v", err)
	}
	defer resp.Body.Close()
	var data []string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode the content of the URL: %v", err)
	}
	return data, err
}

func (c *Crawler) Crawl(ctx context.Context, url string) {
	queue := NewConcurrentQueue()
	queue.Push(url)

	sem := semaphore.NewWeighted(3)
	wg := sync.WaitGroup{}

	var runningCount atomic.Int32

	for {
		url = queue.Pop()
		if url == "" {
			if runningCount.Load() > 0 {
				time.Sleep(1 * time.Millisecond)
				continue
			}
			break
		}
		if c.visited.Contains(url) {
			continue
		}
		// check if the URL starts with https
		if url[:5] != "https" {
			continue
		}
		c.visited.Add(url)

		log.Println(url)

		wg.Add(1)
		runningCount.Add(1)
		go func(url string) {
			defer wg.Done()

			defer runningCount.Add(-1)

			if err := sem.Acquire(ctx, 1); err != nil {
				log.Printf("FYI: failed to acquire semaphore: %v", err)
				return
			}
			defer sem.Release(1)

			urls, err := c.getURLs(url)
			if err != nil {
				log.Printf("FYI: failed to get the URLs: %v", err)
			}
			queue.PushAll(urls)
		}(url)
	}
	wg.Wait()
}

func main() {
	c := NewCrawler()
	c.Crawl(context.Background(), URL)
}
