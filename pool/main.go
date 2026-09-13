package main

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"
)

var logChannel chan *bytes.Buffer

var logBufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func dummyProcess(number int) {
	startTime := time.Now()
	// do some processing
	time.Sleep(time.Millisecond * 100) // Simulate some processing delay
	endTime := time.Since(startTime)

	// 2. Fetch a clean buffer from the pool for logging
	buf := logBufferPool.Get().(*bytes.Buffer)

	// 3. Format a JSON log string directly into our reused buffer without allocations
	buf.WriteString(`{"time":"`)
	buf.WriteString(startTime.Format(time.RFC3339))
	buf.WriteString(`","number":"`)
	buf.WriteString(fmt.Sprintf("%d", number))
	buf.WriteString(`","duration":"`)
	buf.WriteString(endTime.String())
	buf.WriteString(`"}`)
	buf.WriteString("\n")

	// 4. Send the log entry to the log channel.
	// Ownership of the buffer moves to the consumer, which writes it and then
	// returns it to the pool. The producer must NOT Reset/Put it here: sending a
	// value copy (*buf) would share the backing array with the consumer, and a
	// later producer would overwrite the bytes while writeLog is still writing them.
	logChannel <- buf
}

func writeLog() {
	logFilePath := "/tmp/app.log"
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for logEntry := range logChannel {
		_, err := f.Write(logEntry.Bytes())
		if err != nil {
			fmt.Println("Error writing log:", err)
		}

		// The consumer owns the buffer now that it has been written out.
		// Reset it (keeping the allocated capacity) and return it to the pool.
		logEntry.Reset()
		logBufferPool.Put(logEntry)
	}
}

func _main() {
	N := 100
	logChannel = make(chan *bytes.Buffer, N)

	var logWg sync.WaitGroup
	logWg.Go(func() {
		writeLog()
	})

	var wg sync.WaitGroup
	for i := 1; i <= N; i++ {
		wg.Go(func() {
			dummyProcess(i)
		})
	}
	// Block until all the goroutines started by wg are done. A goroutine is done when the function it invokes returns.

	wg.Wait()
	close(logChannel)
	logWg.Wait()
}

func main() {
	_main()
}
