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
check: fmt-check lint test build version-check
	@echo "All checks passed!"

# Release management
version-sync:
	@echo "Syncing version between .versionrc and version.go..."
	./scripts/sync-version.sh

version-check:
	@echo "Checking version consistency..."
	@VERSION_RC=$$(cat .versionrc | tr -d '[:space:]'); \
	VERSION_GO=$$(grep 'const Version' version.go | sed 's/.*"\(.*\)".*/\1/'); \
	if [ "$$VERSION_RC" != "$$VERSION_GO" ]; then \
		echo "❌ Version mismatch:"; \
		echo "  .versionrc: $$VERSION_RC"; \
		echo "  version.go: $$VERSION_GO"; \
		echo "Run 'make version-sync' to fix"; \
		exit 1; \
	else \
		echo "✅ Version consistent: $$VERSION_RC"; \
	fi

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
	@echo "Release Management:"
	@echo "  make version-sync   - Sync version from .versionrc to version.go"
	@echo "  make version-check  - Check version consistency"
	@echo "  make release-test   - Test release configuration"
	@echo "  make release        - Create actual release"
	@echo ""
	@echo "CI/CD:"
	@echo "  make check        - Run all CI checks (fmt, lint, test, build, version)"
	@echo ""
	@echo "Development:"
	@echo "  make dev-deps     - Install development dependencies"
	@echo "  make clean        - Clean build artifacts"
