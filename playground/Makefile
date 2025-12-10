.PHONY: server client clean

server:
	@echo "🚀 Starting event tracking server..."
	@go run server.go

client:
	@echo "🎯 Starting interactive client..."
	@go run client.go

clean:
	@echo "🧹 Cleaning up..."
	@rm -f ripple_events.json
	@echo "✨ Done!"

help:
	@echo "Available commands:"
	@echo "  make server  - Start the event tracking server"
	@echo "  make client  - Run the interactive client"
	@echo "  make clean   - Remove persisted events file"
