package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/storage/managedwriter"
	"golang.org/x/sync/semaphore"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

const (
	defaultProjectID = "ricardo-sandbox07-03-24"
	defaultDatasetID = "ricardo_dataset"
	defaultTableID   = "key_value_go"
)

type Config struct {
	ProjectID      string
	DatasetID      string
	TableID        string
	TotalRows      int
	BatchSize      int
	MaxConcurrency int
	WriteAPI       string // classic | storage
	Location       string
}

type bqRow struct { // classic streaming row
	Key   int64  `bigquery:"key"`
	Value string `bigquery:"value"`
}

func main() {
	cfg := parseFlags()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := bigquery.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		fatalf("create client: %v", err)
	}
	defer client.Close()

	if err := ensureDatasetExists(ctx, client, cfg.DatasetID, cfg.Location); err != nil {
		fatalf("ensure dataset: %v", err)
	}
	tableRef := client.Dataset(cfg.DatasetID).Table(cfg.TableID)
	if err := ensureTableExists(ctx, tableRef); err != nil {
		fatalf("ensure table: %v", err)
	}

	switch cfg.WriteAPI {
	case "classic":
		if err := runClassicStreaming(ctx, tableRef, cfg, cancel); err != nil {
			fatalf("classic write: %v", err)
		}
	case "storage":
		if err := runStorageWrite(ctx, tableRef, cfg); err != nil {
			fatalf("storage write: %v", err)
		}
	default:
		fatalf("unknown write_api: %s", cfg.WriteAPI)
	}

	// Final count verification
	query := client.Query(fmt.Sprintf("SELECT count(*) FROM `%s.%s.%s`", cfg.ProjectID, cfg.DatasetID, cfg.TableID))
	it, err := query.Read(ctx)
	if err != nil {
		fatalf("query count: %v", err)
	}
	var row []bigquery.Value
	if err := it.Next(&row); err == iterator.Done {
		fatalf("no rows returned")
	} else if err != nil {
		fatalf("next row: %v", err)
	}
	c, _ := strconv.Atoi(fmt.Sprintf("%v", row[0]))
	logf("Final row count: %d", c)
}

func parseFlags() Config {
	var cfg Config
	flag.StringVar(&cfg.ProjectID, "project", defaultProjectID, "GCP project ID")
	flag.StringVar(&cfg.DatasetID, "dataset", defaultDatasetID, "BigQuery dataset ID")
	flag.StringVar(&cfg.TableID, "table", defaultTableID, "BigQuery table ID")
	flag.IntVar(&cfg.TotalRows, "rows", 100000, "Total rows to generate")
	flag.IntVar(&cfg.BatchSize, "batch", 1000, "Rows per batch")
	flag.IntVar(&cfg.MaxConcurrency, "concurrency", 2, "Max concurrent batch inserts")
	flag.StringVar(&cfg.WriteAPI, "write_api", "classic", "Write API: classic | storage")
	flag.StringVar(&cfg.Location, "location", "US", "Dataset location (on create)")
	flag.Parse()
	return cfg
}

// Classic streaming path using Inserter().Put with concurrency
func runClassicStreaming(ctx context.Context, tableRef *bigquery.Table, cfg Config, cancel context.CancelFunc) error {
	inserter := tableRef.Inserter()
	sem := semaphore.NewWeighted(int64(cfg.MaxConcurrency))
	var (
		wg       sync.WaitGroup
		firstErr error
		errOnce  sync.Once
	)

	dispatch := func(batch []bqRow) {
		if len(batch) == 0 {
			return
		}
		startKey := batch[0].Key
		logf("[batch queue] key=%d size=%d", startKey, len(batch))
		if err := sem.Acquire(ctx, 1); err != nil {
			errOnce.Do(func() { firstErr = fmt.Errorf("acquire semaphore: %w", err) })
			return
		}
		local := make([]bqRow, len(batch))
		copy(local, batch)
		wg.Add(1)
		go func(rows []bqRow) {
			defer wg.Done()
			defer sem.Release(1)
			logf("[batch start] key=%d size=%d", startKey, len(rows))
			t0 := time.Now()
			if err := inserter.Put(ctx, rows); err != nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("insert batch starting %d: %w", startKey, err); cancel() })
				logf("[batch fail] key=%d size=%d err=%v", startKey, len(rows), err)
			} else {
				logf("[batch done] key=%d size=%d dur=%s", startKey, len(rows), time.Since(t0))
			}
		}(local)
	}

	batch := make([]bqRow, 0, cfg.BatchSize)
	for i := 0; i < cfg.TotalRows; i++ {
		if firstErr != nil {
			break
		}
		k := int64(i)
		batch = append(batch, bqRow{Key: k, Value: fmt.Sprintf(`{"user_id":"%d","name":"User %d","timestamp":"%s"}`, k, k, time.Now().Format(time.RFC3339))})
		if len(batch) == cfg.BatchSize {
			dispatch(batch)
			batch = make([]bqRow, 0, cfg.BatchSize)
		}
	}
	dispatch(batch)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	logf("Classic streaming complete")
	return nil
}

// Storage Write API path using JSON proto rows (no schema descriptor needed)
func runStorageWrite(ctx context.Context, tableRef *bigquery.Table, cfg Config) error {
	logf("Using Storage Write API (JSON proto rows)...")
	mwClient, err := managedwriter.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return fmt.Errorf("create managedwriter client: %w", err)
	}
	defer mwClient.Close()
	parent := managedwriter.TableParentFromParts(cfg.ProjectID, tableRef.DatasetID, tableRef.TableID)
	stream, err := mwClient.NewManagedStream(ctx, managedwriter.WithType(managedwriter.DefaultStream), managedwriter.WithDestinationTable(parent))
	if err != nil {
		return fmt.Errorf("create managed stream: %w", err)
	}

	for batch := 0; batch*cfg.BatchSize < cfg.TotalRows; batch++ {
		start := int64(batch * cfg.BatchSize)
		end := start + int64(cfg.BatchSize)
		if end > int64(cfg.TotalRows) {
			end = int64(cfg.TotalRows)
		}
		rows := make([][]byte, 0, end-start)
		for k := start; k < end; k++ {
			rows = append(rows, []byte(fmt.Sprintf(`{"key": %d, "value": {"user_id": "%d", "name": "User %d", "timestamp": "%s"}}`, k, k, k, time.Now().Format(time.RFC3339))))
		}
		logf("[storage append-start] batch=%d rows=%d", batch, len(rows))
		if _, err := stream.AppendRows(ctx, rows); err != nil {
			return fmt.Errorf("append rows batch %d: %w", batch, err)
		}
		logf("[storage append-done] batch=%d rows=%d", batch, len(rows))
	}
	logf("Storage Write API ingestion complete")
	return nil
}

// ---- Helpers ----
func ensureDatasetExists(ctx context.Context, client *bigquery.Client, datasetID, location string) error {
	ds := client.Dataset(datasetID)
	if _, err := ds.Metadata(ctx); err != nil {
		if !isNotFound(err) {
			return fmt.Errorf("get dataset metadata: %w", err)
		}
		if location == "" {
			location = "US"
		}
		if err := ds.Create(ctx, &bigquery.DatasetMetadata{Location: location}); err != nil {
			return fmt.Errorf("create dataset: %w", err)
		}
		logf("Created dataset %s", datasetID)
	}
	return nil
}

func ensureTableExists(ctx context.Context, tableRef *bigquery.Table) error {
	if _, err := tableRef.Metadata(ctx); err == nil {
		return nil
	} else if !isNotFound(err) {
		return fmt.Errorf("get table metadata: %w", err)
	}
	schema := bigquery.Schema{{Name: "key", Type: bigquery.IntegerFieldType, Required: true}, {Name: "value", Type: bigquery.JSONFieldType, Required: true}}
	if err := tableRef.Create(ctx, &bigquery.TableMetadata{Schema: schema, TimePartitioning: &bigquery.TimePartitioning{Type: bigquery.DayPartitioningType}}); err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	logf("Created table %s", tableRef.TableID)
	return nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if gerr, ok := err.(*bigquery.Error); ok {
		return gerr.Reason == "notFound"
	}
	if gapi, ok := err.(*googleapi.Error); ok {
		return gapi.Code == 404
	}
	return false
}

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
	return fmt.Sprintf("alloc=%.2fMB sys=%.2fMB rss=%.2fMB goroutines=%d", float64(ms.Alloc)/1e6, float64(ms.Sys)/1e6, float64(rssBytes())/1e6, runtime.NumGoroutine())
}

func logf(format string, args ...any) {
	log.Printf("%s | "+format, append([]any{memUsage()}, args...)...)
}

func fatalf(format string, args ...any) {
	log.Fatalf("%s | "+format, append([]any{memUsage()}, args...)...)
}
