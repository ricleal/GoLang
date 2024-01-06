package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// one main routine to write / read from the queue
// Multiple routines to process the queue (e.g. fetch a URL, send an email, etc)

type Status string

const (
	// StatusPending is the status of a task that has not been processed.
	StatusPending Status = "pending"
	// StatusInProgress is the status of a task that is currently being processed.
	StatusInProgress Status = "in_progress"
	// StatusCompleted is the status of a task that has been processed successfully.
	StatusCompleted Status = "completed"
	// StatusFailed is the status of a task that has failed to be processed.
	StatusFailed Status = "failed"
)

func init() {
	// fill the queue with some records
	for i := 0; i < 100; i++ {
		queue = append(queue, &Record{
			ID:     i,
			Status: StatusPending,
			Email:  fmt.Sprintf("jonh.doe%d@mail.com", i),
		})
	}
}

type Record struct {
	ID     int
	Status Status
	Email  string
}

type Result struct {
	ID      int
	Success bool
}

type Queue []*Record

var (
	queue       Queue
	toProcessCh = make(chan Record, 10)
	processedCh = make(chan Result, 10)
)

// Get returns a slice of nRecords in StatusPending status and updates their status to StatusInProgress
func (q Queue) Get(nRecords int) []Record {
	var records []Record
	for i := range q {
		r := q[i]
		if r.Status == StatusPending {
			r.Status = StatusInProgress
			records = append(records, *r)
			if len(records) == nRecords {
				break
			}
		}
	}
	return records
}

func (q Queue) UpdateSuccess(id int) {
	q[id].Status = StatusCompleted
}

func (q Queue) UpdateFailed(id int) {
	q[id].Status = StatusFailed
}

// Email and return success or not
func process(r *Record) *Result {
	// pretend to send an email
	time.Sleep(10 * time.Millisecond)
	if r.ID == 999 {
		fmt.Println("Record 999 is taking a long time to process...")
		time.Sleep(1 * time.Second)
	}
	// 1 in 10 chance of failing
	return &Result{
		ID:      r.ID,
		Success: r.ID%10 != 0,
	}
}

func processInParallel(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for r := range toProcessCh {
		fmt.Printf("<- %d\n", r.ID)
		processedCh <- *process(&r)
	}
}

func main() {

	ctx, cancel := context.WithCancel(context.Background()) // Create a cancelable context
	defer cancel()                                          // Ensure cancellation at the end

	var wg sync.WaitGroup
	// launch 10 workers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go processInParallel(ctx, &wg) // Pass the context to workers
	}

	var totalToProcess int
	var totalProcessed int

outerLoop:
	for {
		select {
		case result := <-processedCh:
			if result.Success {
				queue.UpdateSuccess(result.ID)
			} else {
				queue.UpdateFailed(result.ID)
			}
			totalProcessed++
		default:
			// get 10 records to process
			records := queue.Get(10)
			if len(records) > 0 {
				totalToProcess += len(records)
				for _, r := range records {
					fmt.Printf("  -> %d\n", r.ID)
					toProcessCh <- r
				}
			} else if totalProcessed == totalToProcess {
				fmt.Printf("All records processed: %d = %d\n", totalProcessed, totalToProcess)
				break outerLoop
			}

		}
	}
	fmt.Println("Waiting for workers to finish...")
	close(toProcessCh)
	close(processedCh)
	wg.Wait()
	fmt.Println("Done")
}
