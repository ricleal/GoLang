package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Node identity
	NodeID   string
	RaftPort int
	GrpcPort int
	DataDir  string

	// Cluster
	Bootstrap   bool
	JoinAddress string

	// Aggregation
	WindowDuration time.Duration
	BucketInterval time.Duration

	// Logging
	LogLevel string
}

func Load() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.NodeID, "node-id", defaultStr("NODE_ID", "node1"), "Unique node ID")
	flag.IntVar(&cfg.RaftPort, "raft-port", defaultInt("RAFT_PORT", 50070), "Raft internal port")
	flag.IntVar(&cfg.GrpcPort, "grpc-port", defaultInt("GRPC_PORT", 50051), "gRPC server port")
	flag.StringVar(&cfg.DataDir, "data-dir", defaultStr("DATA_DIR", "/tmp/raft-metrics"), "Raft data directory")
	flag.BoolVar(&cfg.Bootstrap, "bootstrap", false, "Bootstrap the Raft cluster (first node only)")
	flag.StringVar(&cfg.JoinAddress, "join", defaultStr("JOIN_ADDR", ""), "Join an existing cluster at this address (host:raft-port)")
	flag.DurationVar(&cfg.WindowDuration, "window", 10*time.Second, "Sliding window duration")
	flag.DurationVar(&cfg.BucketInterval, "bucket", 1*time.Second, "Aggregation bucket interval")
	flag.StringVar(&cfg.LogLevel, "log-level", defaultStr("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	flag.Parse()

	return cfg
}

func defaultStr(env, fallback string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	return fallback
}

func defaultInt(env string, fallback int) int {
	if v := os.Getenv(env); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

// RaftAddress returns the address for Raft consensus.
func (c *Config) RaftAddress() string {
	return "127.0.0.1:" + strconv.Itoa(c.RaftPort)
}

// GrpcAddress returns the address for gRPC server.
func (c *Config) GrpcAddress() string {
	return "0.0.0.0:" + strconv.Itoa(c.GrpcPort)
}

// GrpcPublicAddress returns the publicly reachable gRPC address.
func (c *Config) GrpcPublicAddress() string {
	return "127.0.0.1:" + strconv.Itoa(c.GrpcPort)
}
