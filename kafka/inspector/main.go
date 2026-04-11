package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/IBM/sarama"

	"exp/kafka/claims"
)

const tabPadding = 2 // minimum column padding for tabwriter output

// run connects to Kafka, prints a snapshot of topic offsets and consumer-group
// lag, then either exits (interval==0) or refreshes on a ticker.
func run() int {
	broker := flag.String("broker", claims.BrokerAddr, "Kafka broker address")
	interval := flag.Duration("interval", 5*time.Second, "Refresh interval (0 = run once)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_6_0_0

	client, err := sarama.NewClient([]string{*broker}, cfg)
	if err != nil {
		logger.Error("connect", "error", err)
		return 1
	}
	defer client.Close()

	admin, adminErr := sarama.NewClusterAdminFromClient(client)
	if adminErr != nil {
		logger.Error("admin client", "error", adminErr)
		return 1
	}
	defer admin.Close()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	printOnce := func() {
		if inspectErr := inspect(client, admin); inspectErr != nil {
			logger.Error("inspect", "error", inspectErr)
		}
	}

	printOnce()
	if *interval == 0 {
		return 0
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		select {
		case <-quit:
			return 0
		case <-ticker.C:
			printOnce()
		}
	}
}

func main() {
	os.Exit(run())
}

// ── types ────────────────────────────────────────────────────────────────────

// partStats holds the oldest and newest committed offsets for one partition.
type partStats struct {
	id     int32
	oldest int64 // first available (may be >0 after log compaction / retention)
	newest int64 // next offset to be written (i.e. current message count from oldest)
}

// topicStats aggregates partition data and the total unread message count for a topic.
type topicStats struct {
	name  string
	parts []partStats
	total int64 // sum of (newest - oldest) across all partitions
}

// ── inspect ───────────────────────────────────────────────────────────────────

// inspect prints one full snapshot: topic offsets followed by consumer-group lag.
func inspect(client sarama.Client, admin sarama.ClusterAdmin) error {
	fmt.Fprintf(os.Stdout, "\n%s\n", time.Now().Format("2006-01-02 15:04:05"))

	allTopicStats, err := gatherTopicStats(client, admin)
	if err != nil {
		return err
	}

	printTopics(allTopicStats)

	groups, listErr := admin.ListConsumerGroups()
	if listErr != nil {
		return fmt.Errorf("list consumer groups: %w", listErr)
	}

	relevantGroups := filterGroups(groups)
	if len(relevantGroups) == 0 {
		fmt.Fprintln(os.Stdout, "\n── Consumer Groups ──  (none found for our topics)")
		return nil
	}

	return printGroupLag(admin, allTopicStats, relevantGroups)
}

// gatherTopicStats fetches metadata and partition offsets for all known topics.
func gatherTopicStats(client sarama.Client, admin sarama.ClusterAdmin) ([]topicStats, error) {
	topicsMeta, err := admin.DescribeTopics(claims.Topics)
	if err != nil {
		return nil, fmt.Errorf("describe topics: %w", err)
	}

	var all []topicStats
	for _, meta := range topicsMeta {
		if meta.Err != sarama.ErrNoError {
			fmt.Fprintf(os.Stderr, "  topic %s: %v\n", meta.Name, meta.Err)
			continue
		}
		ts := buildTopicStats(client, meta)
		all = append(all, ts)
	}
	return all, nil
}

// buildTopicStats resolves oldest/newest offsets for each partition of meta.
func buildTopicStats(client sarama.Client, meta *sarama.TopicMetadata) topicStats {
	ts := topicStats{name: meta.Name}
	for _, p := range meta.Partitions {
		oldest, _ := client.GetOffset(meta.Name, p.ID, sarama.OffsetOldest)
		newest, _ := client.GetOffset(meta.Name, p.ID, sarama.OffsetNewest)
		if oldest >= 0 && newest >= oldest {
			ts.total += newest - oldest
		}
		ts.parts = append(ts.parts, partStats{id: p.ID, oldest: oldest, newest: newest})
	}
	sort.Slice(ts.parts, func(i, j int) bool { return ts.parts[i].id < ts.parts[j].id })
	return ts
}

func printTopics(all []topicStats) {
	fmt.Fprintln(os.Stdout, "── Topics ───────────────────────────────────────────────────────")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(w, "  TOPIC\tPARTITION\tOLDEST\tNEWEST\tMESSAGES")
	for _, ts := range all {
		for _, p := range ts.parts {
			fmt.Fprintf(w, "  %s\t%d\t%d\t%d\t%d\n", ts.name, p.id, p.oldest, p.newest, p.newest-p.oldest)
		}
		fmt.Fprintf(w, "  %s\t(total)\t-\t-\t%d\n", ts.name, ts.total)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "flush: %v\n", err)
	}
}

// printGroupLag writes the consumer-group lag table for all relevant groups.
func printGroupLag(admin sarama.ClusterAdmin, allTopicStats []topicStats, groups []string) error {
	fmt.Fprintln(os.Stdout, "\n── Consumer Group Lag ───────────────────────────────────────────")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(w, "  GROUP\tTOPIC\tPARTITION\tCOMMITTED\tNEWEST\tLAG")

	tpMap := buildTopicPartitionMap(allTopicStats)

	for _, group := range groups {
		offsets, err := admin.ListConsumerGroupOffsets(group, tpMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  group %s offsets: %v\n", group, err)
			continue
		}
		writeGroupRows(w, group, allTopicStats, offsets)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush lag table: %w", err)
	}
	return nil
}

// buildTopicPartitionMap constructs the topic→partition-list map needed by
// ListConsumerGroupOffsets.
func buildTopicPartitionMap(allTopicStats []topicStats) map[string][]int32 {
	tpMap := map[string][]int32{}
	for _, ts := range allTopicStats {
		for _, p := range ts.parts {
			tpMap[ts.name] = append(tpMap[ts.name], p.id)
		}
	}
	return tpMap
}

// writeGroupRows appends one row per partition (plus a total row) for the given
// consumer group to w. Partitions where the group has never committed (offset -1)
// are skipped to avoid misleading lag figures.
func writeGroupRows(
	w *tabwriter.Writer,
	group string,
	allTopicStats []topicStats,
	offsets *sarama.OffsetFetchResponse,
) {
	for _, ts := range allTopicStats {
		tpOffsets, ok := offsets.Blocks[ts.name]
		if !ok {
			continue
		}
		var groupLag int64
		hasAny := false
		for _, p := range ts.parts {
			block := tpOffsets[p.id]
			// committed == -1 means this group has never consumed this topic; skip.
			if block == nil || block.Err != sarama.ErrNoError || block.Offset == -1 {
				continue
			}
			hasAny = true
			lag := max(p.newest-block.Offset, 0)
			groupLag += lag
			fmt.Fprintf(w, "  %s\t%s\t%d\t%d\t%d\t%d\n",
				group, ts.name, p.id, block.Offset, p.newest, lag)
		}
		if hasAny {
			fmt.Fprintf(w, "  %s\t%s\t(total)\t-\t-\t%d\n", group, ts.name, groupLag)
		}
	}
}

// filterGroups returns sorted group IDs, skipping internal Kafka groups.
func filterGroups(groups map[string]string) []string {
	var ids []string
	for id := range groups {
		if !strings.HasPrefix(id, "__") {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}
