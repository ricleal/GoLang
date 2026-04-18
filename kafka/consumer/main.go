package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	duckdb "github.com/marcboeker/go-duckdb"

	"exp/kafka/claims"
)

// statInterval controls how often the consumer prints a stats line to stdout.
const statInterval = 5 * time.Second

const (
	defaultFlushInterval = 10 * time.Second
	defaultBatchSize     = 1000
	defaultDBPath        = "/tmp"
)

// topicFlag is a repeatable string flag: -topic claims-auto -topic claims-home.
// It satisfies the [flag.Value] interface.
type topicFlag []string

func (t *topicFlag) String() string     { return strings.Join(*t, ",") }
func (t *topicFlag) Set(v string) error { *t = append(*t, v); return nil }

func run() int {
	var topics topicFlag
	var logLevel slog.Level
	flag.Var(&topics, "topic", "Kafka topic to consume (repeatable; omit for all topics)")
	broker := flag.String("broker", claims.BrokerAddr, "Kafka broker address")
	groupID := flag.String("group", "", "Consumer group ID (default: derived from topics)")
	dbPath := flag.String("duckdb-path", defaultDBPath, "DuckDB file path or directory")
	flushInterval := flag.Duration("flush-interval", defaultFlushInterval, "How often to flush buffered events to DuckDB")
	batchSize := flag.Int("batch-size", defaultBatchSize, "Flush to DuckDB once this number of events is buffered")
	flag.TextVar(&logLevel, "log-level", slog.LevelInfo, "log level (DEBUG, INFO, WARN, ERROR)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	if *flushInterval <= 0 {
		logger.Error("invalid flush interval", "flush_interval", *flushInterval)
		return 1
	}
	if *batchSize <= 0 {
		logger.Error("invalid batch size", "batch_size", *batchSize)
		return 1
	}

	// Default to all known topics when none are specified.
	if len(topics) == 0 {
		topics = claims.Topics
	}
	// Derive a stable group ID from the topic list so each unique subscription
	// gets its own group, enabling fan-out by default.
	if *groupID == "" {
		*groupID = "group-" + strings.Join(topics, "+")
	}
	resolvedDBPath := resolveDuckDBPath(*dbPath, *groupID)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := sarama.NewConfig()
	// Start from the newest offset so new consumers don't replay old messages.
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}

	db, err := sql.Open("duckdb", resolvedDBPath)
	if err != nil {
		logger.Error("open duckdb", "error", err, "path", resolvedDBPath)
		return 1
	}
	defer db.Close()
	if err := ensureSchema(ctx, db); err != nil {
		logger.Error("ensure duckdb schema", "error", err)
		return 1
	}

	group, err := sarama.NewConsumerGroup([]string{*broker}, *groupID, cfg)
	if err != nil {
		logger.Error("create consumer group", "error", err)
		return 1
	}
	defer group.Close()

	logger.Info("consumer started",
		"broker", *broker,
		"group", *groupID,
		"topics", []string(topics),
		"duckdb_path", resolvedDBPath,
		"flush_interval", *flushInterval,
		"batch_size", *batchSize,
	)

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
				logger.Info("stats", "group", *groupID, "snapshot", st.tick())
			}
		}
	}()

	h := &handler{
		stats:         st,
		group:         *groupID,
		logger:        logger,
		db:            db,
		flushInterval: *flushInterval,
		batchSize:     *batchSize,
	}
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

	logger.Info("final stats", "group", *groupID, "snapshot", st.summary())
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
	stats         *stats
	group         string
	logger        *slog.Logger
	db            *sql.DB
	flushInterval time.Duration
	batchSize     int
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
	batcher := newEventBatcher(h.db, h.group, h.batchSize)
	ticker := time.NewTicker(h.flushInterval)
	defer ticker.Stop()
	flush := func(reason string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := batcher.flush(ctx, session); err != nil {
			h.logger.Error("flush to duckdb", "error", err, "reason", reason, "pending", batcher.pending())
		}
	}

	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				flush("messages-closed")
				return nil
			}
			var c claims.Claim
			if err := json.Unmarshal(msg.Value, &c); err != nil {
				h.logger.Error("unmarshal", "error", err)
				session.MarkMessage(msg, "")
				continue
			}
			h.stats.record(c)
			batcher.add(consumedEvent{msg: msg, claim: c})
			if batcher.pending() >= h.batchSize {
				flush("batch-size")
			}
			h.logger.Debug("consumed",
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"id", c.ID,
				"type", c.ClaimType,
				"amount", fmt.Sprintf("%.2f", c.Amount),
			)
		case <-ticker.C:
			flush("timer")
		case <-session.Context().Done():
			flush("session-done")
			return nil
		}
	}
}

type consumedEvent struct {
	msg   *sarama.ConsumerMessage
	claim claims.Claim
}

type eventBatcher struct {
	db     *sql.DB
	group  string
	limit  int
	events []consumedEvent
}

func newEventBatcher(db *sql.DB, group string, limit int) *eventBatcher {
	return &eventBatcher{db: db, group: group, limit: limit, events: make([]consumedEvent, 0, limit)}
}

func (b *eventBatcher) add(ev consumedEvent) {
	b.events = append(b.events, ev)
}

func (b *eventBatcher) pending() int {
	return len(b.events)
}

func (b *eventBatcher) flush(ctx context.Context, session sarama.ConsumerGroupSession) error {
	if len(b.events) == 0 {
		return nil
	}

	conn, err := b.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	now := time.Now().UTC()
	err = conn.Raw(func(driverConn any) error {
		rawConn, ok := driverConn.(driver.Conn)
		if !ok {
			return fmt.Errorf("unexpected duckdb driver connection type: %T", driverConn)
		}

		// go-duckdb Appender wraps DuckDB's native C appender API for efficient bulk inserts.
		appender, err := duckdb.NewAppenderFromConn(rawConn, "", "claims_events")
		if err != nil {
			return err
		}

		for _, ev := range b.events {
			err = appender.AppendRow(
				ev.msg.Topic,
				ev.msg.Partition,
				ev.msg.Offset,
				b.group,
				ev.claim.ID,
				ev.claim.CustomerID,
				ev.claim.ClaimType,
				ev.claim.Amount,
				ev.claim.Timestamp,
				now,
			)
			if err != nil {
				_ = appender.Close()
				return err
			}
		}

		return appender.Close()
	})
	if err != nil {
		return err
	}

	for _, ev := range b.events {
		session.MarkMessage(ev.msg, "persisted-to-duckdb")
	}
	b.events = b.events[:0]

	return nil
}

func ensureSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS claims_events (
			topic TEXT NOT NULL,
			partition INTEGER NOT NULL,
			message_offset BIGINT NOT NULL,
			consumer_group TEXT NOT NULL,
			claim_id TEXT NOT NULL,
			customer_id TEXT NOT NULL,
			claim_type TEXT NOT NULL,
			amount DOUBLE NOT NULL,
			event_time TIMESTAMP NOT NULL,
			consumed_at TIMESTAMP NOT NULL
		)
	`)
	return err
}

func resolveDuckDBPath(pathOrDir, group string) string {
	clean := filepath.Clean(pathOrDir)
	if strings.HasSuffix(strings.ToLower(clean), ".duckdb") {
		return clean
	}
	return filepath.Join(clean, sanitizeFilename(group)+".duckdb")
}

func sanitizeFilename(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "claims"
	}
	return b.String()
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
