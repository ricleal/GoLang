# ============================================================
# real_time_metrics_agg – Makefile
# Run from:  real_time_metrics_agg/
# Module root (go.mod) lives one level up.
# ============================================================

ROOT := $(shell cd .. && pwd)
PROTO_DIR  := proto
GEN_DIR    := gen

.PHONY: help deps proto build test test-race \
        redis-up redis-down \
        server-1 server-2 server-3 \
        client client-2 client-3 \
        stack-up stack-down clean

# ── default target ──────────────────────────────────────────
help:
	@echo ""
	@echo "Available targets:"
	@echo "  deps        – add Go dependencies (redis, grpc)"
	@echo "  proto       – generate gRPC Go code from proto/metrics.proto"
	@echo "  build       – compile server and client binaries"
	@echo "  test        – run unit tests (verbose)"
	@echo "  test-race   – run unit tests with -race detector"
	@echo "  test-integration – run integration tests (requires Docker)"
	@echo "  bench       – run benchmarks"
	@echo "  redis-up    – start Redis via Docker Compose"
	@echo "  redis-down  – stop Redis"
	@echo "  server-1/2/3 – run a worker node (ports 50051/50052/50053)"
	@echo "  client      – run a streaming client against node-1"
	@echo "  client-2    – run a client against node-2"
	@echo "  client-3    – run a client against node-3"
	@echo "  client-auto – run a client with Redis auto-discovery"
	@echo "  stack-up    – start Redis + 3 servers via Docker Compose"
	@echo "  stack-down  – stop the full Docker Compose stack"
	@echo "  clean       – remove generated files and binaries"
	@echo ""

# ── dependencies ────────────────────────────────────────────
deps:
	cd $(ROOT) && go get \
		github.com/redis/go-redis/v9 \
		google.golang.org/grpc \
		google.golang.org/grpc/health \
		google.golang.org/grpc/health/grpc_health_v1
	cd $(ROOT) && go mod tidy

# ── proto generation ─────────────────────────────────────────
proto:
	mkdir -p $(GEN_DIR)
	cd $(ROOT) && protoc \
		--go_out=. --go_opt=module=exp \
		--go-grpc_out=. --go-grpc_opt=module=exp \
		real_time_metrics_agg/$(PROTO_DIR)/metrics.proto

# ── build ────────────────────────────────────────────────────
build: proto
	cd $(ROOT) && go build -o real_time_metrics_agg/bin/server ./real_time_metrics_agg/server
	cd $(ROOT) && go build -o real_time_metrics_agg/bin/client ./real_time_metrics_agg/client

# ── tests ────────────────────────────────────────────────────
test:
	cd $(ROOT) && go test -v ./real_time_metrics_agg/...

test-race:
	cd $(ROOT) && go test -race -v ./real_time_metrics_agg/...

test-integration:
	cd $(ROOT) && go test -v -tags=integration -timeout=120s \
		./real_time_metrics_agg/server/ -run "TestIntegration_"

bench:
	cd $(ROOT) && go test -bench=. -benchmem ./real_time_metrics_agg/...

# ── Redis ────────────────────────────────────────────────────
redis-up:
	docker compose up -d redis

redis-down:
	docker compose down

# ── server nodes (run each in a separate terminal) ───────────
server-1:
	cd $(ROOT) && go run -race ./real_time_metrics_agg/server \
		--port=50051 --node-id=node-1 --redis=localhost:6379

server-2:
	cd $(ROOT) && go run -race ./real_time_metrics_agg/server \
		--port=50052 --node-id=node-2 --redis=localhost:6379

server-3:
	cd $(ROOT) && go run -race ./real_time_metrics_agg/server \
		--port=50053 --node-id=node-3 --redis=localhost:6379

# ── clients ──────────────────────────────────────────────────
client:
	cd $(ROOT) && go run ./real_time_metrics_agg/client \
		--server=localhost:50051 --node-id=client-1

client-2:
	cd $(ROOT) && go run ./real_time_metrics_agg/client \
		--server=localhost:50052 --node-id=client-2

client-3:
	cd $(ROOT) && go run ./real_time_metrics_agg/client \
		--server=localhost:50053 --node-id=client-3

# Auto-discovery client – queries Redis for available nodes and picks one.
client-auto:
	cd $(ROOT) && go run ./real_time_metrics_agg/client \
		--redis=localhost:6379 --node-id=client-auto

# ── full Docker Compose stack ─────────────────────────────────
# Note: does NOT depend on `build` because the Dockerfile compiles
# the binary inside the container and does not need protoc on the host.
stack-up:
	docker compose up --build -d

stack-down:
	docker compose down

# ── clean ─────────────────────────────────────────────────────
clean:
	rm -rf $(GEN_DIR) bin
