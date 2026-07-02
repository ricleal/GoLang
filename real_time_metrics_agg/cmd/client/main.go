package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ricleal/GoLang/real_time_metrics_agg/internal/transport"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <server-addr> <client-id>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  server-addr: host:port of the gRPC aggregator server\n")
		fmt.Fprintf(os.Stderr, "  client-id:   unique identifier for this client\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s localhost:50051 client-1\n", os.Args[0])
		os.Exit(1)
	}

	serverAddr := os.Args[1]
	clientID := os.Args[2]

	logger := log.New(os.Stdout, fmt.Sprintf("[%s] ", clientID), log.LstdFlags|log.Lmicroseconds)
	logger.Printf("starting metrics client, connecting to %s", serverAddr)

	// Create the metrics client
	client, err := transport.NewMetricsClient(serverAddr, clientID, logger)
	if err != nil {
		logger.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Set up some default thresholds on the server
	if err := client.SetThreshold("cpu_usage", 50.0, 80.0); err != nil {
		logger.Printf("[WARN] set threshold cpu_usage: %v", err)
	}
	if err := client.SetThreshold("memory_usage", 70.0, 90.0); err != nil {
		logger.Printf("[WARN] set threshold memory_usage: %v", err)
	}
	if err := client.SetThreshold("error_rate", 5.0, 10.0); err != nil {
		logger.Printf("[WARN] set threshold error_rate: %v", err)
	}
	logger.Printf("[INFO] alert thresholds configured on server")

	// Context with signal-based cancellation for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Check server health first
	resp, err := client.HealthCheck()
	if err != nil {
		logger.Printf("[WARN] initial health check failed: %v (will continue)", err)
	} else {
		logger.Printf("[INFO] server health: status=%s leader=%v", resp.Status, resp.Leader)
	}

	// Simulate streaming metrics
	logger.Printf("[INFO] streaming metrics to %s...", serverAddr)
	if err := transport.SimulateClient(ctx, client, logger); err != nil && err != context.Canceled {
		logger.Fatalf("simulation error: %v", err)
	}

	logger.Printf("[INFO] client stopped")
}
