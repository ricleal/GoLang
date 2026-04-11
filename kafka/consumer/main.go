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

// topicFlag allows -topic to be repeated: -topic claims_auto -topic claims_home
type topicFlag []string

func (t *topicFlag) String() string     { return strings.Join(*t, ",") }
func (t *topicFlag) Set(v string) error { *t = append(*t, v); return nil }

func main() {
	var topics topicFlag
	flag.Var(&topics, "topic", "Kafka topic to consume (repeatable; omit for all topics)")
	broker := flag.String("broker", claims.BrokerAddr, "Kafka broker address")
	groupID := flag.String("group", "", "Consumer group ID (default: derived from topics)")
	flag.Parse()

	if len(topics) == 0 {
		topics = claims.Topics
	}
	if *groupID == "" {
		*groupID = "group-" + strings.Join(topics, "+")
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}

	group, err := sarama.NewConsumerGroup([]string{*broker}, *groupID, cfg)
	if err != nil {
		slog.Error("create consumer group", "error", err)
		os.Exit(1)
	}
	defer group.Close()

	slog.Info("consumer started", "broker", *broker, "group", *groupID, "topics", []string(topics))

	stats := newStats()

	// Print stats every 5 s.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Printf("── stats (group=%s)  %s\n", *groupID, stats.tick())
			}
		}
	}()

	handler := &handler{stats: stats, group: *groupID}
	for {
		if err := group.Consume(ctx, []string(topics), handler); err != nil {
			if ctx.Err() != nil {
				break
			}
			slog.Error("consume error", "error", err)
		}
		if ctx.Err() != nil {
			break
		}
	}

	fmt.Printf("\n── final stats (group=%s)  %s\n", *groupID, stats.summary())
	slog.Info("consumer stopped")
}

// ── handler ──────────────────────────────────────────────────────────────────

type handler struct {
	stats *stats
	group string
}

func (h *handler) Setup(sarama.ConsumerGroupSession) error {
	slog.Info("rebalanced", "group", h.group)
	return nil
}
func (h *handler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			var c claims.Claim
			if err := json.Unmarshal(msg.Value, &c); err != nil {
				slog.Error("unmarshal", "error", err)
				session.MarkMessage(msg, "")
				continue
			}
			h.stats.record(c)
			session.MarkMessage(msg, "")
			slog.Debug("consumed",
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

type stats struct {
	mu          sync.Mutex
	count       int64
	lastCount   int64 // count at the previous tick
	totalAmount float64
	byType      map[string]int64
	startedAt   time.Time
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

// tick is called once per stats interval; it reports idle when nothing arrived.
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
