.DEFAULT_GOAL := dev

COMPOSE ?= docker compose
COMPOSE_BUILD_ENV ?= BUILDX_GIT_INFO=0

.PHONY: dev up down build logs ps test web-build api-test hono-build

dev: up

up:
	$(COMPOSE_BUILD_ENV) $(COMPOSE) up --build

down:
	$(COMPOSE) down --remove-orphans

build:
	$(COMPOSE_BUILD_ENV) $(COMPOSE) build

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

web-build:
	cd apps/web && npm run build

api-test:
	cd apps/server && GOPATH=/tmp/go GOCACHE=/tmp/go-build GOTOOLCHAIN=local go test ./...

hono-build:
	npm --workspace @lirs/hono run build

test: web-build hono-build api-test
