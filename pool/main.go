package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

type chanContent struct {
	buf          *bytes.Buffer
	workerNumber int
}

var (
	logChannel chan chanContent
	jobNumber  chan int
)

const (
	nWorkers = 5
	nJobs    = 100
)

var logBufferPools [nWorkers]sync.Pool

func init() {
	for i := 0; i < nWorkers; i++ {
		logBufferPools[i] = sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		}
	}
}

func worker(ctx context.Context, workerNumber int, job <-chan int) {
	for {
		select {
		case _, ok := <-job:
			if !ok {
				fmt.Printf("Worker %d exiting\n", workerNumber)
				return
			}

			startTime := time.Now()
			// do some processing
			time.Sleep(time.Millisecond * 100) // Simulate some processing delay
			endTime := time.Since(startTime)

			// 2. Fetch a clean buffer from the pool for logging
			buf := logBufferPools[workerNumber].Get().(*bytes.Buffer)

			// 3. Format a JSON log string directly into our reused buffer without allocations
			buf.WriteString(`{"time":"`)
			buf.WriteString(startTime.Format(time.RFC3339))
			buf.WriteString(`","number":"`)
			buf.WriteString(fmt.Sprintf("%d", workerNumber))
			buf.WriteString(`","duration":"`)
			buf.WriteString(endTime.String())
			buf.WriteString(`"}`)
			buf.WriteString("\n")

			// 4. Send the log entry to the log channel.
			// Ownership of the buffer moves to the consumer, which writes it and then
			// returns it to the pool. The producer must NOT Reset/Put it here: sending a
			// value copy (*buf) would share the backing array with the consumer, and a
			// later producer would overwrite the bytes while writeLog is still writing them.
			logChannel <- chanContent{buf: buf, workerNumber: workerNumber}

		case <-ctx.Done():
			return
		}
	}
}

func writeLog() {
	logFilePath := "/tmp/app.log"
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for logEntry := range logChannel {
		buf := logEntry.buf
		_, err := f.Write(buf.Bytes())
		if err != nil {
			fmt.Println("Error writing log:", err)
		}

		// The consumer owns the buffer now that it has been written out.
		// Reset it (keeping the allocated capacity) and return it to the pool.
		buf.Reset()
		logBufferPools[logEntry.workerNumber%nWorkers].Put(buf)
	}
}

func _main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logChannel = make(chan chanContent, nWorkers)
	jobNumber = make(chan int, nJobs)

	var logWg sync.WaitGroup
	logWg.Go(func() {
		writeLog()
	})

	fmt.Println("Starting workers...")
	var wg sync.WaitGroup
	for i := 0; i < nWorkers; i++ {
		wg.Go(func() {
			worker(ctx, i, jobNumber)
		})
	}
	fmt.Println("Dispatching jobs...")
	for j := 0; j < nJobs; j++ {
		jobNumber <- j
	}
	close(jobNumber)
	fmt.Println("All jobs dispatched, waiting for workers to finish...")
	wg.Wait()
	close(logChannel)
	fmt.Println("All workers finished, waiting for log writer to finish...")
	logWg.Wait()

	// ctx.Done()
	fmt.Println("All done.")
}

func main() {
	_main()
}
