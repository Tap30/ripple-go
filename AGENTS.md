# Ripple Go SDK - Complete Implementation Guide

## Recent Changes

### TypeScript SDK Sync (Latest)

Major refactor to align Go SDK behavior with the TypeScript SDK (source of truth):

- **Auto-initialization** – `Track()` automatically calls `Init()` if not yet initialized
- **Disposal tracking** – Client tracks `disposed` state; disposed clients silently drop events
- **Re-initialization** – Explicit `Init()` after `Dispose()` re-enables the client
- **Double-checked locking** – `Init()` uses mutex with double-check for thread-safe auto-init
- **Dispose clears state** – `Dispose()` clears metadata, resets initialized/disposed flags
- **Retry cancellation** – `Dispose()` aborts in-flight retries via context cancellation
- **One-shot timer** – Replaced repeating `time.Ticker` with one-shot `time.AfterFunc`
- **Backoff cap 30s** – Changed from 60s to 30s to match TypeScript
- **Restore logs errors** – `Restore()` no longer returns errors; logs and continues
- **Dispatcher in constructor** – Dispatcher created in `NewClient()`, not `Init()`
- **Headers built internally** – Dispatcher builds its own headers from config
- **Config validation** – Rejects negative FlushInterval, MaxBatchSize, MaxRetries, MaxBufferSize
- **Buffer < batch = error** – `MaxBufferSize < MaxBatchSize` now returns error (was warning)
- **Removed methods** – `SetHTTPAdapter()`, `SetStorageAdapter()`, `CloseWithoutFlush()`, `DisposeWithoutFlush()`, `Stop()`, `StopWithoutFlush()`
- **Removed validations** – Event name length, metadata key length, payload/metadata/event size limits, JSON serializability checks
- **SetMetadata simplified** – No longer returns error (matches TypeScript)
- **Dispose/Close simplified** – No longer returns error (matches TypeScript)
- **Enqueue calls Flush directly** – No goroutine spawn for batch-triggered flush

### Max Buffer Size Feature

- **Added MaxBufferSize configuration** - Limits the number of events persisted to storage
- **FIFO eviction policy** - When limit is exceeded, oldest events are dropped
- **Applied on enqueue and load** - Limit enforced when adding events and loading from storage
- **Configuration validation** - Error if MaxBufferSize < MaxBatchSize
- **Matches TypeScript SDK** - Consistent behavior across all Ripple SDKs

### Dynamic Rebatching in Flush

- **Updated Flush logic** - Now processes all queued events in optimal batches
- **Improved efficiency** - Clears entire queue at once, then processes in batches
- **Matches TypeScript SDK** - Consistent behavior across all Ripple SDKs

### Smart Retry Logic Update

- **Updated retry behavior** - Now follows intelligent status code-based retry logic
- **4xx Client Errors** - No retry, events are dropped (prevents infinite loops)
- **5xx Server Errors** - Retry with exponential backoff, re-queue on max retries
- **Network Errors** - Retry with exponential backoff, re-queue on max retries
- **2xx Success** - Clear storage, no retry needed

## Project Overview

Ripple Go is a high-performance, scalable, and fault-tolerant event tracking SDK implemented as a single Go package. It provides reliable event delivery, batching, retries, persistence, and graceful shutdown for server-side applications.

This version is not a monorepo. It has no browser package, no Node.js package, and no internal modules exposed. All functionality exists within one cohesive Go module that follows the unified API contract defined in the main Ripple repository.

## SDK Features

### Core Features

- **Auto-Initialization** – `Track()` automatically calls `Init()` if not yet initialized
- **Disposal Tracking** – Disposed clients silently drop events; explicit `Init()` re-enables
- **Unified Metadata System** – Single metadata field that merges shared metadata (client-level) with event-specific metadata
- **Thread-Safe Metadata Management** – MetadataManager with `sync.RWMutex`
- **Logger Interface** – Pluggable logging with PrintLoggerAdapter and NoOpLoggerAdapter
- **Automatic Batching** – dispatch based on batch size
- **One-Shot Timer Flushing** – time-based flush using `time.AfterFunc` (fires once, re-scheduled on next enqueue)
- **Smart Retry Logic** – Intelligent retry behavior based on HTTP status codes:
  - **2xx (Success)**: Clear storage, no retry
  - **4xx (Client Error)**: Drop events, no retry (prevents infinite loops)
  - **5xx (Server Error)**: Retry with exponential backoff (30s cap), re-queue on max retries
  - **Network Errors**: Retry with exponential backoff (30s cap), re-queue on max retries
- **Retry Cancellation** – `Dispose()` aborts in-flight retries via context cancellation
- **StorageQuotaExceededError** – Storage errors of this type are logged as warnings instead of errors
- **Event Persistence** – disk-backed storage for unsent events
- **Queue Management** – FIFO queue using `container/list`
- **Double-Checked Locking** – Thread-safe initialization with mutex
- **Adapters** – pluggable HTTP, storage, and logger implementations

### Go-Specific Features

- **Safe concurrency** (mutex-protected dispatcher and metadata)
- **Native HTTP client** (`net/http`)
- **File-based persistence** using JSON
- **Automatic boot-time recovery** from persisted events
- **Zero external dependencies**; uses only standard library

### Configuration

```go
type ClientConfig struct {
    APIKey         string
    Endpoint       string
    APIKeyHeader   *string        // Optional: Header name for API key (default: "X-API-Key")
    FlushInterval  time.Duration  // Default: 5s
    MaxBatchSize   int            // Default: 10
    MaxRetries     int            // Default: 3
    MaxBufferSize  int            // Optional: Max events in storage (0 = unlimited)
    HTTPAdapter    HTTPAdapter    // Required: Custom HTTP adapter
    StorageAdapter StorageAdapter // Required: Custom storage adapter
    LoggerAdapter  LoggerAdapter  // Optional: Custom logger adapter (default: PrintLoggerAdapter with WARN level)
}
```

---

## Architecture

### Project Structure

```sh
ripple-go/
├── ripple_client.go            # Main client: NewClient, Init, Track, Dispose, SetMetadata
├── ripple_client_test.go       # Client tests
├── dispatcher.go               # Event batching, retry logic, one-shot timer, context cancellation
├── dispatcher_test.go          # Dispatcher tests
├── queue.go                    # FIFO queue implementation
├── queue_test.go               # Queue tests
├── metadata_manager.go         # Shared metadata management
├── mutex.go                    # Mutex with RunAtomic and Release
├── types.go                    # Type definitions and re-exports
├── types_test.go               # Type tests
├── go.mod                      # Go module definition
├── Makefile                    # Build commands (test, fmt, lint, clean)
├── README.md                   # Project documentation
├── AGENTS.md                   # This file - complete implementation guide
├── adapters/
│   ├── http_adapter.go         # HTTP adapter interface
│   ├── net_http_adapter.go     # Default HTTP implementation
│   ├── net_http_adapter_test.go
│   ├── storage_adapter.go      # Storage adapter interface
│   ├── file_storage_adapter.go # Default file storage implementation
│   ├── file_storage_adapter_test.go
│   ├── logger_adapter.go       # Logger adapter interface
│   ├── print_logger_adapter.go
│   ├── noop_logger_adapter.go
│   ├── types.go               # Adapter type definitions
│   └── README.md
└── playground/
    ├── cmd/
    │   ├── client/
    │   │   └── main.go
    │   └── server/
    │       └── main.go
    ├── go.mod
    └── Makefile
```

### Core Components

#### Client

Entry point for the SDK with auto-initialization and disposal tracking.
Responsibilities:

* Configuration validation (required APIKey, Endpoint, adapters; numeric validation)
* Auto-initialization via `Track()` with double-checked locking
* Disposal tracking — disposed clients silently drop events
* Re-initialization — explicit `Init()` after `Dispose()` re-enables
* Managing shared metadata through MetadataManager
* Passing events to the dispatcher

Key methods:

* `Init()` - Initialize client and restore persisted events (auto-called by Track)
* `Track(name, ...args)` - Track event with optional payload and metadata
* `SetMetadata(key, value)` - Set shared metadata attached to all events
* `GetMetadata()` - Get all shared metadata as map
* `GetSessionId()` - Returns nil for server environments
* `Flush()` - Force flush queued events
* `Dispose()` - Clean up resources (aborts retries, clears queue/metadata)
* `Close()` - Alias for Dispose

#### MetadataManager

Manages global metadata attached to all events with thread-safe operations.

Key methods:

* `Set(key, value)` - Set metadata value
* `GetAll()` - Get all metadata (returns empty map if none)
* `IsEmpty()` - Check if metadata is empty
* `Clear()` - Remove all metadata

#### Mutex

Provides mutual exclusion lock for preventing race conditions.

Key methods:

* `RunAtomic(task func() error)` - Execute task with exclusive lock
* `Release()` - Forcefully release mutex (used during disposal)

#### Dispatcher

Handles all operational concerns:

* Event queueing with disposed check
* Persistence with error handling
* **Dynamic rebatching** - Flush processes all events in optimal batches
* **One-shot timer** - `time.AfterFunc` fires once, re-scheduled on next enqueue
* **Context cancellation** - `Dispose()` aborts in-flight retries
* **Smart retry logic** based on HTTP status codes
* Loading persisted events on startup (errors logged, not propagated)
* Headers built internally from config (APIKey + APIKeyHeader)

Key methods:

* `Enqueue(event)` - Add event to queue (rejects if disposed)
* `Flush()` - Send queued events (atomic operation)
* `Restore()` - Load persisted events, reset disposed flag
* `Dispose()` - Abort retries, clear queue, release mutex

#### Queue

FIFO queue built on Go's `container/list`:

* O(1) enqueue/dequeue
* Thread-safe
* Slice conversion helpers for persistence

### Adapter Interfaces

#### HTTP Adapter

```go
type HTTPAdapter interface {
    Send(endpoint string, events []Event, headers map[string]string, apiKeyHeader string) (*HTTPResponse, error)
    SendWithContext(ctx context.Context, endpoint string, events []Event, headers map[string]string, apiKeyHeader string) (*HTTPResponse, error)
}
```

#### Storage Adapter

```go
type StorageAdapter interface {
    Save(events []Event) error
    Load() ([]Event, error)
    Clear() error
}
```

#### Logger Adapter

```go
type LoggerAdapter interface {
    Debug(message string, args ...any)
    Info(message string, args ...any)
    Warn(message string, args ...any)
    Error(message string, args ...any)
}
```

---

## Types

### Event

```go
type Event struct {
    Name      string         `json:"name"`
    Payload   map[string]any `json:"payload"`
    Metadata  map[string]any `json:"metadata"`
    IssuedAt  int64          `json:"issuedAt"`
    SessionID *string        `json:"sessionId"`
    Platform  *Platform      `json:"platform"`
}
```

### Platform

```go
type Platform struct {
    Type string `json:"type"` // "server"
}
```

### StorageQuotaExceededError

```go
type StorageQuotaExceededError struct {
    Message string
}
```

Storage adapters should return this error when they cannot save events due to quota limits. The dispatcher logs it as a warning instead of an error.

### DispatcherConfig

```go
type DispatcherConfig struct {
    APIKey        string
    APIKeyHeader  string
    Endpoint      string
    FlushInterval time.Duration
    MaxBatchSize  int
    MaxRetries    int
    MaxBufferSize int
}
```

---

## Usage Examples

### Basic Usage

```go
import (
    ripple "github.com/Tap30/ripple-go"
    "github.com/Tap30/ripple-go/adapters"
)

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

// Set shared metadata
client.SetMetadata("userId", "123")

// Track events (auto-initializes on first call)
client.Track("page_view", map[string]any{"page": "/home"})

// Track with event-specific metadata
client.Track("user_action", map[string]any{
    "button": "submit",
}, map[string]any{"schemaVersion": "2.0.0"})

client.Flush()
```

### Custom Configuration

```go
client, err := ripple.NewClient(ripple.ClientConfig{
    APIKey:         "your-api-key",
    Endpoint:       "https://api.example.com/events",
    APIKeyHeader:   stringPtr("Authorization"),
    FlushInterval:  10 * time.Second,
    MaxBatchSize:   20,
    MaxRetries:     5,
    MaxBufferSize:  1000,
    HTTPAdapter:    adapters.NewNetHTTPAdapter(),
    StorageAdapter: adapters.NewFileStorageAdapter("ripple_events.json"),
    LoggerAdapter:  adapters.NewPrintLoggerAdapter(adapters.LogLevelDebug),
})
```

---

## Development Workflow

### Development Commands

```bash
make test         # Run all tests
make test-cover   # Run tests with coverage
make fmt          # Format all Go files
make fmt-check    # Check if code is formatted
make lint         # Run go vet linter
make build        # Build all packages
make check        # Run all CI checks (fmt, lint, test, build)
```

### Playground

```bash
# Terminal 1: Start server
cd playground && make server

# Terminal 2: Run client
cd playground && make client
```

---

## Design Principles

### Clear Responsibilities

- Client: API surface, disposal tracking, auto-init
- Dispatcher: internal mechanics, retry, timer
- Queue: data structure, thread-safe
- Adapters: extensibility

### Concurrency Safety

- Double-checked locking for initialization
- Mutex around flush cycles
- RWMutex for metadata access
- Context cancellation for retry abort

### Reliability

- Persistent queueing
- Retried delivery with backoff (30s cap)
- Safe process shutdown
- Proper error handling (no panics in library code)

### Simplicity

- Single self-contained package
- No external dependencies
- Clean, predictable API
- Modern Go idioms

---

## API Contract

The SDK follows a framework-agnostic design and API contract defined in the main Ripple repository. See: <https://github.com/Tap30/ripple/blob/main/DESIGN_AND_CONTRACTS.md>

### Key Contract Points

- **Auto-Initialization**: `Track()` auto-calls `Init()` if not initialized
- **Disposal Tracking**: Disposed clients silently drop events
- **Re-Initialization**: Explicit `Init()` after `Dispose()` re-enables
- **Metadata Merging**: Shared metadata + event-specific metadata
- **Platform Detection**: Automatic "server" platform for Go SDK
- **Retry Logic**: Smart retry behavior based on HTTP status codes
- **Graceful Shutdown**: Resources cleaned up on dispose
