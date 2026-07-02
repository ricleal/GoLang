package raft

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	hashraft "github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

// AlertEvent is replicated across the Raft cluster.
type AlertEvent struct {
	MetricName   string  `json:"metric_name"`
	Threshold    float64 `json:"threshold"`
	CurrentValue float64 `json:"current_value"`
	Severity     string  `json:"severity"`
	Message      string  `json:"message"`
	TriggeredAt  int64   `json:"triggered_at"`
	LeaderID     string  `json:"leader_id"`
}

// ClusterFSM implements raft.FSM — stores alert events in memory.
type ClusterFSM struct {
	Alerts []AlertEvent
}

// Apply is called when a log entry is committed.
func (f *ClusterFSM) Apply(log *hashraft.Log) interface{} {
	var event AlertEvent
	if err := json.Unmarshal(log.Data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal alert: %w", err)
	}
	f.Alerts = append(f.Alerts, event)
	if len(f.Alerts) > 1000 {
		f.Alerts = f.Alerts[len(f.Alerts)-1000:]
	}
	return nil
}

// Snapshot returns a point-in-time snapshot.
func (f *ClusterFSM) Snapshot() (hashraft.FSMSnapshot, error) {
	return &ClusterSnapshot{Alerts: f.Alerts}, nil
}

// Restore restores state from a snapshot.
func (f *ClusterFSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	var alerts []AlertEvent
	if err := json.NewDecoder(rc).Decode(&alerts); err != nil {
		return err
	}
	f.Alerts = alerts
	return nil
}

// ClusterSnapshot implements raft.FSMSnapshot.
type ClusterSnapshot struct {
	Alerts []AlertEvent
}

func (s *ClusterSnapshot) Persist(sink hashraft.SnapshotSink) error {
	err := json.NewEncoder(sink).Encode(s.Alerts)
	if err1 := sink.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

func (s *ClusterSnapshot) Release() {}

// ClusterNode wraps the Raft instance.
type ClusterNode struct {
	Raft    *hashraft.Raft
	NodeID  string
	DataDir string
	fsm     *ClusterFSM
	logger  hclog.Logger
}

// NewClusterNode creates and starts a Raft node.
func NewClusterNode(nodeID, raftAddr, dataDir string, bootstrap bool, logger hclog.Logger) (*ClusterNode, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir data dir: %w", err)
	}

	config := hashraft.DefaultConfig()
	config.LocalID = hashraft.ServerID(nodeID)
	config.Logger = logger
	config.SnapshotInterval = 30 * time.Second
	config.SnapshotThreshold = 128

	// FSM
	fsm := &ClusterFSM{}

	// Log store (bolt)
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft-log.db"))
	if err != nil {
		return nil, fmt.Errorf("bolt log store: %w", err)
	}

	// Stable store
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft-stable.db"))
	if err != nil {
		return nil, fmt.Errorf("bolt stable store: %w", err)
	}

	// Snapshot store
	snapStore, err := hashraft.NewFileSnapshotStore(dataDir, 3, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("snapshot store: %w", err)
	}

	// Transport
	transport, err := hashraft.NewTCPTransport(raftAddr, nil, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("tcp transport: %w", err)
	}

	// Create Raft instance
	raftNode, err := hashraft.NewRaft(config, fsm, logStore, stableStore, snapStore, transport)
	if err != nil {
		return nil, fmt.Errorf("new raft: %w", err)
	}

	if bootstrap {
		cfg := hashraft.Configuration{
			Servers: []hashraft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		future := raftNode.BootstrapCluster(cfg)
		if err := future.Error(); err != nil {
			// If already bootstrapped, that's fine
			logger.Warn("bootstrap (may be ok)", "error", err)
		}
	}

	return &ClusterNode{
		Raft:    raftNode,
		NodeID:  nodeID,
		DataDir: dataDir,
		fsm:     fsm,
		logger:  logger,
	}, nil
}

// IsLeader returns true if this node is the Raft leader.
func (n *ClusterNode) IsLeader() bool {
	return n.Raft.State() == hashraft.Leader
}

// LeaderID returns the current leader's address.
func (n *ClusterNode) LeaderAddr() string {
	return string(n.Raft.Leader())
}

// Join adds a peer to the cluster.
func (n *ClusterNode) Join(nodeID, addr string) error {
	if !n.IsLeader() {
		// Forward to leader
		return fmt.Errorf("not the leader")
	}
	future := n.Raft.AddVoter(hashraft.ServerID(nodeID), hashraft.ServerAddress(addr), 0, 0)
	return future.Error()
}

// ReplicateAlert replicates an alert event to the Raft cluster.
func (n *ClusterNode) ReplicateAlert(event AlertEvent) error {
	if !n.IsLeader() {
		return fmt.Errorf("not the leader, leader is at %s", n.LeaderAddr())
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	future := n.Raft.Apply(data, 5*time.Second)
	return future.Error()
}

// RecentClusterAlerts returns the most recent replicated alerts from the FSM.
func (n *ClusterNode) RecentClusterAlerts(count int) []AlertEvent {
	alerts := n.fsm.Alerts
	if count > len(alerts) {
		count = len(alerts)
	}
	return alerts[len(alerts)-count:]
}
