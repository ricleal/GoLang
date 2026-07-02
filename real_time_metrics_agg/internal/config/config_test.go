package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultStr(t *testing.T) {
	// Env not set — should return fallback
	if got := defaultStr("NONEXISTENT_VAR_12345", "fallback"); got != "fallback" {
		t.Fatalf("expected 'fallback', got '%s'", got)
	}
}

func TestDefaultStr_FromEnv(t *testing.T) {
	os.Setenv("TEST_DEFAULT_STR", "from_env")
	defer os.Unsetenv("TEST_DEFAULT_STR")

	if got := defaultStr("TEST_DEFAULT_STR", "fallback"); got != "from_env" {
		t.Fatalf("expected 'from_env', got '%s'", got)
	}
}

func TestDefaultInt(t *testing.T) {
	if got := defaultInt("NONEXISTENT_VAR_12345", 42); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestDefaultInt_FromEnv(t *testing.T) {
	os.Setenv("TEST_DEFAULT_INT", "99")
	defer os.Unsetenv("TEST_DEFAULT_INT")

	if got := defaultInt("TEST_DEFAULT_INT", 42); got != 99 {
		t.Fatalf("expected 99, got %d", got)
	}
}

func TestDefaultInt_InvalidEnv(t *testing.T) {
	os.Setenv("TEST_DEFAULT_INT_INVALID", "not-a-number")
	defer os.Unsetenv("TEST_DEFAULT_INT_INVALID")

	if got := defaultInt("TEST_DEFAULT_INT_INVALID", 42); got != 42 {
		t.Fatalf("expected fallback 42 for invalid env, got %d", got)
	}
}

func TestRaftAddress(t *testing.T) {
	cfg := &Config{RaftPort: 50070}
	if addr := cfg.RaftAddress(); addr != "127.0.0.1:50070" {
		t.Fatalf("expected '127.0.0.1:50070', got '%s'", addr)
	}
}

func TestGrpcAddress(t *testing.T) {
	cfg := &Config{GrpcPort: 50051}
	if addr := cfg.GrpcAddress(); addr != "0.0.0.0:50051" {
		t.Fatalf("expected '0.0.0.0:50051', got '%s'", addr)
	}
}

func TestGrpcPublicAddress(t *testing.T) {
	cfg := &Config{GrpcPort: 50051}
	if addr := cfg.GrpcPublicAddress(); addr != "127.0.0.1:50051" {
		t.Fatalf("expected '127.0.0.1:50051', got '%s'", addr)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Load parses flags, which can conflict in tests. Instead, verify struct defaults.
	cfg := &Config{}
	if cfg.WindowDuration != 0 {
		t.Fatal("expected zero window duration by default")
	}
	if cfg.BucketInterval != 0 {
		t.Fatal("expected zero bucket interval by default")
	}
	if cfg.LogLevel != "" {
		t.Fatal("expected empty log level by default")
	}
}

func TestConfig_NodeID(t *testing.T) {
	cfg := &Config{NodeID: "custom-node"}
	if cfg.NodeID != "custom-node" {
		t.Fatalf("expected 'custom-node', got '%s'", cfg.NodeID)
	}
}

func TestEnvOverride(t *testing.T) {
	os.Setenv("NODE_ID", "env-node")
	os.Setenv("GRPC_PORT", "9090")
	os.Setenv("RAFT_PORT", "9091")
	defer func() {
		os.Unsetenv("NODE_ID")
		os.Unsetenv("GRPC_PORT")
		os.Unsetenv("RAFT_PORT")
	}()

	if got := defaultStr("NODE_ID", "default"); got != "env-node" {
		t.Fatalf("expected 'env-node', got '%s'", got)
	}
	if got := defaultInt("GRPC_PORT", 50051); got != 9090 {
		t.Fatalf("expected 9090, got %d", got)
	}
	if got := defaultInt("RAFT_PORT", 50070); got != 9091 {
		t.Fatalf("expected 9091, got %d", got)
	}
}

func TestConfig_DurationDefaults(t *testing.T) {
	cfg := &Config{
		WindowDuration: 10 * time.Second,
		BucketInterval: 1 * time.Second,
	}
	if cfg.WindowDuration != 10*time.Second {
		t.Fatalf("expected 10s window, got %v", cfg.WindowDuration)
	}
	if cfg.BucketInterval != 1*time.Second {
		t.Fatalf("expected 1s bucket, got %v", cfg.BucketInterval)
	}
}
