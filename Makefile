.PHONY: test race vet check generate build migrate-up migrate-down migrate-version

test:
	go test ./... -cover

race:
	go test -race ./...

vet:
	go vet ./...

check: test race vet

.PHONY: lint lint-go fmt-check fmt

lint: lint-go

lint-go:
	golangci-lint run --config .golangci.yml ./...

fmt-check:
	golangci-lint fmt --config .golangci.yml --diff

fmt:
	golangci-lint fmt --config .golangci.yml

generate:
	go generate ./core/ports

build:
	go build ./cmd/shorter

migrate-up:
	docker compose run --rm migrate up

migrate-down:
	docker compose run --rm migrate down 1

migrate-version:
	docker compose run --rm migrate version
