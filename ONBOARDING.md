# Ripple Go SDK - Complete Implementation Guide

## Recent Changes

### Adapter Naming Refactor
- Renamed `DefaultHTTPAdapter` to `NetHTTPAdapter` for better Go conventions
- Updated constructor: `NewDefaultHTTPAdapter()` → `NewNetHTTPAdapter()`

### Timer Behavior Enhancement
- Timer now only starts when first new event is tracked, not during SDK initialization
- If persisted events exist, they remain in queue until a new event triggers the timer
- Maintains same API while improving efficiency for apps with persisted events

### Graceful Shutdown Enhancement
- Added `StopWithoutFlush()` and `DisposeWithoutFlush()` methods for graceful shutdown without flushing events
- Fixed playground client exit behavior to persist events without sending to server

## Project Overview

Ripple Go is a high-performance, fault-tolerant event tracking SDK implemented as a single Go package. It provides reliable event delivery, batching, retries, persistence, and graceful shutdown for server-side applications.

This version is not a monorepo. It has no browser package, no Node.js package, and no internal modules exposed. All functionality exists within one cohesive Go module.

## SDK Features

### Core Features

* **Context Management** – shared context automatically attached to all events
* **Event Metadata** – optional schema versioning
* **Automatic Batching** – dispatch based on batch size
* **Scheduled Flushing** – time-based flush via goroutines
* **Retry Logic** – exponential backoff with jitter
* **Event Persistence** – disk-backed storage for unsent events
* **Queue Management** – FIFO queue using `container/list`
* **Graceful Shutdown** – flushes and persists all events on dispose
* **Adapters** – pluggable HTTP and storage implementations

### Go-Specific Features

* **Safe concurrency** (mutex-protected dispatcher and context)
* **Native HTTP client** (`net/http`)
* **File-based persistence** using JSON
* **Automatic boot-time recovery** from persisted events
* **Zero external dependencies**; uses only standard library

### Configuration

```go
type ClientConfig struct {
    APIKey        string
    Endpoint      string
    FlushInterval time.Duration // Default: 5s
    MaxBatchSize  int           // Default: 10
    MaxRetries    int           // Default: 3
}
```

### Developer Experience

* Simple, predictable API
* Explicit `error` returns
* Comprehensive tests for all components
* No external dependencies
* Practical examples included

---

## Architecture

### Project Structure

```sh
ripple-go/
├── client.go
├── client_test.go
├── dispatcher.go
├── dispatcher_test.go
├── queue.go
├── queue_test.go
├── types.go
├── types_test.go
├── go.mod
├── README.md
├── ONBOARDING.md
├── adapters/
│   ├── http_adapter.go
│   ├── http_adapter_test.go
│   ├── storage_adapter.go
│   ├── file_storage_adapter.go
│   ├── file_storage_adapter_test.go
│   ├── types.go
│   └── README.md
├── examples/
│   └── basic/
│       ├── go.mod
│       └── main.go
└── playground/
    ├── server.go
    ├── client.go
    ├── go.mod
    ├── Makefile
    └── README.md
```

### Components

#### Client

Entry point for the SDK.
Responsibilities:

* Initialization
* Managing global context
* Accepting new events
* Passing events to the dispatcher
* Exposing flushing and shutdown

Thread safety is enforced through internal locking.

Key methods:

* `Init()`
* `Track(name, payload, metadata)`
* `SetContext(key, value)`
* `GetContext()`
* `SetHTTPAdapter(adapter)` - Set custom HTTP adapter (before Init)
* `SetStorageAdapter(adapter)` - Set custom storage adapter (before Init)
* `Flush()`
* `Dispose()`

#### Context Manager

Provides thread-safe access to global context:

* Stored as `map[string]interface{}`
* Protected with `sync.RWMutex`
* Merged into every event at dispatch time

#### Dispatcher

Handles all operational concerns:

* Queueing
* Persistence
* Automatic and manual flushing
* Batch formation
* Retry with exponential backoff and jitter
* De-queuing and re-queuing failed events
* Loading persisted events on startup
* Graceful shutdown

A single mutex prevents concurrent flushes.

#### Queue

The queue is built on Go's `container/list` and wrapped in a small API:

* FIFO ordering
* O(1) enqueue/dequeue
* Thread-safe
* Slice conversion helpers for persistence

Wrapper methods include:

* `Enqueue(event)`
* `Dequeue()`
* `IsEmpty()`
* `Len()`
* `Clear()`
* `ToSlice()`
* `LoadFromSlice(events)`

#### HTTP Adapter

Interface defined in `adapters/http_adapter.go`:

```go
type HTTPAdapter interface {
    Send(endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error)
}
```

Default implementation (`NetHTTPAdapter`):

* Uses `net/http`
* JSON payloads
* Combined headers (default + user headers)

#### Storage Adapter

Interface defined in `adapters/storage_adapter.go`:

```go
type StorageAdapter interface {
    Save(events []Event) error
    Load() ([]Event, error)
    Clear() error
}
```

Default implementation (`FileStorageAdapter`):

* JSON file written to disk (`ripple_events.json`)
* Unlimited capacity
* Suitable for server environments

---

## Types

### EventMetadata

```go
type EventMetadata struct {
    SchemaVersion string `json:"schemaVersion,omitempty"`
}
```

### Platform

All events identify the runtime as server:

```go
type Platform struct {
    Type string `json:"type"` // "server"
}
```

### Event

```go
type Event struct {
    Name     string                 `json:"name"`
    Payload  map[string]interface{} `json:"payload,omitempty"`
    IssuedAt int64                  `json:"issuedAt"`
    Context  map[string]interface{} `json:"context,omitempty"`
    Metadata *EventMetadata         `json:"metadata,omitempty"`
    Platform *Platform              `json:"platform,omitempty"`
}
```

### DispatcherConfig

```go
type DispatcherConfig struct {
    Endpoint      string
    FlushInterval time.Duration
    MaxBatchSize  int
    MaxRetries    int
}
```

### HTTPResponse

```go
type HTTPResponse struct {
    OK     bool
    Status int
    Data   interface{}
}
```

---

## Usage Examples

### Basic Usage

```go
client := ripple.NewClient(ripple.ClientConfig{
    APIKey:   "your-api-key",
    Endpoint: "https://api.example.com/events",
})

if err := client.Init(); err != nil {
    panic(err)
}
defer client.Dispose()

client.SetContext("userId", "123")
client.SetContext("appVersion", "1.0.0")

client.Track("page_view", map[string]interface{}{
    "page": "/home",
}, nil)

client.Track("user_action", map[string]interface{}{
    "button": "submit",
}, &ripple.EventMetadata{SchemaVersion: "1.0.0"})

client.Flush()
```

### Using Metadata

```go
client.Track("user_signup", map[string]interface{}{
    "email": "user@example.com",
}, &ripple.EventMetadata{SchemaVersion: "1.0.0"})
```

### Custom HTTP Adapter

```go
import "github.com/Tap30/ripple-go/adapters"

type MyHTTPAdapter struct {}

func (a *MyHTTPAdapter) Send(endpoint string, events []adapters.Event, headers map[string]string) (*adapters.HTTPResponse, error) {
    // custom logic
    return &adapters.HTTPResponse{OK: true, Status: 200}, nil
}
```

### Custom Storage Adapter

```go
import "github.com/Tap30/ripple-go/adapters"

type RedisStorage struct {}

func (r *RedisStorage) Save(events []adapters.Event) error { /* ... */ return nil }
func (r *RedisStorage) Load() ([]adapters.Event, error)    { /* ... */ return nil, nil }
func (r *RedisStorage) Clear() error                       { /* ... */ return nil }
```

---

## Development Workflow

### Testing

The project includes test files for every component:

* `client_test.go`
* `dispatcher_test.go`
* `queue_test.go`
* `storage_adapter_test.go`
* `http_adapter_test.go`

### Commands

* `go build ./...` - Build all packages
* `go test ./...` - Run all tests
* `go test -v ./...` - Run tests with verbose output
* `go test -cover ./...` - Run tests with coverage
* `go vet ./...` - Run Go vet for static analysis

### Playground

The playground provides a local testing environment:

* `playground/server.go` - HTTP server that receives and logs events
* `playground/client.go` - Example client that sends events

**Usage:**
```bash
# Terminal 1: Start server
cd playground && make server

# Terminal 2: Run client
cd playground && make client
```

See [playground/README.md](./playground/README.md) for E2E testing scenarios.

### Recommendations

* Strong test coverage for dispatcher and queue logic
* Integration tests for persistence and HTTP transport
* Benchmarks for high-volume event throughput
* Linting via `golangci-lint`

---

## Design Principles

### Clear Responsibilities

* Client: API surface
* Dispatcher: internal mechanics
* Queue: data structure, thread-safe
* Adapters: extensibility

### Concurrency Safety

* Mutex around flush cycles
* RWMutex for context access
* Controlled goroutine lifecycle

### Reliability

* Persistent queueing
* Retried delivery with backoff
* Safe process shutdown

### Simplicity

* Single self-contained package
* No external dependencies
* Clean, predictable API

---

## Implementation Notes

### File Organization

Following Go best practices:
* All source files in root directory (no `src/` folder)
* Test files co-located with source (`*_test.go`)
* Adapters in separate `adapters/` package for modularity
* Examples in `examples/` subdirectory
* Single main package name: `ripple`
* Adapter interfaces and implementations in `adapters` package

### Concurrency Model

* Dispatcher runs a background goroutine for scheduled flushing
* All queue operations are mutex-protected
* Context reads use RWMutex for concurrent access
* Flush operations are serialized to prevent race conditions

### Error Handling

* All errors are returned explicitly
* No panics in library code
* Graceful degradation on network failures
* Failed events are re-queued and persisted

### Memory Management

* Events are stored in a linked list for efficient FIFO operations
* Batching prevents unbounded memory growth
* Persistence ensures events survive process restarts
* No memory leaks from goroutines (proper cleanup on Dispose)
