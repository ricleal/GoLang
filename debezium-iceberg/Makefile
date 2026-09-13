COMPOSE = docker compose
PROFILES_ALL = --profile core --profile analysis --profile observability

# Terminal styling for messages (kept minimal; works in most shells).
BOLD = \033[1m
CYAN = \033[36m
OFF  = \033[0m

# ---------------------------------------------------------------------------
# Every target below carries a `## help text` comment. `make help` lists them.
# ---------------------------------------------------------------------------

.PHONY: help env \
	up-core up-analysis up-obs up-all \
	run-dq run-maintain run-datagen \
	ps logs-debezium verify maintain \
	test verify-unit lint \
	down clean

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-20s$(OFF) %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

env: ## Create .env from .env.example on first run
	@test -f .env || cp .env.example .env

# ---------------------------------------------------------------------------
# Containerised runs (docker compose) - the whole stand lives in containers
# ---------------------------------------------------------------------------

up-core: env ## Start the core stack (postgres, minio, lakekeeper, debezium, datagen)
	$(COMPOSE) --profile core up -d --build
	@echo ""
	@echo "$(BOLD)Core stack is up:$(OFF)"
	@echo "  Debezium health:  $(CYAN)http://localhost:8080/q/health$(OFF)"
	@echo "  MinIO console:    $(CYAN)http://localhost:9001$(OFF)   (minio / minio12345)"
	@echo "  Lakekeeper UI:    $(CYAN)http://localhost:8181/ui/$(OFF)"

up-analysis: env ## Start core + Jupyter/GoNB analysis layer
	$(COMPOSE) --profile core --profile analysis up -d --build
	@echo ""
	@echo "$(BOLD)Analysis is up:$(OFF)"
	@echo "  Jupyter (GoNB):   $(CYAN)http://localhost:8888$(OFF)   (token lake, notebooks under work/)"

up-obs: env ## Start core + observability (dq-runner, prometheus, grafana, marquez)
	$(COMPOSE) --profile core --profile observability up -d --build
	@echo ""
	@echo "$(BOLD)Observability is up:$(OFF)"
	@echo "  DQ metrics:       $(CYAN)http://localhost:8000/metrics$(OFF)"
	@echo "  Prometheus:       $(CYAN)http://localhost:9090$(OFF)"
	@echo "  Grafana:          $(CYAN)http://localhost:3000$(OFF)   (admin / admin)"
	@echo "  Marquez UI:       $(CYAN)http://localhost:3001$(OFF)"

up-all: env ## Start everything (core + analysis + observability)
	$(COMPOSE) $(PROFILES_ALL) up -d --build
	@echo ""
	@echo "$(BOLD)Full stand is up:$(OFF)"
	@echo "  Debezium health:  $(CYAN)http://localhost:8080/q/health$(OFF)"
	@echo "  MinIO console:    $(CYAN)http://localhost:9001$(OFF)   (minio / minio12345)"
	@echo "  Lakekeeper UI:    $(CYAN)http://localhost:8181/ui/$(OFF)"
	@echo "  Jupyter (GoNB):   $(CYAN)http://localhost:8888$(OFF)   (token lake)"
	@echo "  DQ metrics:       $(CYAN)http://localhost:8000/metrics$(OFF)"
	@echo "  Prometheus:       $(CYAN)http://localhost:9090$(OFF)"
	@echo "  Grafana:          $(CYAN)http://localhost:3000$(OFF)   (admin / admin)"
	@echo "  Marquez UI:       $(CYAN)http://localhost:3001$(OFF)"

ps: ## Show the running containers
	$(COMPOSE) $(PROFILES_ALL) --profile verify ps

logs-debezium: ## Follow the Debezium container logs
	$(COMPOSE) logs -f debezium

verify: env ## Run the integration tests in a container (needs the stand up)
	$(COMPOSE) run --build --rm verify

maintain: env ## Expire old Iceberg snapshots in a container (dq maintain)
	$(COMPOSE) run --build --rm verify dq maintain

# ---------------------------------------------------------------------------
# Local (host) runs - same Go code, run with `go run` instead of a container.
# They need the core stack up (`make up-core`) and connect to the published
# ports (5433 postgres, 8181 lakekeeper, 9000 minio, 5002 marquez). .env is
# sourced for the tunables (DQ_*, DATAGEN_*, MAINT_*). Use these when you want
# to iterate on the Go source without rebuilding images.
#
# Lakekeeper advertises the in-network S3 endpoint (http://minio:9000), which
# the host cannot resolve. DQ_S3_ENDPOINT_OVERRIDE re-points the table I/O at
# the published port (localhost:9000) - no /etc/hosts edits, no sudo. Host runs
# also get a longer per-table scan budget (DQ_HTTP_TIMEOUT_MS): Docker Desktop
# port-forwarding adds latency vs the in-network container path (default 15s).
# ---------------------------------------------------------------------------

run-dq: env ## Run the DQ loop on the host (needs `make up-core`)
	@echo "$(BOLD)DQ metrics will be at:$(OFF) $(CYAN)http://localhost:8000/metrics$(OFF)"
	@set -a; . ./.env; set +a; \
	DQ_PG_DSN=postgresql://$${SOURCE_PG_USER:-postgres}:$${SOURCE_PG_PASSWORD:-postgres}@localhost:5433/postgres \
	DQ_ICEBERG_URI=http://localhost:8181/catalog \
	DQ_ICEBERG_WAREHOUSE=lakehouse \
	DQ_ICEBERG_NAMESPACE=cdc \
	DQ_TOPIC_PREFIX=dbz \
	DQ_S3_ENDPOINT=http://localhost:9000 \
	DQ_S3_ENDPOINT_OVERRIDE=http://localhost:9000 \
	DQ_S3_KEY=$${MINIO_ROOT_USER:-minio} \
	DQ_S3_SECRET=$${MINIO_ROOT_PASSWORD:-minio12345} \
	DQ_S3_REGION=local-01 \
	DQ_HTTP_TIMEOUT_MS=$${DQ_HTTP_TIMEOUT_MS:-90000} \
	DQ_TABLES=inventory.customers,inventory.products,inventory.products_on_hand,inventory.orders \
	OPENLINEAGE_URL=http://localhost:5002 \
	go run ./cmd/dq run

run-maintain: env ## Run Iceberg maintenance on the host (needs `make up-core`)
	@set -a; . ./.env; set +a; \
	DQ_ICEBERG_URI=http://localhost:8181/catalog \
	DQ_ICEBERG_WAREHOUSE=lakehouse \
	DQ_ICEBERG_NAMESPACE=cdc \
	DQ_S3_ENDPOINT=http://localhost:9000 \
	DQ_S3_ENDPOINT_OVERRIDE=http://localhost:9000 \
	DQ_S3_KEY=$${MINIO_ROOT_USER:-minio} \
	DQ_S3_SECRET=$${MINIO_ROOT_PASSWORD:-minio12345} \
	DQ_S3_REGION=local-01 \
	MAINT_MAX_AGE_HOURS=$${MAINT_MAX_AGE_HOURS:-2} \
	MAINT_RETAIN_LAST=$${MAINT_RETAIN_LAST:-20} \
	go run ./cmd/dq maintain

run-datagen: env ## Run the datagen on the host (needs `make up-core`)
	@set -a; . ./.env; set +a; \
	DATAGEN_DSN=postgresql://$${SOURCE_PG_USER:-postgres}:$${SOURCE_PG_PASSWORD:-postgres}@localhost:5433/postgres \
	DATAGEN_INTERVAL_SECONDS=$${DATAGEN_INTERVAL_SECONDS:-2} \
	go run ./cmd/datagen

# ---------------------------------------------------------------------------
# Tests / checks
# ---------------------------------------------------------------------------

test: ## Run the Go unit tests (fast, no stack required)
	go test ./internal/dq/...

verify-unit: test ## Alias of `test` (pure check functions, no stack)

lint: ## Run go vet over the whole module
	go vet ./...

# ---------------------------------------------------------------------------
# Teardown
# ---------------------------------------------------------------------------

down: ## Stop everything (volumes are kept)
	$(COMPOSE) $(PROFILES_ALL) --profile verify down

clean: ## Stop everything and remove volumes (fresh start)
	$(COMPOSE) $(PROFILES_ALL) --profile verify down -v
