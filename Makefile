.PHONY: test race vet check generate build

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
