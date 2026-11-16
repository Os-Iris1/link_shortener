.PHONY: test race vet check generate build frontend-install frontend-dev frontend-build migrate-up migrate-down migrate-version

test:
	cd backend && go test ./... -cover

race:
	cd backend && go test -race ./...

vet:
	cd backend && go vet ./...

check: test race vet

.PHONY: lint lint-go lint-frontend fmt-check fmt

lint: lint-go lint-frontend

lint-go:
	cd backend && golangci-lint run --config ../.golangci.yml ./...

lint-frontend:
	cd frontend && npm run lint

fmt-check:
	go run ./scripts/go-format.go

fmt:
	go run ./scripts/go-format.go --write

generate:
	cd backend && go generate ./core/ports

build:
	cd backend && go build ./cmd/shorter

frontend-install:
	cd frontend && npm ci

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

migrate-up:
	docker compose run --rm migrate up

migrate-down:
	docker compose run --rm migrate down 1

migrate-version:
	docker compose run --rm migrate version
