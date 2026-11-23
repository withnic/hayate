.PHONY: fmt fmt-check lint test check build

fmt:
	gofmt -s -w cmd internal

fmt-check:
	@files=$$(gofmt -s -l cmd internal) || exit 1; \
	if [ -n "$$files" ]; then printf '%s\n' "$$files" 'Run make fmt to fix formatting.'; exit 1; fi

lint:
	go vet ./...

test:
	CGO_ENABLED=1 go test -race ./...

check: fmt-check lint test

build:
	CGO_ENABLED=0 go build -trimpath -o bin/ ./cmd/hayate
