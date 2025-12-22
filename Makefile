GO := go

.PHONY: test test-cover fmt lint clean build check release-test release help

# Testing
test:
	@echo "Running tests..."
	$(GO) test ./...

test-cover:
	@echo "Running tests with coverage..."
	$(GO) test -cover ./...

# Code quality
fmt:
	@echo "Formatting code..."
	gofmt -s -w .

fmt-check:
	@echo "Checking code formatting..."
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "The following files are not formatted:"; \
		gofmt -s -l .; \
		exit 1; \
	fi

lint:
	@echo "Running linter..."
	$(GO) vet ./...

# Building
build:
	@echo "Building all packages..."
	$(GO) build ./...

# CI checks (same as GitHub Actions)
check: fmt-check lint test build
	@echo "All checks passed!"

# Release management
release-test:
	@echo "Testing release configuration..."
	goreleaser check
	goreleaser release --snapshot --clean

release:
	@echo "Creating release..."
	goreleaser release --clean

# Cleanup
clean:
	@echo "Cleaning up..."
	$(GO) clean ./...
	rm -f coverage.out
	rm -rf dist/
	@echo "Done!"

# Development
dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/goreleaser/goreleaser@latest

# Help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Testing:"
	@echo "  make test         - Run all tests"
	@echo "  make test-cover   - Run tests with coverage"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt          - Format all Go files"
	@echo "  make fmt-check    - Check if code is formatted"
	@echo "  make lint         - Run go vet linter"
	@echo ""
	@echo "Building:"
	@echo "  make build        - Build all packages"
	@echo ""
	@echo "CI/CD:"
	@echo "  make check        - Run all CI checks (fmt, lint, test, build)"
	@echo "  make release-test - Test release configuration"
	@echo "  make release      - Create actual release"
	@echo ""
	@echo "Development:"
	@echo "  make dev-deps     - Install development dependencies"
	@echo "  make clean        - Clean build artifacts"
