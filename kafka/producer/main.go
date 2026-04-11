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

func main() {
	broker := flag.String("broker", claims.BrokerAddr, "Kafka broker address")
	workers := flag.Int("workers", claims.NumProducerWorkers, "Number of producer goroutines")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true
	// Hash partitioner: same CustomerID → same partition, avoiding hot spots.
	cfg.Producer.Partitioner = sarama.NewHashPartitioner

	sp, err := sarama.NewSyncProducer([]string{*broker}, cfg)
	if err != nil {
		slog.Error("create producer", "error", err)
		os.Exit(1)
	}
	defer sp.Close()

	slog.Info("producer started", "broker", *broker, "workers", *workers, "topics", claims.Topics)

	var wg sync.WaitGroup
	for i := range *workers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			work(ctx, sp, id)
		}(i)
	}
	wg.Wait()
	slog.Info("producer stopped")
}

var claimTypes = []string{"auto", "home", "life"}

func work(ctx context.Context, sp sarama.SyncProducer, workerID int) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopping", "worker", workerID)
			return
		default:
		}

		c := randomClaim()
		topic := claims.TopicFor(c.ClaimType)

		payload, err := json.Marshal(c)
		if err != nil {
			slog.Error("marshal", "error", err, "worker", workerID)
			continue
		}

		msg := &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(c.CustomerID),
			Value: sarama.ByteEncoder(payload),
		}

		partition, offset, err := sp.SendMessage(msg)
		if err != nil {
			slog.Error("send", "error", err, "worker", workerID)
			continue
		}

		slog.Debug("produced",
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
	ct := claimTypes[rand.IntN(len(claimTypes))]
	return claims.Claim{
		ID:         uuid.NewString(),
		CustomerID: fmt.Sprintf("customer-%04d", rand.IntN(1000)+1),
		ClaimType:  ct,
		Amount:     100 + rand.Float64()*9900,
		Timestamp:  time.Now().UTC(),
	}
}
