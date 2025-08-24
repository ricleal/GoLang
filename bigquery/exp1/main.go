// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/sync/semaphore"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

const (
	projectID            = "ricardo-sandbox07-03-24" // Replace with your GCP project ID
	datasetID            = "ricardo_dataset"         // Replace with your BigQuery dataset ID
	tableID              = "key_value_go"            // Table name to be created
	totalRows            = 10000                     // Total number of rows to insert
	batchSize            = 1000                      // Number of rows per batch
	maxConcurrentBatches = 2                         // Tune: limit concurrent BigQuery insert calls
)

// ensureTableExists checks if the table exists, and creates it if it doesn't.
func ensureTableExists(ctx context.Context, client *bigquery.Client, tableRef *bigquery.Table) error {
	logf("Checking for table %s...", tableRef.TableID)
	if _, err := tableRef.Metadata(ctx); err == nil {
		logf("Table %s already exists.", tableRef.TableID)
		return nil
	} else if !isNotFound(err) {
		return fmt.Errorf("failed to get table metadata: %w", err)
	}
	// Create table
	schema := bigquery.Schema{
		{Name: "key", Type: bigquery.IntegerFieldType, Required: true},
		{Name: "value", Type: bigquery.JSONFieldType, Required: true},
	}
	meta := &bigquery.TableMetadata{Schema: schema, TimePartitioning: &bigquery.TimePartitioning{Type: bigquery.DayPartitioningType}}
	if err := tableRef.Create(ctx, meta); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	logf("Created table %s with JSON column.", tableRef.TableID)
	return nil
}

// isNotFound returns true if the error represents a 404/notFound from BigQuery/Google API.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if gerr, ok := err.(*bigquery.Error); ok {
		return gerr.Reason == "notFound"
	}
	if gapiErr, ok := err.(*googleapi.Error); ok {
		return gapiErr.Code == 404
	}
	return false
}

// ensureDatasetExists creates the dataset if missing.
func ensureDatasetExists(ctx context.Context, client *bigquery.Client, datasetID string) error {
	ds := client.Dataset(datasetID)
	if _, err := ds.Metadata(ctx); err != nil {
		if !isNotFound(err) {
			return fmt.Errorf("failed to get dataset metadata: %w", err)
		}
		// Create dataset (assume US location; adjust if needed).
		if err := ds.Create(ctx, &bigquery.DatasetMetadata{Location: "US"}); err != nil {
			return fmt.Errorf("failed to create dataset: %w", err)
		}
		logf("Created dataset %s", datasetID)
	} else {
		logf("Dataset %s exists", datasetID)
	}
	return nil
}

// rssBytes returns current resident set size (Linux specific best-effort).
func rssBytes() uint64 {
	f, err := os.Open("/proc/self/statm")
	if err != nil {
		return 0
	}
	defer f.Close()
	var size, resident, share, text, lib, data, dt uint64
	fmt.Fscan(f, &size, &resident, &share, &text, &lib, &data, &dt)
	return resident * uint64(os.Getpagesize())
}

func memUsage() string {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	rss := rssBytes()
	return fmt.Sprintf("alloc=%.2fMB sys=%.2fMB rss=%.2fMB goroutines=%d", float64(ms.Alloc)/1e6, float64(ms.Sys)/1e6, float64(rss)/1e6, runtime.NumGoroutine())
}

func logf(format string, args ...any) {
	log.Printf("%s | "+format, append([]any{memUsage()}, args...)...)
}

func fatalf(format string, args ...any) {
	log.Fatalf("%s | "+format, append([]any{memUsage()}, args...)...)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize BigQuery client
	// Use default BigQuery endpoint; custom endpoint previously produced 404 paths without service prefix.
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		fatalf("Failed to create BigQuery client: %v", err)
	}
	defer client.Close()

	if err := ensureDatasetExists(ctx, client, datasetID); err != nil {
		fatalf("Failed to ensure dataset exists: %v", err)
	}
	tableRef := client.Dataset(datasetID).Table(tableID)
	if err := ensureTableExists(ctx, client, tableRef); err != nil {
		fatalf("Failed to ensure table exists: %v", err)
	}

	// Row model matching the table schema.
	type bqRow struct {
		Key   int64  `bigquery:"key"`
		Value string `bigquery:"value"`
	}

	inserter := tableRef.Inserter()

	// Sequential generation with concurrent flushing when batchSize reached.
	var (
		pending  = make([]*bqRow, 0, batchSize)
		wg       sync.WaitGroup
		sem      = semaphore.NewWeighted(int64(maxConcurrentBatches))
		errOnce  sync.Once
		firstErr error
	)

	// helper to dispatch a full (or final) batch asynchronously
	dispatch := func(batch []*bqRow) {
		if len(batch) == 0 {
			return
		}
		startKey := batch[0].Key
		logf("[batch start-queue] key=%d size=%d", startKey, len(batch))
		if err := sem.Acquire(ctx, 1); err != nil {
			errOnce.Do(func() {
				firstErr = fmt.Errorf("failed to acquire semaphore: %w", err)
			})
			return
		}
		wg.Add(1)
		go func(local []*bqRow) {
			defer wg.Done()
			defer sem.Release(1)
			logf("[batch write-start] key=%d size=%d", startKey, len(local))
			beg := time.Now()
			if err := inserter.Put(ctx, local); err != nil {
				errOnce.Do(func() {
					firstErr = fmt.Errorf("inserting batch starting at key %d: %w", local[0].Key, err)
					cancel() // cancel context to unblock any pending acquires
				})
				logf("[batch write-fail] key=%d size=%d err=%v", startKey, len(local), err)
			} else {
				logf("[batch write-done] key=%d size=%d duration=%s", startKey, len(local), time.Since(beg))
			}
		}(batch)
	}

	for key := int64(0); key < totalRows; key++ {
		// If an earlier batch failed, stop generating more rows.
		if firstErr != nil {
			break
		}
		pending = append(pending, &bqRow{
			Key: key,
			Value: fmt.Sprintf(`{"user_id":"%d","name":"User %d","timestamp":"%s"}`,
				key, key, time.Now().Format(time.RFC3339)),
		})
		if len(pending) == batchSize {
			// Hand off current full batch; start a new slice with fresh backing array (avoids data race / mutation issues).
			batch := pending
			pending = make([]*bqRow, 0, batchSize)
			dispatch(batch)
		}
	}
	// Dispatch any final partial batch.
	dispatch(pending)

	// Wait for all batches to finish.
	wg.Wait()
	if firstErr != nil {
		fatalf("One or more batches failed: %v", firstErr)
	}
	logf("All batches inserted successfully.")

	// Verify the count of the inserted rows
	logf("All batches loaded. Verifying row count...")
	query := client.Query(fmt.Sprintf("SELECT count(*) FROM `%s.%s.%s`", projectID, datasetID, tableID))
	it, err := query.Read(ctx)
	if err != nil {
		fatalf("Failed to read query result: %v", err)
	}

	var countRow []bigquery.Value
	if err := it.Next(&countRow); err == iterator.Done {
		fatalf("No rows returned from count query")
	} else if err != nil {
		fatalf("Failed to get count: %v", err)
	}
	count, _ := strconv.Atoi(fmt.Sprintf("%v", countRow[0]))
	logf("Total rows in table after loading: %d", count)
}
