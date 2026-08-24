BIN := dango

# Real mutator used by CI. Distinct from `mutate` (hand-written Mutant_* tests).
GREMLINS ?= gremlins

.PHONY: build dist test cover run mutate mutation

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN) ./cmd/dango

# Cross-compiled stripped executables for the nightly prerelease.
# linux/amd64 and darwin/arm64 only. Same flags as `build`. Files are the binaries.
dist:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/dango-linux-amd64 ./cmd/dango
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/dango-darwin-arm64 ./cmd/dango

test:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Seedless, deterministic Mutant_* tests only. Ordinary go test. Not a mutator.
mutate:
	go test -count=1 -run Mutant ./...

# Same gremlins invocation CI uses. Thresholds stay 0: report, do not gate.
# Scope: ./internal (app, cli, data, domain, live, summary, tui).
# testdata is not a Go package. cmd/dango has no tests — skipped.
mutation:
	$(GREMLINS) unleash --output=gremlins.json --threshold-efficacy=0 --threshold-mcover=0 ./internal

run:
	go run ./cmd/dango

