// Package main implements the DLQ reader/replayer for the claims dead-letter
// topic. It prints every message that landed in claims-dlq (with its failure
// metadata) and, when -replay is set, re-publishes each message back to its
// original topic so a healthy consumer can process it again.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/IBM/sarama"

	"exp/kafka/claims"
)

const (
	tabMinWidth    = 2
	dlqReadTimeout = 5 * time.Second
)

func run() int {
	broker := flag.String("broker", claims.BrokerAddr, "Kafka broker address")
	replay := flag.Bool("replay", false, "Re-publish each DLQ message back to its original topic")
	var logLevel slog.Level
	flag.TextVar(&logLevel, "log-level", slog.LevelInfo, "log level (DEBUG, INFO, WARN, ERROR)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_6_0_0
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest // read from the very beginning

	consumer, err := sarama.NewConsumer([]string{*broker}, cfg)
	if err != nil {
		logger.Error("create consumer", "error", err)
		return 1
	}
	defer consumer.Close()

	pc, err := consumer.ConsumePartition(claims.TopicDLQ, 0, sarama.OffsetOldest)
	if err != nil {
		logger.Error("consume partition", "error", err, "topic", claims.TopicDLQ)
		return 1
	}
	defer pc.Close()

	newest, err := getNewestOffset(*broker, cfg, logger)
	if err != nil {
		return 1
	}
	if newest == 0 {
		fmt.Fprintln(os.Stdout, "claims-dlq is empty — no dead letters.")
		return 0
	}

	var replayProducer sarama.SyncProducer
	if *replay {
		replayProducer, err = newReplayProducer(*broker, logger)
		if err != nil {
			return 1
		}
		defer replayProducer.Close()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, tabMinWidth, ' ', 0)
	fmt.Fprintf(w, "\n── Dead Letters in %s (%d messages) ──\n\n", claims.TopicDLQ, newest)
	fmt.Fprintln(w, "DLQ_OFFSET\tORIG_TOPIC\tORIG_PART\tORIG_OFFSET\tCONSUMER_GROUP\tFAILED_AT\tERROR")

	read, replayed := consumeDLQ(pc, w, replayProducer, logger, newest, *replay, quit)

	_ = w.Flush()
	fmt.Fprintf(os.Stdout, "\nTotal: %d dead letter(s) read", read)
	if *replay {
		fmt.Fprintf(os.Stdout, ", %d replayed", replayed)
	}
	fmt.Fprintln(os.Stdout)
	return 0
}

// getNewestOffset creates a short-lived client to fetch the current high-water
// mark for the DLQ partition, then closes the client.
func getNewestOffset(broker string, cfg *sarama.Config, logger *slog.Logger) (int64, error) {
	client, err := sarama.NewClient([]string{broker}, cfg)
	if err != nil {
		logger.Error("create client", "error", err)
		return 0, err
	}
	newest, err := client.GetOffset(claims.TopicDLQ, 0, sarama.OffsetNewest)
	if closeErr := client.Close(); closeErr != nil {
		logger.Warn("close client", "error", closeErr)
	}
	if err != nil {
		logger.Error("get newest offset", "error", err)
		return 0, err
	}
	return newest, nil
}

// newReplayProducer creates a sync producer that hash-partitions messages back
// to their original topics.
func newReplayProducer(broker string, logger *slog.Logger) (sarama.SyncProducer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true
	cfg.Producer.Partitioner = sarama.NewHashPartitioner
	p, err := sarama.NewSyncProducer([]string{broker}, cfg)
	if err != nil {
		logger.Error("create replay producer", "error", err)
		return nil, err
	}
	logger.Info("replay mode active — messages will be re-published to their original topics")
	return p, nil
}

// consumeDLQ reads dead-letter messages from pc until caught up (msg.Offset+1 >= newest),
// a quit signal is received, or the read timeout elapses.
// Returns the number of messages read and replayed.
func consumeDLQ(
	pc sarama.PartitionConsumer,
	w *tabwriter.Writer,
	replayProducer sarama.SyncProducer,
	logger *slog.Logger,
	newest int64,
	doReplay bool,
	quit <-chan os.Signal,
) (int, int) {
	read, replayed := 0, 0
	for {
		select {
		case <-quit:
			return read, replayed

		case msg, ok := <-pc.Messages():
			if !ok {
				return read, replayed
			}
			replayed += processDeadLetter(w, replayProducer, logger, msg, doReplay)
			read++
			if msg.Offset+1 >= newest {
				return read, replayed
			}

		case pcErr := <-pc.Errors():
			logger.Error("partition consumer error", "error", pcErr)

		case <-time.After(dlqReadTimeout):
			// No new messages within timeout — DLQ is caught up.
			return read, replayed
		}
	}
}

// processDeadLetter prints one DLQ message to the table writer and optionally
// replays it to its original topic. Returns 1 if replayed, 0 otherwise.
func processDeadLetter(
	w *tabwriter.Writer,
	replayProducer sarama.SyncProducer,
	logger *slog.Logger,
	msg *sarama.ConsumerMessage,
	doReplay bool,
) int {
	headers := headerMap(msg.Headers)

	origTopic := headers["dlq-original-topic"]
	origPart := headers["dlq-original-partition"]
	origOffset := headers["dlq-original-offset"]
	consumerGroup := headers["dlq-consumer-group"]
	failedAt := headers["dlq-failed-at"]
	dlqErr := headers["dlq-error"]

	fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
		msg.Offset, origTopic, origPart, origOffset, consumerGroup, failedAt, dlqErr)

	if doReplay && origTopic != "" && replayMessage(replayProducer, logger, msg, origTopic) {
		return 1
	}
	return 0
}

// replayMessage re-publishes msg to origTopic. Returns true on success.
func replayMessage(
	producer sarama.SyncProducer,
	logger *slog.Logger,
	msg *sarama.ConsumerMessage,
	origTopic string,
) bool {
	replayMsg := &sarama.ProducerMessage{
		Topic: origTopic,
		Key:   sarama.ByteEncoder(msg.Key),
		Value: sarama.ByteEncoder(msg.Value),
	}
	if _, _, err := producer.SendMessage(replayMsg); err != nil {
		logger.Error("replay failed", "error", err, "original_topic", origTopic)
		return false
	}
	logger.Info("replayed", "dlq_offset", msg.Offset, "original_topic", origTopic)
	return true
}

func main() {
	os.Exit(run())
}

// headerMap converts a slice of sarama.RecordHeader into a string→string map.
func headerMap(headers []*sarama.RecordHeader) map[string]string {
	m := make(map[string]string, len(headers))
	for _, h := range headers {
		if h != nil {
			m[string(h.Key)] = string(h.Value)
		}
	}
	return m
}
