SHELL := /bin/bash

API_DIR    := src/backend
WEB_DIR    := src/frontend
COMPOSE_DIR := deploy/compose

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

## ---- Backend ----

.PHONY: api-tidy
api-tidy: ## go mod tidy
	cd $(API_DIR) && go mod tidy

.PHONY: api-build
api-build: ## Build api + worker binaries into bin/
	cd $(API_DIR) && go build -o ../../bin/api ./cmd/api && go build -o ../../bin/worker ./cmd/worker

.PHONY: api-run
api-run: ## Run the API locally
	cd $(API_DIR) && go run ./cmd/api

.PHONY: worker-run
worker-run: ## Run the worker locally
	cd $(API_DIR) && go run ./cmd/worker

.PHONY: migrate
migrate: ## Apply DB migrations
	cd $(API_DIR) && go run ./cmd/migrate

.PHONY: seed
seed: ## Seed the host account from env
	cd $(API_DIR) && go run ./cmd/migrate -seed

## Run the microservices separately (each listens on its own port).
## Start each in its own terminal; the gateway fronts them on :8080.
.PHONY: identity-run submission-run conversation-run attachment-run gateway-run
identity-run: ## Run identity service (:8081)
	cd $(API_DIR) && HTTP_ADDR=:8081 go run ./cmd/identity
submission-run: ## Run submission service (:8082)
	cd $(API_DIR) && HTTP_ADDR=:8082 go run ./cmd/submission
conversation-run: ## Run conversation service (:8083)
	cd $(API_DIR) && HTTP_ADDR=:8083 go run ./cmd/conversation
attachment-run: ## Run attachment service (:8084)
	cd $(API_DIR) && HTTP_ADDR=:8084 go run ./cmd/attachment
gateway-run: ## Run API gateway (:8080)
	cd $(API_DIR) && GATEWAY_ADDR=:8080 go run ./cmd/gateway

.PHONY: test
test: ## Run Go tests
	cd $(API_DIR) && go test ./...

.PHONY: fmt
fmt: ## gofmt the backend
	cd $(API_DIR) && gofmt -w .

.PHONY: vet
vet: ## go vet
	cd $(API_DIR) && go vet ./...

.PHONY: lint
lint: fmt vet ## Format and vet

## ---- Frontend ----

.PHONY: web-install
web-install: ## Install frontend deps
	cd $(WEB_DIR) && npm install

.PHONY: web-dev
web-dev: ## Run frontend dev server
	cd $(WEB_DIR) && npm run dev

.PHONY: web-build
web-build: ## Build frontend
	cd $(WEB_DIR) && npm run build

.PHONY: web-lint
web-lint: ## Lint frontend
	cd $(WEB_DIR) && npm run lint

## ---- Stack ----

.PHONY: up
up: ## Start the whole stack (docker compose)
	docker compose -f $(COMPOSE_DIR)/docker-compose.yml --env-file $(COMPOSE_DIR)/.env up -d --build

.PHONY: down
down: ## Stop the stack
	docker compose -f $(COMPOSE_DIR)/docker-compose.yml --env-file $(COMPOSE_DIR)/.env down

.PHONY: micro-up
micro-up: ## Start the microservice stack (compose)
	docker compose -f $(COMPOSE_DIR)/docker-compose.micro.yml --env-file $(COMPOSE_DIR)/.env up -d --build

.PHONY: micro-down
micro-down: ## Stop the microservice stack
	docker compose -f $(COMPOSE_DIR)/docker-compose.micro.yml --env-file $(COMPOSE_DIR)/.env down

.PHONY: logs
logs: ## Tail stack logs
	docker compose -f $(COMPOSE_DIR)/docker-compose.yml --env-file $(COMPOSE_DIR)/.env logs -f --tail=100

.PHONY: check
check: lint test web-lint web-build ## Run all checks
