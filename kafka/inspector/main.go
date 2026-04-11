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

func main() {
	broker := flag.String("broker", claims.BrokerAddr, "Kafka broker address")
	interval := flag.Duration("interval", 5*time.Second, "Refresh interval (0 = run once)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_6_0_0

	client, err := sarama.NewClient([]string{*broker}, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "admin client: %v\n", err)
		os.Exit(1)
	}
	defer admin.Close()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	print := func() {
		if err := inspect(client, admin); err != nil {
			fmt.Fprintf(os.Stderr, "inspect error: %v\n", err)
		}
	}

	print()
	if *interval == 0 {
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			print()
		}
	}
}

// ── main inspection ───────────────────────────────────────────────────────────

func inspect(client sarama.Client, admin sarama.ClusterAdmin) error {
	fmt.Printf("\n%s\n", time.Now().Format("2006-01-02 15:04:05"))

	// ── topics ───────────────────────────────────────────────────────────────
	topicsMeta, err := admin.DescribeTopics(claims.Topics)
	if err != nil {
		return fmt.Errorf("describe topics: %w", err)
	}

	type partStats struct {
		id     int32
		oldest int64
		newest int64
	}
	type topicStats struct {
		name  string
		parts []partStats
		total int64
	}

	var allTopicStats []topicStats
	for _, meta := range topicsMeta {
		if meta.Err != sarama.ErrNoError {
			fmt.Printf("  topic %s: %v\n", meta.Name, meta.Err)
			continue
		}

		ts := topicStats{name: meta.Name}
		for _, p := range meta.Partitions {
			oldest, err := client.GetOffset(meta.Name, p.ID, sarama.OffsetOldest)
			if err != nil {
				oldest = -1
			}
			newest, err := client.GetOffset(meta.Name, p.ID, sarama.OffsetNewest)
			if err != nil {
				newest = -1
			}
			msgs := int64(0)
			if oldest >= 0 && newest >= oldest {
				msgs = newest - oldest
			}
			ts.parts = append(ts.parts, partStats{id: p.ID, oldest: oldest, newest: newest})
			ts.total += msgs
		}
		sort.Slice(ts.parts, func(i, j int) bool { return ts.parts[i].id < ts.parts[j].id })
		allTopicStats = append(allTopicStats, ts)
	}

	fmt.Println("── Topics ───────────────────────────────────────────────────────")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  TOPIC\tPARTITION\tOLDEST\tNEWEST\tMESSAGES")
	for _, ts := range allTopicStats {
		for _, p := range ts.parts {
			msgs := p.newest - p.oldest
			fmt.Fprintf(w, "  %s\t%d\t%d\t%d\t%d\n", ts.name, p.id, p.oldest, p.newest, msgs)
		}
		fmt.Fprintf(w, "  %s\t(total)\t-\t-\t%d\n", ts.name, ts.total)
	}
	w.Flush()

	// ── consumer groups ───────────────────────────────────────────────────────
	groups, err := admin.ListConsumerGroups()
	if err != nil {
		return fmt.Errorf("list consumer groups: %w", err)
	}

	// Filter to groups that consume our topics.
	relevantGroups := filterGroups(admin, groups, claims.Topics)
	if len(relevantGroups) == 0 {
		fmt.Println("\n── Consumer Groups ──  (none found for our topics)")
		return nil
	}

	fmt.Println("\n── Consumer Group Lag ───────────────────────────────────────────")
	w2 := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w2, "  GROUP\tTOPIC\tPARTITION\tCOMMITTED\tNEWEST\tLAG")

	for _, group := range relevantGroups {
		// Build topic→partitions map for this group.
		tpMap := map[string][]int32{}
		for _, ts := range allTopicStats {
			for _, p := range ts.parts {
				tpMap[ts.name] = append(tpMap[ts.name], p.id)
			}
		}

		offsets, err := admin.ListConsumerGroupOffsets(group, tpMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  group %s offsets: %v\n", group, err)
			continue
		}

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
				committed := block.Offset
				lag := p.newest - committed
				if lag < 0 {
					lag = 0
				}
				groupLag += lag
				fmt.Fprintf(w2, "  %s\t%s\t%d\t%d\t%d\t%d\n",
					group, ts.name, p.id, committed, p.newest, lag)
			}
			if hasAny {
				fmt.Fprintf(w2, "  %s\t%s\t(total)\t-\t-\t%d\n", group, ts.name, groupLag)
			}
		}
	}
	w2.Flush()
	return nil
}

// filterGroups returns only the group IDs that have committed offsets for at
// least one of the given topics.
func filterGroups(admin sarama.ClusterAdmin, groups map[string]string, topics []string) []string {
	topicSet := make(map[string]struct{}, len(topics))
	for _, t := range topics {
		topicSet[t] = struct{}{}
	}

	var ids []string
	for id := range groups {
		// Quick heuristic: skip internal / unrelated groups.
		if strings.HasPrefix(id, "__") {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
