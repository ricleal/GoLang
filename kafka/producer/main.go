package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"

	"exp/kafka/claims"
)

const (
	minAmount    = 100.0  // minimum claim amount in USD
	amountRange  = 9900.0 // range added to minAmount: results in [100, 10000]
	maxCustomers = 1000   // pool of simulated customers (customer-0001 … customer-1000)
)

// claimTypeList drives random claim-type selection in randomClaim.
var claimTypeList = []string{"auto", "home", "life"} //nolint:gochecknoglobals // package-level constant list

func run() int {
	broker := flag.String("broker", claims.BrokerAddr, "Kafka broker address")
	workers := flag.Int("workers", claims.NumProducerWorkers, "Number of producer goroutines")
	poisonRate := flag.Float64("poison-rate", 0.0, "Fraction of messages sent with malformed JSON (0.0–1.0)")
	var logLevel slog.Level
	flag.TextVar(&logLevel, "log-level", slog.LevelInfo, "log level (DEBUG, INFO, WARN, ERROR)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	// Cancel the context on SIGINT/SIGTERM so all worker goroutines stop cleanly.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true
	// Hash partitioner: same CustomerID → same partition, avoiding hot spots.
	cfg.Producer.Partitioner = sarama.NewHashPartitioner

	sp, err := sarama.NewSyncProducer([]string{*broker}, cfg)
	if err != nil {
		logger.Error("create producer", "error", err)
		return 1
	}
	defer sp.Close()

	logger.Info("producer started", "broker", *broker, "workers", *workers, "topics", claims.Topics)

	// wg.Go launches each worker and wg.Wait blocks until all return.
	var wg sync.WaitGroup
	for i := range *workers {
		wg.Go(func() {
			work(ctx, logger, sp, i, *poisonRate)
		})
	}
	wg.Wait()
	logger.Info("producer stopped")
	return 0
}

func main() {
	os.Exit(run())
}

// work runs a tight produce loop until ctx is cancelled. It generates a random
// claim, marshals it to JSON, and sends it to the appropriate topic.
// When poisonRate > 0, a random fraction of messages are sent as malformed JSON.
func work(ctx context.Context, logger *slog.Logger, sp sarama.SyncProducer, workerID int, poisonRate float64) {
	for {
		// Non-blocking check so the worker exits promptly on shutdown.
		select {
		case <-ctx.Done():
			logger.Info("worker stopping", "worker", workerID)
			return
		default:
		}

		c := randomClaim()
		topic := claims.TopicFor(c.ClaimType)

		var payload []byte
		var err error
		if poisonRate > 0 && rand.Float64() < poisonRate { //nolint:gosec // math/rand/v2 is fine for chaos
			payload = []byte(`{"id":"` + c.ID + `",BROKEN_JSON`)
			logger.Info("injecting poison pill", "worker", workerID, "topic", topic, "id", c.ID)
		} else {
			payload, err = json.Marshal(c)
			if err != nil {
				logger.Error("marshal", "error", err, "worker", workerID)
				continue
			}
		}

		msg := &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(c.CustomerID),
			Value: sarama.ByteEncoder(payload),
		}

		partition, offset, sendErr := sp.SendMessage(msg)
		if sendErr != nil {
			logger.Error("send", "error", sendErr, "worker", workerID)
			continue
		}

		logger.Debug("produced",
			"worker", workerID,
			"topic", topic,
			"partition", partition,
			"offset", offset,
			"id", c.ID,
			"customer", c.CustomerID,
			"type", c.ClaimType,
			"amount", fmt.Sprintf("%.2f", c.Amount),
		)
	}
}

// randomClaim generates a claim with uniformly distributed type, customer, and amount.
func randomClaim() claims.Claim {
	ct := claimTypeList[rand.IntN(len(claimTypeList))] //nolint:gosec // simulation data, crypto randomness not needed
	return claims.Claim{
		ID:         uuid.NewString(),
		CustomerID: fmt.Sprintf("customer-%04d", rand.IntN(maxCustomers)+1), //nolint:gosec // simulation data
		ClaimType:  ct,
		Amount:     minAmount + rand.Float64()*amountRange, //nolint:gosec // simulation data
		Timestamp:  time.Now().UTC(),
	}
}
