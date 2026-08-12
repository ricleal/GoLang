COMPOSE = docker compose
PROFILES_ALL = --profile core --profile analysis --profile observability

.PHONY: env up-core up-analysis up-obs up-all ps logs-debezium verify maintain verify-unit down clean

env:
	@test -f .env || cp .env.example .env

up-core: env
	$(COMPOSE) --profile core up -d --build

up-analysis: env
	$(COMPOSE) --profile core --profile analysis up -d --build

up-obs: env
	$(COMPOSE) --profile core --profile observability up -d --build

up-all: env
	$(COMPOSE) $(PROFILES_ALL) up -d --build

ps: env
	$(COMPOSE) $(PROFILES_ALL) --profile verify ps

logs-debezium:
	$(COMPOSE) logs -f debezium

verify: env
	$(COMPOSE) run --build --rm verify

maintain: env
	$(COMPOSE) run --build --rm verify dq maintain

verify-unit:
	go test ./internal/dq/...

down: env
	$(COMPOSE) $(PROFILES_ALL) --profile verify down

clean: env
	$(COMPOSE) $(PROFILES_ALL) --profile verify down -v
