GO := /usr/local/go/bin/go

.PHONY: server client clean build

server:
	@echo "🚀 Starting event tracking server..."
	@$(GO) run cmd/server/main.go

client:
	@echo "🎯 Starting interactive client..."
	@$(GO) run cmd/client/*.go

build:
	@echo "🔨 Building binaries..."
	@cd cmd/client && $(GO) build -o ../../client .
	@cd cmd/server && $(GO) build -o ../../server .
	@echo "✅ Built: client, server"

clean:
	@echo "🧹 Cleaning up..."
	@rm -f ripple_events.json client server
	@echo "✨ Done!"

help:
	@echo "Available commands:"
	@echo "  make server  - Start the event tracking server"
	@echo "  make client  - Run the interactive client"
	@echo "  make build   - Build client and server binaries"
	@echo "  make clean   - Remove persisted events file and binaries"
