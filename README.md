<div align="center">

<img width="64" height="64" alt="Ripple Logo" src="https://raw.githubusercontent.com/Tap30/ripple/refs/heads/main/ripple-logo.png" />

# Ripple | Go

</div>

<div align="center">

A high-performance, scalable, and fault-tolerant event tracking TypeScript SDK
for browsers.

</div>

<hr />

## Features

- **Zero Runtime Dependencies** – Built entirely with Go standard library
- **Thread-Safe** – Concurrent event tracking with mutex protection
- **Automatic Batching** – Efficient event grouping with dynamic rebatching for optimal network usage
- **Smart Retry Logic** – Intelligent retry behavior based on HTTP status codes:
  - **2xx (Success)**: Clear storage, no retry
  - **4xx (Client Error)**: Drop events, no retry (prevents infinite loops)
  - **5xx (Server Error)**: Retry with exponential backoff, re-queue on max retries
  - **Network Errors**: Retry with exponential backoff, re-queue on max retries
- **Event Persistence** – Disk-backed storage for reliability
- **Graceful Shutdown** – Ensures all events are flushed and persisted
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
    "time"
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

    // Or use NoOpStorageAdapter if persistence is not needed
    client, err := ripple.NewClient(ripple.ClientConfig{
        APIKey:         "your-api-key",
        Endpoint:       "https://api.example.com/events",
        HTTPAdapter:    adapters.NewNetHTTPAdapter(),
        StorageAdapter: adapters.NewNoOpStorageAdapter(),
    })
    if err != nil {
        panic(err)
    }

    if err := client.Init(); err != nil {
        panic(err)
    }
    defer client.Dispose()

    // Set global metadata
    if err := client.SetMetadata("userId", "123"); err != nil {
        panic(err)
    }
    if err := client.SetMetadata("appVersion", "1.0.0"); err != nil {
        panic(err)
    }

    // Track events
    if err := client.Track("page_view", map[string]any{
        "page": "/home",
    }); err != nil {
        panic(err)
    }

    // Track with metadata
    if err := client.Track("user_action", map[string]any{
        "button": "submit",
    }, map[string]any{
        "schemaVersion": "1.0.0",
    }); err != nil {
        panic(err)
    }

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

### Understanding `MaxBatchSize` vs `MaxBufferSize`

These two parameters serve different purposes and work together:

**`MaxBatchSize` (default: 10)** - Controls **when** events are sent

- Triggers immediate flush when queue reaches this size
- Determines how many events are sent in each HTTP request
- Smaller values = more frequent network requests
- Larger values = fewer requests but higher latency

**`MaxBufferSize` (default: 0 = unlimited)** - Controls **how many** events are stored

- Limits total events persisted to storage
- When limit is reached, oldest events are dropped (FIFO eviction)
- Protects against unbounded storage growth
- Useful for:
  - Preventing disk space issues during extended offline periods
  - Controlling memory usage in high-throughput scenarios
  - Limiting event retention for privacy/compliance

**Important**: `MaxBufferSize` should always be **greater than or equal to** `MaxBatchSize`. If `MaxBufferSize` is smaller, the batch size will never be reached and events will be dropped unnecessarily.

**Examples:**

```go
// ✅ Good: Buffer is 10x batch size
client, err := ripple.NewClient(ripple.ClientConfig{
    MaxBatchSize:  10,
    MaxBufferSize: 100,
    // ...
})

// ✅ Good: Large buffer for extended offline periods
client, err := ripple.NewClient(ripple.ClientConfig{
    MaxBatchSize:  20,
    MaxBufferSize: 1000,
    // ...
})

// ❌ Bad: Buffer smaller than batch (batch will never be reached)
client, err := ripple.NewClient(ripple.ClientConfig{
    MaxBatchSize:  100,
    MaxBufferSize: 50, // Events dropped before batch is full!
    // ...
})
```

**Behavior in different scenarios**:

- **Normal operation** (`MaxBatchSize: 10, MaxBufferSize: 100`): Events flush every 10 events, buffer rarely fills
- **Offline mode**: Buffer accumulates up to 100 events, then starts dropping oldest
- **Misconfigured** (`MaxBatchSize: 100, MaxBufferSize: 50`): Batch never reached, events only sent via time-based flush

## API

### Client Methods

#### `Init() error`

Initializes the client and starts the dispatcher. Must be called before tracking events.

#### `Track(name string, args ...any) error`

Tracks an event with optional payload and metadata. Supports three usage patterns:

- `Track(name)` - Simple event tracking
- `Track(name, payload)` - Event with payload
- `Track(name, payload, metadata)` - Event with payload and metadata

Returns error if event name is empty, exceeds 255 characters, or if client is not initialized.

#### `SetMetadata(key string, value any) error`

Sets a metadata value that will be attached to all subsequent events. Returns error if key is empty or exceeds 255 characters.

#### `GetMetadata() map[string]any`

Returns a copy of all stored metadata. Returns empty map if no metadata is set.

#### `GetSessionId() *string`

Returns the current session ID or `nil` if not set. Always returns `nil` for server environments.

#### `Flush()`

Manually triggers a flush of all queued events.

#### `Dispose() error`

Gracefully shuts down the client, flushing and persisting all events.

## Advanced Usage

### Custom HTTP Adapter

Implement the `HTTPAdapter` interface to use custom HTTP clients:

```go
import (
    ripple "github.com/Tap30/ripple-go"
    "github.com/Tap30/ripple-go/adapters"
)

type MyHTTPAdapter struct {
    // custom fields
}

func (a *MyHTTPAdapter) Send(endpoint string, events []adapters.Event, headers map[string]string) (*adapters.HTTPResponse, error) {
    // custom HTTP logic
    return &adapters.HTTPResponse{Status: 200}, nil
}

// Use custom adapter
client, err := ripple.NewClient(ripple.ClientConfig{
    APIKey:         "your-api-key",
    Endpoint:       "https://api.example.com/events",
    HTTPAdapter:    &MyHTTPAdapter{},
    StorageAdapter: adapters.NewFileStorageAdapter("ripple_events.json"),
})
if err != nil {
    panic(err)
}
client.Init()
```

### Custom Storage Adapter

Implement the `StorageAdapter` interface to use custom storage backends:

```go
import (
    ripple "github.com/Tap30/ripple-go"
    "github.com/Tap30/ripple-go/adapters"
)

type RedisStorage struct {
    // Redis client
}

func (r *RedisStorage) Save(events []adapters.Event) error {
    // Save to Redis
    return nil
}

func (r *RedisStorage) Load() ([]adapters.Event, error) {
    // Load from Redis
    return nil, nil
}

func (r *RedisStorage) Clear() error {
    // Clear Redis storage
    return nil
}

// Use custom adapter
client, err := ripple.NewClient(ripple.ClientConfig{
    APIKey:         "your-api-key",
    Endpoint:       "https://api.example.com/events",
    HTTPAdapter:    adapters.NewNetHTTPAdapter(),
    StorageAdapter: &RedisStorage{},
})
if err != nil {
    panic(err)
}
client.Init()
```

### Graceful Shutdown

```go
import (
    "os"
    "os/signal"
    "syscall"
    ripple "github.com/Tap30/ripple-go"
)

client, err := ripple.NewClient(config)
if err != nil {
    panic(err)
}
client.Init()

// Handle graceful shutdown
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

// Persistent storage (default)
fileStorage := adapters.NewFileStorageAdapter("ripple_events.json")

// No persistence - events are discarded if not sent
noopStorage := adapters.NewNoOpStorageAdapter()
```

By default, events are persisted to `ripple_events.json` in the current working directory.

## Concurrency Guarantees

The SDK is designed to handle concurrent operations safely:

- **Thread-Safe Flush**: Multiple concurrent `Flush()` calls are automatically serialized using mutex locks
- **Event Ordering**: FIFO order is maintained even during retry failures and concurrent operations
- **No Event Loss**: Events tracked during flush operations are safely queued and sent in the next batch
- **Automatic Cleanup**: Mutex locks are automatically released even if errors occur, preventing deadlocks

You can safely call `Track()` and `Flush()` from multiple goroutines without worrying about race conditions.

## Error Handling

The SDK handles HTTP errors differently based on their type:

- **2xx Success**: Events are cleared from storage
- **4xx Client Errors**: Events are dropped (not retried or persisted) since client errors won't resolve without code changes
- **5xx Server Errors**: Retried with exponential backoff up to `MaxRetries`, then re-queued and persisted for later retry
- **Network Errors**: Same behavior as 5xx errors

## Architecture

The SDK consists of several key components:

- **Client** – Public API and metadata management
- **Dispatcher** – Event batching, flushing, and retry logic
- **Queue** – Thread-safe FIFO event queue
- **MetadataManager** – Shared metadata management
- **HTTP Adapter** – Network communication layer
- **Storage Adapter** – Event persistence layer
- **Logger Adapter** – Logging interface

See [AGENTS.md](./AGENTS.md) for detailed architecture documentation.

## API Contract

See the  
[API Contract Documentation](https://github.com/Tap30/ripple/blob/main/DESIGN_AND_CONTRACTS.md)  
for details on the shared, framework-independent interface all Ripple SDKs follow.

## Development

### Running Tests

```bash
go test ./...
```

### Running Tests with Coverage

```bash
go test -cover ./...
```

### Running Examples

```bash
cd examples/basic
go run main.go
```

### Playground

Test the SDK with a local server:

```bash
# Terminal 1: Start server
cd playground
make server

# Terminal 2: Run client
cd playground
make client
```

See [playground/README.md](./playground/README.md) for more details.

## Contributing

Check the  
[contributing guide](https://github.com/Tap30/ripple-go/blob/main/CONTRIBUTING.md)  
for information on development workflow, proposing improvements, and running tests.

**Note**: This project uses [Conventional Commits](https://www.conventionalcommits.org/) and automated semantic versioning with [go-semantic-release](https://github.com/go-semantic-release/semantic-release). Use proper commit message formats like `feat:`, `fix:`, `docs:`, etc.

## License

Distributed under the  
[MIT license](https://github.com/Tap30/ripple-go/blob/main/packages/browser/LICENSE).
