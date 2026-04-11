package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"

	"exp/kafka/claims"
)

// statInterval controls how often the consumer prints a stats line to stdout.
const statInterval = 5 * time.Second

// topicFlag is a repeatable string flag: -topic claims-auto -topic claims-home.
// It satisfies the flag.Value interface.
type topicFlag []string

func (t *topicFlag) String() string     { return strings.Join(*t, ",") }
func (t *topicFlag) Set(v string) error { *t = append(*t, v); return nil }

func run() int {
	var topics topicFlag
	flag.Var(&topics, "topic", "Kafka topic to consume (repeatable; omit for all topics)")
	broker := flag.String("broker", claims.BrokerAddr, "Kafka broker address")
	groupID := flag.String("group", "", "Consumer group ID (default: derived from topics)")
	flag.Parse()

	// Default to all known topics when none are specified.
	if len(topics) == 0 {
		topics = claims.Topics
	}
	// Derive a stable group ID from the topic list so each unique subscription
	// gets its own group, enabling fan-out by default.
	if *groupID == "" {
		*groupID = "group-" + strings.Join(topics, "+")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := sarama.NewConfig()
	// Start from the newest offset so new consumers don't replay old messages.
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}

	group, err := sarama.NewConsumerGroup([]string{*broker}, *groupID, cfg)
	if err != nil {
		logger.Error("create consumer group", "error", err)
		return 1
	}
	defer group.Close()

	logger.Info("consumer started", "broker", *broker, "group", *groupID, "topics", []string(topics))

	st := newStats()

	// Background goroutine periodically prints a stats snapshot.
	go func() {
		ticker := time.NewTicker(statInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Fprintf(os.Stdout, "── stats (group=%s)  %s\n", *groupID, st.tick())
			}
		}
	}()

	h := &handler{stats: st, group: *groupID, logger: logger}
	for {
		if consumeErr := group.Consume(ctx, []string(topics), h); consumeErr != nil {
			if ctx.Err() != nil {
				break
			}
			logger.Error("consume error", "error", consumeErr)
		}
		if ctx.Err() != nil {
			break
		}
	}

	fmt.Fprintf(os.Stdout, "\n── final stats (group=%s)  %s\n", *groupID, st.summary())
	logger.Info("consumer stopped")
	return 0
}

func main() {
	os.Exit(run())
}

// ── handler ──────────────────────────────────────────────────────────────────

// handler implements sarama.ConsumerGroupHandler. One instance is shared across
// all topic/partition goroutines spawned by the consumer group.
type handler struct {
	stats  *stats
	group  string
	logger *slog.Logger
}

// Setup is called at the start of each rebalance cycle.
func (h *handler) Setup(sarama.ConsumerGroupSession) error {
	h.logger.Info("rebalanced", "group", h.group)
	return nil
}

// Cleanup is called at the end of each rebalance cycle.
func (h *handler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

// ConsumeClaim processes one partition claim until the session ends or the
// messages channel is closed by the library.
func (h *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			var c claims.Claim
			if err := json.Unmarshal(msg.Value, &c); err != nil {
				h.logger.Error("unmarshal", "error", err)
				session.MarkMessage(msg, "")
				continue
			}
			h.stats.record(c)
			session.MarkMessage(msg, "")
			h.logger.Debug("consumed",
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"id", c.ID,
				"type", c.ClaimType,
				"amount", fmt.Sprintf("%.2f", c.Amount),
			)
		case <-session.Context().Done():
			return nil
		}
	}
}

// ── stats ─────────────────────────────────────────────────────────────────────

// stats accumulates counters for all messages processed by this consumer instance.
type stats struct {
	mu          sync.Mutex
	count       int64
	lastCount   int64 // snapshot at the previous tick, used to compute delta
	totalAmount float64
	byType      map[string]int64
	startedAt   time.Time // set on the first message, used for throughput rate
}

func newStats() *stats {
	return &stats{byType: make(map[string]int64)}
}

func (s *stats) record(c claims.Claim) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count == 0 {
		s.startedAt = time.Now()
	}
	s.count++
	s.totalAmount += c.Amount
	s.byType[c.ClaimType]++
}

// tick is called once per statInterval. It computes the delta since the last call
// and returns a human-readable status string, reporting idle when no new messages arrived.
func (s *stats) tick() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	delta := s.count - s.lastCount
	s.lastCount = s.count
	if delta == 0 {
		if s.count == 0 {
			return "idle — no messages received yet"
		}
		return fmt.Sprintf("idle (caught up)  lifetime_total=%d  auto=%d home=%d life=%d",
			s.count, s.byType["auto"], s.byType["home"], s.byType["life"])
	}
	elapsed := time.Since(s.startedAt).Seconds()
	return fmt.Sprintf("total=%d (+%d)  avg_amount=%.2f  rate=%.1f/s  auto=%d home=%d life=%d",
		s.count, delta, s.totalAmount/float64(s.count), float64(s.count)/elapsed,
		s.byType["auto"], s.byType["home"], s.byType["life"])
}

func (s *stats) summary() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count == 0 {
		return "no messages received"
	}
	elapsed := time.Since(s.startedAt).Seconds()
	return fmt.Sprintf("total=%d  avg_amount=%.2f  rate=%.1f/s  auto=%d home=%d life=%d",
		s.count,
		s.totalAmount/float64(s.count),
		float64(s.count)/elapsed,
		s.byType["auto"], s.byType["home"], s.byType["life"],
	)
}
