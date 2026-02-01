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

- **Zero Dependencies** – Built entirely with Go standard library
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
    if err := client.Track("page_view", map[string]interface{}{
        "page": "/home",
    }); err != nil {
        panic(err)
    }

    // Track with metadata
    if err := client.Track("user_action", map[string]interface{}{
        "button": "submit",
    }, map[string]interface{}{
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
    FlushInterval  time.Duration  // Optional: Default 5s
    MaxBatchSize   int            // Optional: Default 10
    MaxRetries     int            // Optional: Default 3
    HTTPAdapter    HTTPAdapter    // Required: Custom HTTP adapter
    StorageAdapter StorageAdapter // Required: Custom storage adapter
    LoggerAdapter  LoggerAdapter  // Optional: Custom logger adapter
}
```

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

#### `SetMetadata(key string, value interface{}) error`

Sets a metadata value that will be attached to all subsequent events. Returns error if key is empty or exceeds 255 characters.

#### `GetMetadata() map[string]interface{}`

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
    return &adapters.HTTPResponse{OK: true, Status: 200}, nil
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
client, err := ripple.NewClient[map[string]any, map[string]any](ripple.ClientConfig{
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

## Architecture

The SDK consists of several key components:

- **Client** – Public API and context management
- **Dispatcher** – Event batching, flushing, and retry logic
- **Queue** – Thread-safe FIFO event queue
- **HTTP Adapter** – Network communication layer
- **Storage Adapter** – Event persistence layer

See [ONBOARDING.md](./ONBOARDING.md) for detailed architecture documentation.

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
