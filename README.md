<div align="center">

<img width="64" height="64" alt="Ripple Logo" src="https://raw.githubusercontent.com/Tap30/ripple/refs/heads/main/ripple-logo.png" />

# Ripple | Go

</div>

<div align="center">

A high-performance, scalable, and fault-tolerant event tracking Go SDK
for server-side applications.

</div>

<hr />

## Features

- **Zero Runtime Dependencies** – Built entirely with Go standard library
- **Thread-Safe** – Concurrent event tracking with mutex protection
- **Auto-Initialization** – `Track()` automatically calls `Init()` if not yet initialized
- **Disposal Tracking** – Disposed clients silently drop events; explicit `Init()` re-enables
- **Automatic Batching** – Efficient event grouping with dynamic rebatching for optimal network usage
- **One-Shot Timer** – Flush timer fires once per scheduling cycle, not on a repeating interval
- **Smart Retry Logic** – Intelligent retry behavior based on HTTP status codes:
  - **2xx (Success)**: Clear storage, no retry
  - **4xx (Client Error)**: Drop events, no retry (prevents infinite loops)
  - **5xx (Server Error)**: Retry with exponential backoff, re-queue on max retries
  - **Network Errors**: Retry with exponential backoff, re-queue on max retries
- **Retry Cancellation** – `Dispose()` aborts in-flight retries via context cancellation
- **Event Persistence** – Disk-backed storage for reliability
- **Pluggable Adapters** – Custom HTTP and storage implementations

## Installation

```bash
go get github.com/Tap30/ripple-go@latest
```

Or install a specific version:

```bash
go get github.com/Tap30/ripple-go@v0.0.1
```

## Quick Start

### Basic Usage

```go
package main

import (
    ripple "github.com/Tap30/ripple-go"
    "github.com/Tap30/ripple-go/adapters"
)

func main() {
    client, err := ripple.NewClient(ripple.ClientConfig{
        APIKey:         "your-api-key",
        Endpoint:       "https://api.example.com/events",
        HTTPAdapter:    adapters.NewNetHTTPAdapter(),
        StorageAdapter: adapters.NewFileStorageAdapter("ripple_events.json"),
    })
    if err != nil {
        panic(err)
    }
    defer client.Dispose()

    // Set global metadata
    client.SetMetadata("userId", "123")
    client.SetMetadata("appVersion", "1.0.0")

    // Track events (auto-initializes on first call)
    client.Track("page_view", map[string]any{
        "page": "/home",
    })

    // Track with event-specific metadata
    client.Track("user_action", map[string]any{
        "button": "submit",
    }, map[string]any{
        "schemaVersion": "1.0.0",
    })

    // Manually flush
    client.Flush()
}
```

## Configuration

```go
type ClientConfig struct {
    APIKey         string         // Required: API authentication key
    Endpoint       string         // Required: Event collection endpoint
    APIKeyHeader   *string        // Optional: Header name for API key (default: "X-API-Key")
    FlushInterval  time.Duration  // Optional: Default 5s
    MaxBatchSize   int            // Optional: Default 10
    MaxRetries     int            // Optional: Default 3
    MaxBufferSize  int            // Optional: Max events in storage (0 = unlimited)
    HTTPAdapter    HTTPAdapter    // Required: Custom HTTP adapter
    StorageAdapter StorageAdapter // Required: Custom storage adapter
    LoggerAdapter  LoggerAdapter  // Optional: Custom logger adapter
}
```

Configuration validation:
- `FlushInterval` must be positive if provided
- `MaxBatchSize` must be positive if provided
- `MaxRetries` must be non-negative if provided
- `MaxBufferSize` must be positive if provided, and >= `MaxBatchSize`

### Understanding `MaxBatchSize` vs `MaxBufferSize`

**`MaxBatchSize` (default: 10)** - Controls **when** events are sent

- Triggers immediate flush when queue reaches this size
- Determines how many events are sent in each HTTP request

**`MaxBufferSize` (default: 0 = unlimited)** - Controls **how many** events are stored

- Limits total events persisted to storage
- When limit is reached, oldest events are dropped (FIFO eviction)
- Must be >= `MaxBatchSize` (returns error otherwise)

## API

### Client Methods

#### `Init() error`

Initializes the client and restores persisted events. Uses double-checked locking for thread safety. Resets the disposed state, so calling `Init()` after `Dispose()` re-enables the client.

Note: `Track()` automatically calls `Init()`, so explicit initialization is optional.

#### `Track(name string, args ...any) error`

Tracks an event with optional payload and metadata. Supports three usage patterns:

- `Track(name)` - Simple event tracking
- `Track(name, payload)` - Event with payload
- `Track(name, payload, metadata)` - Event with payload and metadata

If the client is disposed, events are silently dropped (returns nil). Otherwise, auto-calls `Init()` if not yet initialized.

#### `SetMetadata(key string, value any)`

Sets a metadata value that will be attached to all subsequent events.

#### `GetMetadata() map[string]any`

Returns a copy of all stored metadata. Returns empty map if no metadata is set.

#### `GetSessionId() *string`

Returns `nil` for server environments.

#### `Flush()`

Manually triggers a flush of all queued events.

#### `Dispose()`

Cleans up resources: aborts in-flight retries, clears queue, clears metadata, resets state. Does NOT flush events. Call `Flush()` before `Dispose()` if you want to send remaining events.

#### `Close()`

Alias for `Dispose()`.

## Advanced Usage

### Custom HTTP Adapter

Implement the `HTTPAdapter` interface to use custom HTTP clients:

```go
import (
    "context"
    "github.com/Tap30/ripple-go/adapters"
)

type MyHTTPAdapter struct{}

func (a *MyHTTPAdapter) Send(endpoint string, events []adapters.Event, headers map[string]string, apiKeyHeader string) (*adapters.HTTPResponse, error) {
    return a.SendWithContext(context.Background(), endpoint, events, headers, apiKeyHeader)
}

func (a *MyHTTPAdapter) SendWithContext(ctx context.Context, endpoint string, events []adapters.Event, headers map[string]string, apiKeyHeader string) (*adapters.HTTPResponse, error) {
    // custom HTTP logic
    return &adapters.HTTPResponse{Status: 200}, nil
}
```

### Custom Storage Adapter

```go
import "github.com/Tap30/ripple-go/adapters"

type RedisStorage struct{}

func (r *RedisStorage) Save(events []adapters.Event) error { return nil }
func (r *RedisStorage) Load() ([]adapters.Event, error)    { return nil, nil }
func (r *RedisStorage) Clear() error                       { return nil }
```

### Graceful Shutdown

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

go func() {
    <-sigChan
    client.Flush()
    client.Dispose()
    os.Exit(0)
}()
```

## Logger Adapters

| Adapter                  | Output  | Configurable | Use Case                    |
| ------------------------ | ------- | ------------ | --------------------------- |
| **PrintLoggerAdapter**   | Stdout  | Yes          | Development and debugging   |
| **NoOpLoggerAdapter**    | None    | No           | Production (silent logging) |

### Log Levels

- `DEBUG`: Detailed debugging information
- `INFO`: General information messages
- `WARN`: Warning messages (default level)
- `ERROR`: Error messages
- `NONE`: No logging output

```go
import "github.com/Tap30/ripple-go/adapters"

client, err := ripple.NewClient(ripple.ClientConfig{
    // ... other config
    LoggerAdapter: adapters.NewPrintLoggerAdapter(adapters.LogLevelDebug),
})
```

## Storage Adapters

| Adapter                | Capacity  | Persistence | Use Case                          |
| ---------------------- | --------- | ----------- | --------------------------------- |
| **FileStorageAdapter** | Unlimited | Permanent   | Default, persistent event storage |
| **NoOpStorageAdapter** | N/A       | None        | When persistence is not needed    |

```go
import "github.com/Tap30/ripple-go/adapters"

// Persistent storage
fileStorage := adapters.NewFileStorageAdapter("ripple_events.json")

// No persistence
noopStorage := adapters.NewNoOpStorageAdapter()
```

## Concurrency Guarantees

- **Thread-Safe Flush**: Multiple concurrent `Flush()` calls are serialized via mutex
- **Thread-Safe Init**: Double-checked locking prevents race conditions during auto-init
- **Event Ordering**: FIFO order is maintained even during retry failures
- **No Event Loss**: Events tracked during flush are queued for the next batch

## Error Handling

- **2xx Success**: Events cleared from storage
- **4xx Client Errors**: Events dropped (no retry)
- **5xx Server Errors**: Retried with exponential backoff (30s cap), re-queued on max retries
- **Network Errors**: Same as 5xx

## Architecture

- **Client** – Public API, metadata management, disposal tracking
- **Dispatcher** – Event batching, one-shot timer flushing, retry with context cancellation
- **Queue** – Thread-safe FIFO event queue
- **MetadataManager** – Thread-safe shared metadata
- **Adapters** – Pluggable HTTP, storage, and logger implementations

See [AGENTS.md](./AGENTS.md) for detailed architecture documentation.

## API Contract

See the [API Contract Documentation](https://github.com/Tap30/ripple/blob/main/DESIGN_AND_CONTRACTS.md) for the shared interface all Ripple SDKs follow.

## Development

```bash
make test         # Run all tests
make test-cover   # Run tests with coverage
make fmt          # Format code
make lint         # Run linter
make build        # Build all packages
make check        # Run all CI checks
```

### Playground

```bash
# Terminal 1: Start server
cd playground && make server

# Terminal 2: Run client
cd playground && make client
```

## Contributing

See the [contributing guide](https://github.com/Tap30/ripple-go/blob/main/CONTRIBUTING.md).

Uses [Conventional Commits](https://www.conventionalcommits.org/) and automated semantic versioning.

## License

Distributed under the [MIT license](https://github.com/Tap30/ripple-go/blob/main/LICENSE).
