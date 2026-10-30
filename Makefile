.PHONY: test race vet check generate build migrate-up migrate-down migrate-version

test:
	go test ./... -cover

race:
	go test -race ./...

vet:
	go vet ./...

check: test race vet

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
