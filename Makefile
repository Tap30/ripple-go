GO := /usr/local/go/bin/go

.PHONY: test fmt lint clean help

test:
	@echo "Running tests..."
	$(GO) test ./...

test-cover:
	@echo "Running tests with coverage..."
	$(GO) test -cover ./...

fmt:
	@echo "Formatting code..."
	/usr/local/go/bin/gofmt -w .

lint:
	@echo "Running linter..."
	$(GO) vet ./...

clean:
	@echo "Cleaning up..."
	$(GO) clean ./...
	rm -f coverage.out
	@echo "Done!"

help:
	@echo "Available commands:"
	@echo "  make test       - Run all tests"
	@echo "  make test-cover - Run tests with coverage"
	@echo "  make fmt        - Format all Go files"
	@echo "  make lint       - Run go vet linter"
	@echo "  make clean      - Clean build artifacts"
