<div align="center">

# Ripple | Go

</div>

<div align="center">

A fast, resilient, and scalable event-tracking SDK built in Go.

</div>

<hr />

## Features

- **Zero Dependencies** – Built entirely with Go standard library
- **Thread-Safe** – Concurrent event tracking with mutex protection
- **Automatic Batching** – Efficient event grouping for network optimization
- **Retry Logic** – Exponential backoff with jitter for failed requests
- **Event Persistence** – Disk-backed storage for reliability
- **Graceful Shutdown** – Ensures all events are flushed and persisted
- **Pluggable Adapters** – Custom HTTP and storage implementations

## Installation

```bash
go get github.com/Tap30/ripple-go
```

## Quick Start

```go
package main

import (
    "time"
    ripple "github.com/Tap30/ripple-go"
)

func main() {
    client := ripple.NewClient(ripple.ClientConfig{
        APIKey:   "your-api-key",
        Endpoint: "https://api.example.com/events",
    })

    if err := client.Init(); err != nil {
        panic(err)
    }
    defer client.Dispose()

    // Set global context
    client.SetContext("userId", "123")
    client.SetContext("appVersion", "1.0.0")

    // Track events
    client.Track("page_view", map[string]interface{}{
        "page": "/home",
    }, nil)

    // Track with metadata
    client.Track("user_action", map[string]interface{}{
        "button": "submit",
    }, &ripple.EventMetadata{
        SchemaVersion: "1.0.0",
    })

    // Manually flush
    client.Flush()
}
```

## Configuration

```go
type ClientConfig struct {
    APIKey        string        // Required: API authentication key
    Endpoint      string        // Required: Event collection endpoint
    FlushInterval time.Duration // Optional: Default 5s
    MaxBatchSize  int           // Optional: Default 10
    MaxRetries    int           // Optional: Default 3
}
```

## API

### Client Methods

#### `Init() error`
Initializes the client and starts the dispatcher. Must be called before tracking events.

#### `Track(name string, payload map[string]interface{}, metadata *EventMetadata)`
Tracks an event with optional payload and metadata.

#### `SetContext(key string, value interface{})`
Sets a global context value that will be attached to all events.

#### `GetContext() map[string]interface{}`
Returns a copy of the current global context.

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
client := ripple.NewClient(config)
client.SetHTTPAdapter(&MyHTTPAdapter{})
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
client := ripple.NewClient(config)
client.SetStorageAdapter(&RedisStorage{})
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
[API Contract Documentation](https://github.com/Tap30/ripple/blob/main/API_CONTRACT.md)  
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
[contributing guide](https://github.com/Tap30/ripple-ts/blob/main/CONTRIBUTING.md)  
for information on development workflow, proposing improvements, and running tests.

## License

Distributed under the  
[MIT license](https://github.com/Tap30/ripple-ts/blob/main/packages/browser/LICENSE).
