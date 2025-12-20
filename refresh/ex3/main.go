package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
)

var (
	lines     = make(chan string)
	m         = sync.Mutex{}
	agResults = make(map[int]int) // map[workerID]countLinesProcessed
)

// 1 scanner that reads the file and sends lines to the lines channel
func Scanner(path string) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines <- line
		}
	}

	close(lines) // Close channel when done
}

func Parser(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Canceling worker", id)
			return
		case line, ok := <-lines:
			if !ok {
				log.Println("Channel closed. Exit:", id)
				return
			}
			c := len(line)

			m.Lock()
			agResults[id] += c
			m.Unlock()
		}
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	path := "/home/leal/Downloads/approval-workflow-backend-design.md"
	var wgScanner sync.WaitGroup
	var wgParser sync.WaitGroup

	wgScanner.Go(
		func() {
			Scanner(path)
		})

	for i := range 10 {
		wgParser.Go(func() {
			Parser(ctx, i)
		})
	}

	wgScanner.Wait()
	wgParser.Wait()

	fmt.Println(agResults)
}
