package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	_ "github.com/marcboeker/go-duckdb" // registers the "duckdb" sql driver

	"exp/kafka/claims"
)

// statInterval controls how often the consumer prints a stats line to stdout.
const statInterval = 5 * time.Second

const (
	defaultFlushInterval = 10 * time.Second
	defaultBatchSize     = 1000
	defaultDBPath        = "/tmp"
)

const (
	flushRetryTimeout  = 30 * time.Second
	retryBaseBackoffMS = 500
	maxFlushAttempts   = 3
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
	flushInterval := flag.Duration(
		"flush-interval", defaultFlushInterval, "How often to flush buffered events to DuckDB",
	)
	batchSize := flag.Int("batch-size", defaultBatchSize, "Flush to DuckDB once this number of events is buffered")
	chaosCrashAfter := flag.Int(
		"chaos-crash-after", 0, "Crash the consumer after processing this many messages (0 = disabled)",
	)
	chaosFailDBProb := flag.Float64(
		"chaos-fail-db-prob", 0.0, "Probability (0.0–1.0) that a flush attempt fails with a simulated DB error",
	)
	chaosSlowMS := flag.Int("chaos-slow-ms", 0, "Add this many ms of artificial delay per message (0 = disabled)")
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

	db, err := setupDB(ctx, logger, resolvedDBPath)
	if err != nil {
		return 1
	}
	defer db.Close()

	clients, err := newKafkaClients(logger, *broker, *groupID)
	if err != nil {
		return 1
	}
	defer clients.group.Close()
	defer clients.dlqProducer.Close()

	logger.Info("consumer started",
		"broker", *broker,
		"group", *groupID,
		"topics", []string(topics),
		"duckdb_path", resolvedDBPath,
		"flush_interval", *flushInterval,
		"batch_size", *batchSize,
		"chaos_crash_after", *chaosCrashAfter,
		"chaos_fail_db_prob", *chaosFailDBProb,
		"chaos_slow_ms", *chaosSlowMS,
	)

	st := newStats()

	runStatsPrinter(ctx, logger, *groupID, st)

	h := &handler{
		stats:           st,
		group:           *groupID,
		logger:          logger,
		db:              db,
		dlqProducer:     clients.dlqProducer,
		flushInterval:   *flushInterval,
		batchSize:       *batchSize,
		chaosCrashAfter: *chaosCrashAfter,
		chaosFailDBProb: *chaosFailDBProb,
		chaosSlowMS:     *chaosSlowMS,
	}
	for {
		if consumeErr := clients.group.Consume(ctx, []string(topics), h); consumeErr != nil {
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

// runStatsPrinter starts a goroutine that logs a stats snapshot on every statInterval
// tick until ctx is cancelled.
func runStatsPrinter(ctx context.Context, logger *slog.Logger, groupID string, st *stats) {
	go func() {
		ticker := time.NewTicker(statInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logger.Info("stats", "group", groupID, "snapshot", st.tick())
			}
		}
	}()
}

// setupDB opens the DuckDB database at path and ensures the schema exists.
func setupDB(ctx context.Context, logger *slog.Logger, path string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		logger.Error("open duckdb", "error", err, "path", path)
		return nil, err
	}
	if schemaErr := ensureSchema(ctx, db); schemaErr != nil {
		_ = db.Close()
		logger.Error("ensure duckdb schema", "error", schemaErr)
		return nil, schemaErr
	}
	return db, nil
}

// kafkaClients groups the consumer group and DLQ producer used by run.
type kafkaClients struct {
	group       sarama.ConsumerGroup
	dlqProducer sarama.SyncProducer
}

// newKafkaClients creates a consumer group and a DLQ sync producer.
func newKafkaClients(logger *slog.Logger, broker, groupID string) (*kafkaClients, error) {
	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	group, err := sarama.NewConsumerGroup([]string{broker}, groupID, cfg)
	if err != nil {
		logger.Error("create consumer group", "error", err)
		return nil, err
	}
	dlqCfg := sarama.NewConfig()
	dlqCfg.Producer.Return.Successes = true
	dlqCfg.Producer.Return.Errors = true
	dlqProducer, err := sarama.NewSyncProducer([]string{broker}, dlqCfg)
	if err != nil {
		if closeErr := group.Close(); closeErr != nil {
			logger.Warn("close consumer group after DLQ producer error", "error", closeErr)
		}
		logger.Error("create DLQ producer", "error", err)
		return nil, err
	}
	return &kafkaClients{group: group, dlqProducer: dlqProducer}, nil
}

func main() {
	os.Exit(run())
}

// ── handler ──────────────────────────────────────────────────────────────────

// handler implements sarama.ConsumerGroupHandler. One instance is shared across
// all topic/partition goroutines spawned by the consumer group.
type handler struct {
	stats           *stats
	group           string
	logger          *slog.Logger
	db              *sql.DB
	dlqProducer     sarama.SyncProducer
	flushInterval   time.Duration
	batchSize       int
	chaosCrashAfter int
	chaosFailDBProb float64
	chaosSlowMS     int
	msgCount        atomic.Int64
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
	batcher := newEventBatcher(h.db, h.logger, h.group, h.batchSize)
	ticker := time.NewTicker(h.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				h.retryFlush(batcher, session, "messages-closed")
				return nil
			}
			h.handleMessage(session, msg, batcher)
		case <-ticker.C:
			h.retryFlush(batcher, session, "timer")
		case <-session.Context().Done():
			h.retryFlush(batcher, session, "session-done")
			return nil
		}
	}
}

// handleMessage deserialises a single Kafka message, applies chaos hooks, records
// stats, and triggers a flush when the batch is full.
func (h *handler) handleMessage(
	session sarama.ConsumerGroupSession,
	msg *sarama.ConsumerMessage,
	batcher *eventBatcher,
) {
	var c claims.Claim
	if err := json.Unmarshal(msg.Value, &c); err != nil {
		h.logger.Error("unmarshal (poison pill)", "error", err,
			"topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)
		h.sendRawToDLQ(msg, "unmarshal-error: "+err.Error())
		session.MarkMessage(msg, "dlq-poison-pill")
		return
	}
	// Chaos: artificial slowdown to simulate a slow processing path.
	if h.chaosSlowMS > 0 {
		time.Sleep(time.Duration(h.chaosSlowMS) * time.Millisecond)
	}
	h.stats.record(c)
	batcher.add(consumedEvent{msg: msg, claim: c})
	if batcher.pending() >= h.batchSize {
		h.retryFlush(batcher, session, "batch-size")
	}
	// Chaos: hard crash after N total messages to simulate an OOM-kill.
	if h.chaosCrashAfter > 0 {
		if n := h.msgCount.Add(1); n >= int64(h.chaosCrashAfter) {
			h.logger.Error("chaos: crashing consumer",
				"msg_count", n, "crash_after", h.chaosCrashAfter)
			os.Exit(1)
		}
	}
	h.logger.Debug("consumed",
		"topic", msg.Topic,
		"partition", msg.Partition,
		"offset", msg.Offset,
		"id", c.ID,
		"type", c.ClaimType,
		"amount", fmt.Sprintf("%.2f", c.Amount),
	)
}

// retryFlush attempts the DuckDB flush up to maxFlushAttempts times with
// exponential backoff. On permanent failure it drains the pending batch to the DLQ.
func (h *handler) retryFlush(batcher *eventBatcher, session sarama.ConsumerGroupSession, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), flushRetryTimeout)
	defer cancel()
	var lastErr error
retryLoop:
	for attempt := range maxFlushAttempts {
		if attempt > 0 {
			backoff := time.Duration(retryBaseBackoffMS<<uint(attempt)) * time.Millisecond
			h.logger.Warn("flush retry", "attempt", attempt+1, "max", maxFlushAttempts, "backoff", backoff)
			select {
			case <-ctx.Done():
				lastErr = ctx.Err()
				break retryLoop
			case <-time.After(backoff):
			}
		}
		// Chaos: randomly return a fake DB error before attempting the real flush.
		if h.chaosFailDBProb > 0 && rand.Float64() < h.chaosFailDBProb { //nolint:gosec // intentional: math/rand/v2 is sufficient for chaos
			lastErr = errors.New("chaos: simulated database failure")
			h.logger.Error("chaos: injected DB failure", "attempt", attempt+1, "pending", batcher.pending())
			continue
		}
		lastErr = batcher.flush(ctx, session)
		if lastErr == nil {
			return
		}
		h.logger.Error("flush to duckdb", "error", lastErr, "reason", reason, "attempt", attempt+1)
	}
	if lastErr != nil {
		h.logger.Error("all flush retries exhausted, routing to DLQ",
			"pending", batcher.pending(), "error", lastErr)
		batcher.drainToDLQ(h.dlqProducer, h.logger, session, lastErr)
	}
}

// sendRawToDLQ forwards a single unprocessable Kafka message to the dead-letter
// topic, attaching the original coordinates and failure reason as headers.
func (h *handler) sendRawToDLQ(msg *sarama.ConsumerMessage, reason string) {
	dlqMsg := &sarama.ProducerMessage{
		Topic: claims.TopicDLQ,
		Key:   sarama.ByteEncoder(msg.Key),
		Value: sarama.ByteEncoder(msg.Value),
		Headers: []sarama.RecordHeader{
			{Key: []byte("dlq-original-topic"), Value: []byte(msg.Topic)},
			{Key: []byte("dlq-original-partition"), Value: fmt.Appendf(nil, "%d", msg.Partition)},
			{Key: []byte("dlq-original-offset"), Value: fmt.Appendf(nil, "%d", msg.Offset)},
			{Key: []byte("dlq-consumer-group"), Value: []byte(h.group)},
			{Key: []byte("dlq-error"), Value: []byte(reason)},
			{Key: []byte("dlq-failed-at"), Value: []byte(time.Now().UTC().Format(time.RFC3339))},
		},
	}
	if _, _, err := h.dlqProducer.SendMessage(dlqMsg); err != nil {
		h.logger.Error("send to DLQ failed", "error", err,
			"original_topic", msg.Topic, "original_offset", msg.Offset)
	} else {
		h.logger.Warn("message routed to DLQ",
			"original_topic", msg.Topic, "original_partition", msg.Partition,
			"original_offset", msg.Offset, "reason", reason)
	}
}

type consumedEvent struct {
	msg   *sarama.ConsumerMessage
	claim claims.Claim
}

type eventBatcher struct {
	db     *sql.DB
	logger *slog.Logger
	group  string
	limit  int
	events []consumedEvent
}

func newEventBatcher(db *sql.DB, logger *slog.Logger, group string, limit int) *eventBatcher {
	return &eventBatcher{
		db:     db,
		logger: logger,
		group:  group,
		limit:  limit,
		events: make([]consumedEvent, 0, limit),
	}
}

func (b *eventBatcher) add(ev consumedEvent) {
	b.events = append(b.events, ev)
}

func (b *eventBatcher) pending() int {
	return len(b.events)
}

// flush writes all buffered events to DuckDB inside a single transaction.
// The INSERT uses ON CONFLICT DO NOTHING so re-delivered messages (at-least-once
// re-delivery after a crash before the Kafka offset was committed) are silently
// skipped rather than producing duplicates. Kafka offsets are only marked after
// a successful commit, preserving at-least-once semantics end-to-end.
func (b *eventBatcher) flush(ctx context.Context, session sarama.ConsumerGroupSession) error {
	if len(b.events) == 0 {
		return nil
	}

	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	const query = `
		INSERT INTO claims_events
			(topic, partition, message_offset, consumer_group,
			 claim_id, customer_id, claim_type, amount, event_time, consumed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC()
	skipped := 0
	for _, ev := range b.events {
		res, execErr := stmt.ExecContext(ctx,
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
		if execErr != nil {
			return execErr
		}
		if n, _ := res.RowsAffected(); n == 0 {
			skipped++
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	if skipped > 0 {
		b.logger.Warn("duplicate messages skipped (at-least-once re-delivery)",
			"skipped", skipped, "total", len(b.events))
	}

	for _, ev := range b.events {
		session.MarkMessage(ev.msg, "persisted-to-duckdb")
	}
	b.events = b.events[:0]

	return nil
}

// drainToDLQ forwards every buffered event to the dead-letter topic, marks each
// offset as committed, and clears the buffer. Called when all flush retries fail.
func (b *eventBatcher) drainToDLQ(
	producer sarama.SyncProducer,
	logger *slog.Logger,
	session sarama.ConsumerGroupSession,
	reason error,
) {
	reasonStr := ""
	if reason != nil {
		reasonStr = reason.Error()
	}
	failedAt := time.Now().UTC().Format(time.RFC3339)
	for _, ev := range b.events {
		dlqMsg := &sarama.ProducerMessage{
			Topic: claims.TopicDLQ,
			Key:   sarama.ByteEncoder(ev.msg.Key),
			Value: sarama.ByteEncoder(ev.msg.Value),
			Headers: []sarama.RecordHeader{
				{Key: []byte("dlq-original-topic"), Value: []byte(ev.msg.Topic)},
				{Key: []byte("dlq-original-partition"), Value: fmt.Appendf(nil, "%d", ev.msg.Partition)},
				{Key: []byte("dlq-original-offset"), Value: fmt.Appendf(nil, "%d", ev.msg.Offset)},
				{Key: []byte("dlq-consumer-group"), Value: []byte(b.group)},
				{Key: []byte("dlq-error"), Value: []byte(reasonStr)},
				{Key: []byte("dlq-failed-at"), Value: []byte(failedAt)},
			},
		}
		if _, _, err := producer.SendMessage(dlqMsg); err != nil {
			logger.Error("drain to DLQ failed", "error", err,
				"topic", ev.msg.Topic, "offset", ev.msg.Offset)
		}
		session.MarkMessage(ev.msg, "dlq-forwarded")
	}
	if n := len(b.events); n > 0 {
		logger.Warn("batch drained to DLQ", "count", n, "reason", reasonStr)
	}
	b.events = b.events[:0]
}

func ensureSchema(ctx context.Context, db *sql.DB) error {
	// UNIQUE on (topic, partition, message_offset, consumer_group) is the
	// idempotency key: if the consumer crashes after writing to DuckDB but
	// before committing the Kafka offset, Kafka re-delivers those messages on
	// restart. ON CONFLICT DO NOTHING in flush() then silently skips them.
	// NOTE: if you have an existing DB without this constraint, delete the
	// .duckdb file and let it be recreated (default path is /tmp).
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
			consumed_at TIMESTAMP NOT NULL,
			UNIQUE (topic, partition, message_offset, consumer_group)
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
