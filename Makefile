BIN := dango

.PHONY: build test run

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN) ./cmd/dango

test:
	go test ./...

run:
	go run ./cmd/dango
