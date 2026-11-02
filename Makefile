.PHONY: test race vet check generate build frontend-install frontend-dev frontend-build migrate-up migrate-down migrate-version

test:
	cd backend && go test ./... -cover

race:
	cd backend && go test -race ./...

vet:
	cd backend && go vet ./...

check: test race vet

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
