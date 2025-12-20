package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
)

// Concurrency (The "Go" Way)
// Producer-Consumer Pattern: * Goroutine 1 (Scanner): Reads the file line by line and sends lines into a chan string.

var (
	lines   = make(chan string)
	results = make(chan int) // worker id who processed the line
)

var agResults = make(map[int]int) // map[workerID]countLinesProcessed

// 1 scanner that reads the file and sends lines to the lines channel
func Scanner() {
	defer close(lines) // Close channel when done

	path := "/home/leal/Downloads/approval-workflow-backend-design.md"
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			// log.Println(line)
			lines <- line
		}
	}
}

// multiple parsers that read from lines channel and send results to results channel
func Parser(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			log.Println("leaving parser:", id)
			return
		case line, ok := <-lines:
			if !ok {
				// Channel closed, exit
				log.Println("parser done:", id)
				return
			}
			_ = line
			// process line
			results <- id
		}
	}
}

// single aggregator that reads from results channel and aggregates results
func Aggregator(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Leaving Aggregator")
			return
		case workerID, ok := <-results:
			if !ok {
				// Channel closed, exit
				log.Println("Aggregator done")
				return
			}
			agResults[workerID] += 1
		}
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go Scanner()

	var parserWg sync.WaitGroup
	var aggWg sync.WaitGroup

	// 5 workers
	for i := range 5 {
		parserWg.Go(func() {
			Parser(ctx, i)
		})
	}

	// Close results channel when all parsers are done
	go func() {
		parserWg.Wait()
		close(results)
	}()

	// Aggregator
	aggWg.Go(func() {
		Aggregator(ctx)
	})

	aggWg.Wait()

	// Aggregation results
	log.Println(agResults)
}
