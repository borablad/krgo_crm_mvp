SHELL := /bin/bash
export PATH := $(HOME)/.local/go/bin:$(HOME)/go/bin:$(PATH)

DEV_DB_URL  ?= postgres://cargo:cargo@localhost:5432/cargo?sslmode=disable
TEST_DB_URL ?= postgres://cargo:cargo@localhost:5432/cargo_test?sslmode=disable
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS      = -s -w -X github.com/borablad/krgo_crm_mvp/backend/internal/app.Version=$(VERSION)

.PHONY: help dev-db dev-db-down migrate seed run build web check vet test test-db frontend-check sqlc api-types

help: ## список команд
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

dev-db: ## Postgres для разработки и тестов в Docker (:5432)
	docker compose -f deploy/compose.dev.yml up -d --wait

dev-db-down: ## остановить локальный Postgres
	docker compose -f deploy/compose.dev.yml down

migrate: ## применить миграции к локальной базе
	cd backend && DATABASE_URL="$(DEV_DB_URL)" go run ./cmd/cargo migrate

seed: ## демо-данные в локальную базу
	cd backend && DATABASE_URL="$(DEV_DB_URL)" go run ./cmd/cargo seed

run: ## запустить бэкенд на :8080
	cd backend && DATABASE_URL="$(DEV_DB_URL)" go run ./cmd/cargo serve

build: ## собрать бинарник backend/bin/cargo
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/cargo ./cmd/cargo

web: ## собрать фронтенд и положить в backend/internal/web/dist
	cd frontend && npm run build
	rm -rf backend/internal/web/dist && cp -r frontend/dist backend/internal/web/dist

sqlc: ## перегенерировать store-пакеты из SQL
	cd backend && sqlc generate

api-types: ## перегенерировать TypeScript-типы API из OpenAPI
	cd backend && DATABASE_URL=unused go run ./cmd/cargo openapi > ../frontend/openapi.json
	cd frontend && npm run api-types

vet:
	cd backend && go vet ./...

test: ## тесты бэкенда (с базой, если она поднята через make dev-db)
	cd backend && TEST_DATABASE_URL="$(TEST_DB_URL)" go test ./...

test-db: dev-db test

frontend-check: ## typecheck и сборка фронтенда
	cd frontend && npm run check

check: vet test frontend-check ## всё, что должно быть зелёным перед сдачей задачи
