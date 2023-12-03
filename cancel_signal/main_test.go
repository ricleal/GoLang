package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"
)

func Test_Queue(t *testing.T) {
	t.Parallel()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Let's start the consumer
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		Consumer(ctx)
		wg.Done()
	}()

	// let's make sure the consumer is running and handling properly any signal
	time.Sleep(500 * time.Millisecond)

	// emit a signal to stop the consumer
	err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	if err != nil {
		t.Errorf("Error sending signal: %v", err)
	}

	// Wait for the consumer to stop
	<-ctx.Done()

	// Wait for the consumer to finish
	wg.Wait()

}
