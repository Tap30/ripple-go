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

Ripple Go is a high-performance, scalable, and fault-tolerant event tracking SDK implemented as a single Go package. It provides reliable event delivery, batching, retries, persistence, and graceful shutdown for server-side applications.

This version is not a monorepo. It has no browser package, no Node.js package, and no internal modules exposed. All functionality exists within one cohesive Go module that follows the unified API contract defined in the main Ripple repository.

## SDK Features

### Core Features

* **Unified Metadata System** – Single metadata field that merges shared metadata (client-level) with event-specific metadata
* **Type-Safe Metadata Management** – MetadataManager for handling shared metadata with thread-safe operations
* **Initialization Validation** – Track() throws error if called before Init() to prevent data loss
* **Logger Interface** – Pluggable logging with PrintLoggerAdapter and NoOpLoggerAdapter implementations
* **Context Management** – shared context automatically attached to all events
* **Event Metadata** – optional schema versioning and event-specific metadata
* **Automatic Batching** – dispatch based on batch size
* **Scheduled Flushing** – time-based flush via goroutines
* **Retry Logic** – exponential backoff with jitter (1000ms × 2^attempt + random jitter)
* **Event Persistence** – disk-backed storage for unsent events
* **Queue Management** – FIFO queue using `container/list`
* **Race Condition Prevention** – Mutex-based atomic operations for concurrent safety
* **Graceful Shutdown** – flushes and persists all events on dispose
* **Adapters** – pluggable HTTP, storage, and logger implementations

### Go-Specific Features

* **Safe concurrency** (mutex-protected dispatcher and context)
* **Native HTTP client** (`net/http`)
* **File-based persistence** using JSON
* **Automatic boot-time recovery** from persisted events
* **Zero external dependencies**; uses only standard library

### Configuration

```go
type ClientConfig struct {
    APIKey         string
    Endpoint       string
    APIKeyHeader   *string        // Optional: Header name for API key (default: "X-API-Key")
    FlushInterval  time.Duration  // Default: 5s
    MaxBatchSize   int            // Default: 10
    MaxRetries     int            // Default: 3
    HTTPAdapter    HTTPAdapter    // Required: Custom HTTP adapter
    StorageAdapter StorageAdapter // Required: Custom storage adapter
    LoggerAdapter  LoggerAdapter  // Optional: Custom logger adapter (default: PrintLoggerAdapter with WARN level)
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
├── ripple_client.go            # Main client implementation with metadata management
├── ripple_client_test.go       # Client tests
├── dispatcher.go               # Event batching, retry logic, and HTTP dispatch
├── dispatcher_test.go          # Dispatcher tests
├── queue.go                    # FIFO queue implementation
├── queue_test.go               # Queue tests
├── metadata_manager.go         # Shared metadata management
├── mutex.go                    # Race condition prevention
├── types.go                    # Type definitions and re-exports
├── types_test.go               # Type tests
├── go.mod                      # Go module definition
├── Makefile                    # Build commands (test, fmt, lint, clean)
├── README.md                   # Project documentation
├── ONBOARDING.md              # This file - complete implementation guide
├── adapters/
│   ├── http_adapter.go         # HTTP adapter interface
│   ├── net_http_adapter.go     # Default HTTP implementation
│   ├── net_http_adapter_test.go # HTTP adapter tests
│   ├── storage_adapter.go      # Storage adapter interface
│   ├── file_storage_adapter.go # Default file storage implementation
│   ├── file_storage_adapter_test.go # Storage adapter tests
│   ├── logger_adapter.go       # Logger adapter interface
│   ├── print_logger_adapter.go  # Print logger implementation
│   ├── noop_logger_adapter.go  # No-op logger implementation
│   ├── types.go               # Adapter type definitions
│   └── README.md              # Adapter documentation
└── playground/
    ├── cmd/
    │   ├── client/
    │   │   └── main.go        # Interactive test client with comprehensive options
    │   └── server/
    │       └── main.go        # Test server with error simulation
    ├── go.mod
    └── Makefile              # Build commands
```
    ├── Makefile
    └── README.md
```

### Core Components

#### Client

Entry point for the SDK with enhanced initialization validation and metadata management.
Responsibilities:

* Configuration validation (required APIKey and Endpoint)
* Initialization state management
* Managing shared metadata through MetadataManager
* Accepting new events (with initialization check)
* Passing events to the dispatcher
* Exposing flushing and shutdown
* Logger integration

Thread safety is enforced through internal locking and MetadataManager.

Key methods:

* `Init()` - Initialize client and restore persisted events (must be called first)
* `Track(name, payload, metadata)` - Track event (throws error if not initialized)
* `SetMetadata(key, value)` - Set shared metadata attached to all events
* `GetMetadata(key)` - Get shared metadata value
* `GetAllMetadata()` - Get all shared metadata
* `Flush()` - Force flush queued events
* `Dispose()` - Clean up resources and flush events
* `DisposeWithoutFlush()` - Clean up without flushing (persist to storage only)

#### MetadataManager

Manages global metadata attached to all events with thread-safe operations.
Responsibilities:

* Thread-safe metadata storage using `sync.RWMutex`
* Metadata merging (shared + event-specific)
* Null handling (returns `nil` when no metadata is set)

Key methods:

* `Set(key, value)` - Set metadata value
* `Get(key)` - Get metadata value
* `GetAll()` - Get all metadata (returns `nil` if empty)
* `IsEmpty()` - Check if metadata is empty
* `Clear()` - Remove all metadata

#### Mutex

Provides mutual exclusion lock for preventing race conditions in concurrent operations.
Responsibilities:

* Atomic task execution
* Race condition prevention in Dispatcher flush operations
* Queue-based task scheduling with automatic lock release

Key method:

* `RunAtomic(task func() error)` - Execute task with exclusive lock

#### Dispatcher

Handles all operational concerns with enhanced logging and race condition prevention:

* Event queueing with atomic operations
* Persistence with error handling
* Automatic and manual flushing using Mutex
* Batch formation with configurable size
* Retry with exponential backoff and jitter (1000ms × 2^attempt + random jitter)
* De-queuing and re-queuing failed events with proper ordering
* Loading persisted events on startup
* Graceful shutdown with optional flush
* Comprehensive logging for debugging and monitoring

The Mutex prevents concurrent flush operations and ensures thread safety.

Key methods:

* `Enqueue(event)` - Add event to queue
* `Flush()` - Send queued events (atomic operation)
* `Start()` - Initialize and start background processing
* `Stop()` - Graceful shutdown with flush
* `StopWithoutFlush()` - Graceful shutdown without flush
* `SetLoggerAdapter(logger)` - Set custom logger

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

### Adapter Interfaces

#### Logger Adapter

Interface defined in `adapters/logger_adapter.go`:

```go
type LoggerAdapter interface {
    Debug(message string, args ...interface{})
    Info(message string, args ...interface{})
    Warn(message string, args ...interface{})
    Error(message string, args ...interface{})
}
```

**Log Levels**: `DEBUG`, `INFO`, `WARN`, `ERROR`, `NONE` (string-based)

**Built-in Implementations**:

* `PrintLoggerAdapter` - Standard log output with configurable log level (default: WARN)
* `NoOpLoggerAdapter` - Silent logger that discards all messages

**Usage in SDK**:
* Client initialization and disposal
* Event tracking operations
* HTTP request attempts and failures
* Retry logic with backoff timing
* Storage operations

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
* Configurable API key header name

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
import (
    ripple "github.com/Tap30/ripple-go"
    "github.com/Tap30/ripple-go/adapters"
)

client := ripple.NewClient(ripple.ClientConfig{
    APIKey:         "your-api-key",
    Endpoint:       "https://api.example.com/events",
    HTTPAdapter:    adapters.NewNetHTTPAdapter(),
    StorageAdapter: adapters.NewFileStorageAdapter("ripple_events.json"),
})

// Initialize client (required before tracking)
if err := client.Init(); err != nil {
    panic(err)
}
defer client.Dispose()

// Set shared metadata (attached to all events)
client.SetMetadata("userId", "123")
client.SetMetadata("appVersion", "1.0.0")

// Track events
client.Track("page_view", map[string]interface{}{
    "page": "/home",
}, nil)

// Track with event-specific metadata
client.Track("user_action", map[string]interface{}{
    "button": "submit",
}, &ripple.EventMetadata{SchemaVersion: stringPtr("2.0.0")})

// Manual flush
client.Flush()

// Helper function for string pointers
func stringPtr(s string) *string {
    return &s
}
```

**Important**: `Init()` must be called before `Track()`. Calling `Track()` before initialization will return an error to prevent data loss.

### Unified Metadata System

```go
// Set shared metadata (attached to all events)
client.SetMetadata("userId", "user-123")
client.SetMetadata("sessionId", "session-abc")

// Track event with additional metadata
err := client.Track("user_signup", map[string]interface{}{
    "email": "user@example.com",
    "plan":  "premium",
}, &ripple.EventMetadata{
    SchemaVersion: stringPtr("2.0.0"),
})

// Final event will have merged metadata:
// - userId: "user-123" (from shared)
// - sessionId: "session-abc" (from shared)  
// - schemaVersion: "2.0.0" (from event-specific)
```

### Custom Configuration

```go
client := ripple.NewClient(ripple.ClientConfig{
    APIKey:         "your-api-key",
    Endpoint:       "https://api.example.com/events",
    APIKeyHeader:   stringPtr("Authorization"), // Custom header name
    FlushInterval:  10 * time.Second,           // Custom flush interval
    MaxBatchSize:   20,                         // Custom batch size
    MaxRetries:     5,                          // Custom retry count
    HTTPAdapter:    adapters.NewNetHTTPAdapter(),
    StorageAdapter: adapters.NewFileStorageAdapter("ripple_events.json"),
    LoggerAdapter:  adapters.NewPrintLoggerAdapter(adapters.LogLevelDebug),
})
```

### Custom Adapters

#### Custom HTTP Adapter

```go
import "github.com/Tap30/ripple-go/adapters"

type MyHTTPAdapter struct {}

func (a *MyHTTPAdapter) Send(endpoint string, events []adapters.Event, headers map[string]string) (*adapters.HTTPResponse, error) {
    // custom HTTP logic (e.g., using different HTTP client)
    return &adapters.HTTPResponse{OK: true, Status: 200}, nil
}

// Usage
client := ripple.NewClient(ripple.ClientConfig{
    APIKey:         "your-api-key",
    Endpoint:       "https://api.example.com/events",
    HTTPAdapter:    &MyHTTPAdapter{},
    StorageAdapter: adapters.NewFileStorageAdapter("ripple_events.json"),
})
```

#### Custom Logger Adapter

```go
import "github.com/Tap30/ripple-go/adapters"

type MyLoggerAdapter struct {
    logger *log.Logger
}

func (l *MyLoggerAdapter) Debug(message string, args ...interface{}) {
    l.logger.Printf("[DEBUG] "+message, args...)
}

func (l *MyLoggerAdapter) Info(message string, args ...interface{}) {
    l.logger.Printf("[INFO] "+message, args...)
}

func (l *MyLoggerAdapter) Warn(message string, args ...interface{}) {
    l.logger.Printf("[WARN] "+message, args...)
}

func (l *MyLoggerAdapter) Error(message string, args ...interface{}) {
    l.logger.Printf("[ERROR] "+message, args...)
}

// Usage
client := ripple.NewClient(ripple.ClientConfig{
    APIKey:         "your-api-key",
    Endpoint:       "https://api.example.com/events",
    HTTPAdapter:    adapters.NewNetHTTPAdapter(),
    StorageAdapter: adapters.NewFileStorageAdapter("ripple_events.json"),
    LoggerAdapter:  &MyLoggerAdapter{logger: log.New(os.Stdout, "", log.LstdFlags)},
})
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

### Development Commands

Use the root Makefile for common development tasks:

```bash
make test       # Run all tests
make test-cover # Run tests with coverage
make fmt        # Format all Go files
make lint       # Run go vet linter
make clean      # Clean build artifacts
```

### Testing

The project includes test files for every component:

* `ripple_client_test.go`
* `dispatcher_test.go`
* `queue_test.go`
* `storage_adapter_test.go`
* `http_adapter_test.go`

### Manual Commands

If you prefer to run commands directly:

* `go build ./...` - Build all packages
* `go test ./...` - Run all tests
* `go test -v ./...` - Run tests with verbose output
* `go test -cover ./...` - Run tests with coverage
* `go vet ./...` - Run Go vet for static analysis

### Playground

The playground provides a local testing environment:

* `playground/cmd/server/main.go` - HTTP server that receives and logs events
* `playground/cmd/client/main.go` - Interactive client with comprehensive testing options

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
* Modern Go idioms (use `any` instead of `interface{}`)

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
* Use `any` instead of `interface{}` (Go 1.18+ best practice)

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

---

## API Contract

The SDK follows a framework-agnostic design and API contract defined in the main Ripple repository. See: https://github.com/Tap30/ripple/blob/main/DESIGN_AND_CONTRACTS.md

### Key Contract Points

* **Initialization Required**: `Init()` must be called before `Track()`
* **Error Handling**: `Track()` returns error if not initialized
* **Metadata Merging**: Shared metadata + event-specific metadata
* **Platform Detection**: Automatic "server" platform for Go SDK
* **Retry Logic**: Exponential backoff with jitter (1000ms × 2^attempt + random jitter)
* **Graceful Shutdown**: Events are flushed and persisted on dispose

---

## Recent Changes

### API Unification (Breaking Change)
- **Removed** `SetContext()` and `GetContext()` methods to match TypeScript SDK
- **Context is now unified with metadata** - use `SetMetadata()` instead
- Updated API to match TypeScript version exactly:
  - `SetMetadata(key, value)` - Set shared metadata attached to all events
  - `GetMetadata(key)` - Get shared metadata value  
  - `GetAllMetadata()` - Get all shared metadata
- Updated all tests and playground to use new unified API

### Adapter Requirements (Breaking Change)
- **HTTPAdapter** and **StorageAdapter** are now **required** (matching TypeScript SDK)
- **LoggerAdapter** remains optional with PrintLoggerAdapter as default
- Added validation that panics if required adapters are missing
- Removed default adapter creation - must be explicitly provided in config
- Updated all tests and playground to provide required adapters
- Added playground binaries to .gitignore to prevent accidental commits

### File Naming Improvements
- Renamed `client.go` to `ripple_client.go` for better clarity
- Renamed `client_test.go` to `ripple_client_test.go` to match
- Restructured playground to follow Go conventions: `cmd/client/main.go` and `cmd/server/main.go`
- Updated project structure documentation

### Enhanced Playground Client
- Added comprehensive testing options matching TypeScript playground maturity
- **Basic Event Tracking**: Simple events, events with payload, metadata, and custom metadata
- **Metadata Management**: Set shared metadata, track with shared metadata
- **Batch and Flush**: Multiple event tracking, manual flush testing
- **Error Handling**: Retry logic testing, invalid endpoint testing
- **Lifecycle Management**: Client disposal, graceful exit
- Organized menu with categorized options for better user experience

### Logger Interface Addition
- Added `LoggerAdapter` interface with Debug/Info/Warn/Error methods
- Implemented `PrintLoggerAdapter` with configurable log levels
- Implemented `NoOpLoggerAdapter` for silent operation
- Integrated logging throughout Client and Dispatcher operations

### Unified Metadata System
- Added `MetadataManager` for thread-safe shared metadata management
- Implemented metadata merging (shared + event-specific)
- Added `SetMetadata()`, `GetMetadata()`, `GetAllMetadata()` methods
- Maintains backward compatibility with `SetContext()` and `GetContext()`

### Initialization Validation
- `Track()` now returns error if called before `Init()`
- Added initialization state tracking in Client
- Prevents data loss from uninitialized client usage

### Race Condition Prevention
- Added `Mutex` component for atomic operations
- Updated Dispatcher to use `RunAtomic()` for flush operations
- Enhanced thread safety for concurrent operations

### Enhanced Configuration
- Added `APIKeyHeader` support for custom header names
- Flattened adapter configuration directly in `ClientConfig`
- Improved configuration validation with required field checks

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
