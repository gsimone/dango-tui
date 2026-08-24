BIN := dango

.PHONY: build test run mutate

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN) ./cmd/dango

test:
	go test ./...

# Seedless, deterministic mutants only. No math/rand. No shuffled fixtures.
mutate:
	go test -count=1 -run Mutant ./...

run:
	go run ./cmd/dango
