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
	minAmount    = 100.0
	amountRange  = 9900.0
	maxCustomers = 1000
)

var claimTypeList = []string{"auto", "home", "life"} //nolint:gochecknoglobals // package-level constant list

func run() int {
	broker := flag.String("broker", claims.BrokerAddr, "Kafka broker address")
	workers := flag.Int("workers", claims.NumProducerWorkers, "Number of producer goroutines")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

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

	var wg sync.WaitGroup
	for i := range *workers {
		wg.Go(func() {
			work(ctx, logger, sp, i)
		})
	}
	wg.Wait()
	logger.Info("producer stopped")
	return 0
}

func main() {
	os.Exit(run())
}

func work(ctx context.Context, logger *slog.Logger, sp sarama.SyncProducer, workerID int) {
	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopping", "worker", workerID)
			return
		default:
		}

		c := randomClaim()
		topic := claims.TopicFor(c.ClaimType)

		payload, err := json.Marshal(c)
		if err != nil {
			logger.Error("marshal", "error", err, "worker", workerID)
			continue
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
