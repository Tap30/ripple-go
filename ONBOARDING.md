# Ripple Go SDK - Complete Implementation Guide

## Recent Changes

### Dynamic Rebatching in Flush (Latest)
- **Updated Flush logic** - Now processes all queued events in optimal batches
- **Improved efficiency** - Clears entire queue at once, then processes in batches
- **Better performance** - Reduces queue operations and improves throughput
- **Matches TypeScript SDK** - Consistent behavior across all Ripple SDKs

### Smart Retry Logic Update
- **Updated retry behavior** - Now follows intelligent status code-based retry logic
- **4xx Client Errors** - No retry, events are dropped (prevents infinite loops)
- **5xx Server Errors** - Retry with exponential backoff, re-queue on max retries
- **Network Errors** - Retry with exponential backoff, re-queue on max retries
- **2xx Success** - Clear storage, no retry needed
- **Enhanced logging** - Better visibility into retry decisions and event handling

### Generic Type System Removal
- **Removed all generic types** - Simplified from `Client[TEvents, TMetadata]` to simple `Client`
- **Updated API** - Changed from `NewClient[T, M](config)` to `NewClient(config)`
- **Performance optimizations** - Added object pooling and pre-allocated platform objects
- **Simplified codebase** - Removed type complexity while maintaining functionality
- **Updated all documentation** - Removed generic examples and type-safe usage sections

### Adapter Naming Refactor

- Renamed `DefaultHTTPAdapter` to `NetHTTPAdapter` for better Go conventions
- Updated constructor: `NewDefaultHTTPAdapter()` → `NewNetHTTPAdapter()`

### Timer Behavior Enhancement

- Timer now only starts when first new event is tracked, not during SDK initialization
- Timer automatically stops when queue becomes empty to save CPU cycles and reduce log noise
- If persisted events exist, they remain in queue until a new event triggers the timer
- Maintains same API while improving efficiency for apps with persisted events

### Graceful Shutdown Enhancement

- Added `StopWithoutFlush()` and `DisposeWithoutFlush()` methods for graceful shutdown without flushing events
- Fixed playground client exit behavior to persist events without sending to server

### Error Handling Improvement

- Changed `NewClient()` to return `(*Client, error)` instead of panicking on invalid configuration
- Libraries should never panic as it crashes the entire application and can't be handled by users
- Configuration validation errors are now properly returnable and handleable

### Go Version Upgrade

- Upgraded from Go 1.23 to Go 1.25
- Replaced manual `wg.Add(1)` + `go func()` + `defer wg.Done()` with cleaner `wg.Go()` method
- Reduces boilerplate code and eliminates WaitGroup management errors

## Project Overview

Ripple Go is a high-performance, scalable, and fault-tolerant event tracking SDK implemented as a single Go package. It provides reliable event delivery, batching, retries, persistence, and graceful shutdown for server-side applications.

This version is not a monorepo. It has no browser package, no Node.js package, and no internal modules exposed. All functionality exists within one cohesive Go module that follows the unified API contract defined in the main Ripple repository.

## SDK Features

### Core Features

- **Unified Metadata System** – Single metadata field that merges shared metadata (client-level) with event-specific metadata
- **Type-Safe Metadata Management** – MetadataManager for handling shared metadata with thread-safe operations
- **Initialization Validation** – Track() returns error if called before Init() to prevent data loss
- **Logger Interface** – Pluggable logging with PrintLoggerAdapter and NoOpLoggerAdapter implementations
- **Metadata Management** – shared metadata automatically attached to all events
- **Event Metadata** – optional schema versioning and event-specific metadata
- **Automatic Batching** – dispatch based on batch size
- **Scheduled Flushing** – time-based flush via goroutines
- **Smart Retry Logic** – Intelligent retry behavior based on HTTP status codes:
  - **2xx (Success)**: Clear storage, no retry
  - **4xx (Client Error)**: Drop events, no retry (prevents infinite loops)
  - **5xx (Server Error)**: Retry with exponential backoff, re-queue on max retries
  - **Network Errors**: Retry with exponential backoff, re-queue on max retries
- **Event Persistence** – disk-backed storage for unsent events
- **Queue Management** – FIFO queue using `container/list`
- **Race Condition Prevention** – Mutex-based atomic operations for concurrent safety
- **Graceful Shutdown** – flushes and persists all events on dispose
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
    HTTPAdapter    HTTPAdapter    // Required: Custom HTTP adapter
    StorageAdapter StorageAdapter // Required: Custom storage adapter
    LoggerAdapter  LoggerAdapter  // Optional: Custom logger adapter (default: PrintLoggerAdapter with WARN level)
}
```

### Developer Experience

- Simple, predictable API
- Explicit `error` returns
- Comprehensive tests for all components
- No external dependencies
- Practical examples included

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

````

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
* `Track(name, payload, metadata)` - Track event (returns error if not initialized)
* `SetMetadata(key, value)` - Set shared metadata attached to all events (returns error for validation)
* `GetMetadata()` - Get all shared metadata as map
* `GetSessionId()` - Returns nil for server environments
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
* `GetAll()` - Get all metadata (returns empty map if none)
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
* **Dynamic rebatching** - Flush processes all events in optimal batches for better performance
* Automatic and manual flushing using Mutex
* Batch formation with configurable size
* **Smart retry logic** based on HTTP status codes:
  - **2xx (Success)**: Clear storage, no retry
  - **4xx (Client Error)**: Drop events, no retry (prevents infinite loops)
  - **5xx (Server Error)**: Retry with exponential backoff, re-queue on max retries
  - **Network Errors**: Retry with exponential backoff, re-queue on max retries
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
````

**Log Levels**: `DEBUG`, `INFO`, `WARN`, `ERROR`, `NONE` (string-based)

**Built-in Implementations**:

- `PrintLoggerAdapter` - Standard log output with configurable log level (default: WARN)
- `NoOpLoggerAdapter` - Silent logger that discards all messages

**Usage in SDK**:

- Client initialization and disposal
- Event tracking operations
- HTTP request attempts and failures
- Retry logic with backoff timing
- Storage operations

#### HTTP Adapter

Interface defined in `adapters/http_adapter.go`:

```go
type HTTPAdapter interface {
    Send(endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error)
}
```

Default implementation (`NetHTTPAdapter`):

- Uses `net/http`
- JSON payloads
- Combined headers (default + user headers)
- Configurable API key header name

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

- JSON file written to disk (`ripple_events.json`)
- Unlimited capacity
- Suitable for server environments

---

## Types

### EventMetadata

```go
type EventMetadata = map[string]any
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
    Metadata map[string]any         `json:"metadata,omitempty"`
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

client, err := ripple.NewClient(ripple.ClientConfig{
    APIKey:         "your-api-key",
    Endpoint:       "https://api.example.com/events",
    HTTPAdapter:    adapters.NewNetHTTPAdapter(),
    StorageAdapter: adapters.NewFileStorageAdapter("ripple_events.json"),
})
if err != nil {
    panic(err)
}

// Initialize client (required before tracking)
if err := client.Init(); err != nil {
    panic(err)
}
defer client.Dispose()

// Set shared metadata (attached to all events)
if err := client.SetMetadata("userId", "123"); err != nil {
    panic(err)
}
if err := client.SetMetadata("appVersion", "1.0.0"); err != nil {
    panic(err)
}

// Track events
if err := client.Track("page_view", map[string]interface{}{
    "page": "/home",
}, nil); err != nil {
    panic(err)
}

// Track with event-specific metadata
if err := client.Track("user_action", map[string]interface{}{
    "button": "submit",
}, map[string]any{"schemaVersion": "2.0.0"})

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
if err := client.SetMetadata("userId", "user-123"); err != nil {
    panic(err)
}
if err := client.SetMetadata("sessionId", "session-abc"); err != nil {
    panic(err)
}

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
client, err := ripple.NewClient(ripple.ClientConfig{
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
if err != nil {
    panic(err)
}
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
client, err := ripple.NewClient(ripple.ClientConfig{
    APIKey:         "your-api-key",
    Endpoint:       "https://api.example.com/events",
    HTTPAdapter:    &MyHTTPAdapter{},
    StorageAdapter: adapters.NewFileStorageAdapter("ripple_events.json"),
})
if err != nil {
    panic(err)
}
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
client, err := ripple.NewClient(ripple.ClientConfig{
    APIKey:         "your-api-key",
    Endpoint:       "https://api.example.com/events",
    HTTPAdapter:    adapters.NewNetHTTPAdapter(),
    StorageAdapter: adapters.NewFileStorageAdapter("ripple_events.json"),
    LoggerAdapter:  &MyLoggerAdapter{logger: log.New(os.Stdout, "", log.LstdFlags)},
})
if err != nil {
    panic(err)
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

### CI/CD Pipeline

The project uses GitHub Actions for continuous integration on all pull requests:

**Workflow File**: `.github/workflows/development.yml`

**Jobs**:

- **Unit Tests** - Runs `make test` and `make test-cover`
- **Lint Code** - Runs `make fmt-check` and `make lint`
- **Build Check** - Runs `make build`

**Triggers**: Pull request events (opened, edited, synchronize, reopened)

**Requirements**: All jobs must pass before PR can be merged

**Benefits**: Uses Makefile commands for consistency - modify behavior by updating only the Makefile

### Development Commands

Use the root Makefile for common development tasks:

```bash
# Testing
make test         # Run all tests
make test-cover   # Run tests with coverage

# Code Quality
make fmt          # Format all Go files
make fmt-check    # Check if code is formatted
make lint         # Run go vet linter

# Building
make build        # Build all packages

# CI/CD
make check        # Run all CI checks (fmt, lint, test, build)
make release-test # Test release configuration
make release      # Create actual release

# Development
make dev-deps     # Install development dependencies (goreleaser)
make clean        # Clean build artifacts and release files
```

The `make check` command runs the same validation as GitHub Actions CI, ensuring local development consistency.

### Testing

The project includes test files for every component:

- `ripple_client_test.go`
- `dispatcher_test.go`
- `queue_test.go`
- `storage_adapter_test.go`
- `http_adapter_test.go`

### Manual Commands

If you prefer to run commands directly:

- `go build ./...` - Build all packages
- `go test ./...` - Run all tests
- `go test -v ./...` - Run tests with verbose output
- `go test -cover ./...` - Run tests with coverage
- `go vet ./...` - Run Go vet for static analysis

### Playground

The playground provides a local testing environment:

- `playground/cmd/server/main.go` - HTTP server that receives and logs events
- `playground/cmd/client/main.go` - Interactive client with comprehensive testing options

**Usage:**

```bash
# Terminal 1: Start server
cd playground && make server

# Terminal 2: Run client
cd playground && make client
```

See [playground/README.md](./playground/README.md) for E2E testing scenarios.

### Recommendations

- Strong test coverage for dispatcher and queue logic
- Integration tests for persistence and HTTP transport
- Benchmarks for high-volume event throughput
- Linting via `golangci-lint`

### Contributing Guidelines

**Pull Request Requirements**:

- All CI checks must pass (tests, linting, build)
- Code must be formatted with `gofmt`
- Tests must pass with coverage
- No `go vet` warnings allowed

**Local Development**:

```bash
# Run the same checks as CI
make check       # All CI checks in one command
make test        # Run tests
make fmt         # Format code
make lint        # Run go vet
make build       # Verify build
```

### GitHub Templates

The project includes GitHub templates to ensure consistent contributions:

**Pull Request Template** (`.github/pull_request_template.md`):

- Provides checklists for Bug and Feature PRs
- Ensures proper issue linking with `fixes #number`
- Requires tests and documentation for new features

**Issue Templates** (`.github/ISSUE_TEMPLATE/`):

- **Bug Report** (`bug_report.md`) - Structured template for reporting bugs with Go-specific environment details (OS, Go version, SDK version)
- **Feature Request** (`feature_request.md`) - Template for suggesting new features with problem description and proposed solutions

These templates automatically appear when users create issues or pull requests, ensuring high-quality contributions and comprehensive bug reports.

### Release Process

The project uses [GoReleaser](https://goreleaser.com) for automated releases:

**Configuration**: `.goreleaser.yaml`

- **Library-focused**: Skips binary builds, focuses on source code releases
- **Multi-platform archives**: Creates tar.gz (Linux/macOS) and zip (Windows) archives
- **Comprehensive changelog**: Groups commits by type (Features, Bug Fixes, Performance)
- **Source archives**: Includes all source files, documentation, and examples

**Release Workflow** (`.github/workflows/release.yml`):

- **Trigger**: Merge PR from branch matching `release/x.x.x` pattern (e.g., `release/1.0.0`)
- **Process**: Runs tests, builds archives, generates changelog, creates GitHub release
- **Assets**: Source archives, checksums, and release notes

**Creating a Release**:

```bash
# Create release branch with version in name
git checkout -b release/0.0.1        # Stable release
# or
git checkout -b release/1.0.0-rc     # Release candidate
# or
git checkout -b release/2.0.0-beta   # Beta release

# Make any final changes, update version references, etc.
git commit -m "Prepare release"
git push origin release/0.0.1

# Create and merge PR from release/x.x.x to main
# GitHub Actions will automatically:
# 1. Extract version from branch name
# 2. Run tests and validation
# 3. Create and push tag (e.g., v0.0.1, v1.0.0-rc, v2.0.0-beta)
# 4. Generate changelog and publish GitHub release
```

**Local Testing**:

```bash
# Test release configuration
goreleaser check

# Create snapshot release (no publishing)
goreleaser release --snapshot --clean
```

---

## Design Principles

### Clear Responsibilities

- Client: API surface
- Dispatcher: internal mechanics
- Queue: data structure, thread-safe
- Adapters: extensibility

### Concurrency Safety

- Mutex around flush cycles
- RWMutex for metadata access
- Controlled goroutine lifecycle

### Reliability

- Persistent queueing
- Retried delivery with backoff
- Safe process shutdown
- Proper error handling (no panics in library code)

### Simplicity

- Single self-contained package
- No external dependencies
- Clean, predictable API
- Modern Go idioms (use `any` instead of `interface{}`)

---

## Implementation Notes

### File Organization

Following Go best practices:

- All source files in root directory (no `src/` folder)
- Test files co-located with source (`*_test.go`)
- Adapters in separate `adapters/` package for modularity
- Examples in `examples/` subdirectory
- Single main package name: `ripple`
- Adapter interfaces and implementations in `adapters` package
- Use `any` instead of `interface{}` (Go 1.18+ best practice)

### Concurrency Model

- Dispatcher runs a background goroutine for scheduled flushing
- All queue operations are mutex-protected
- Metadata reads use RWMutex for concurrent access
- Flush operations are serialized to prevent race conditions

### Error Handling

- All errors are returned explicitly
- No panics in library code
- Graceful degradation on network failures
- Failed events are re-queued and persisted

### Memory Management

- Events are stored in a linked list for efficient FIFO operations
- Batching prevents unbounded memory growth
- Persistence ensures events survive process restarts
- No memory leaks from goroutines (proper cleanup on Dispose)

---

## API Contract

The SDK follows a framework-agnostic design and API contract defined in the main Ripple repository. See: <https://github.com/Tap30/ripple/blob/main/DESIGN_AND_CONTRACTS.md>

### Key Contract Points

- **Initialization Required**: `Init()` must be called before `Track()`
- **Error Handling**: `Track()` returns error if not initialized
- **Metadata Merging**: Shared metadata + event-specific metadata
- **Platform Detection**: Automatic "server" platform for Go SDK
- **Retry Logic**: Smart retry behavior based on HTTP status codes (2xx/4xx/5xx/Network)
- **Graceful Shutdown**: Events are flushed and persisted on dispose

---

## Recent Changes

### Generic Type System Removal (Latest)
- **Removed all generic types** - Simplified from `Client[TEvents, TMetadata]` to simple `Client`
- **Updated API** - Changed from `NewClient[T, M](config)` to `NewClient(config)`
- **Performance optimizations** - Added object pooling and pre-allocated platform objects
- **Simplified codebase** - Removed type complexity while maintaining functionality
- **Updated all documentation** - Removed generic examples and type-safe usage sections

### Contract Compliance

- **Removed** non-contract `Context` field from Event struct per specification
- **Fixed** metadata API to match contract exactly:
  - `SetMetadata(key, value)` - Set shared metadata attached to all events
  - `GetMetadata()` - Returns all metadata as map (empty map if none set)
- **Removed** individual metadata getter `GetMetadata(key)` (not in contract)
- **Updated** Track method to use only metadata without context merging
- **Achieved** 98.8% test coverage with contract-compliant implementation

### API Unification (Breaking Change)

- **Removed** `SetContext()` and `GetContext()` methods to match TypeScript SDK
- **Context is now unified with metadata** - use `SetMetadata()` instead
- Updated API to match TypeScript version exactly:
  - `SetMetadata(key, value)` - Set shared metadata attached to all events
  - `GetMetadata()` - Returns all metadata as map (contract-compliant)
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
- Added `SetMetadata()`, `GetMetadata()` methods (contract-compliant)

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
- Timer automatically stops when queue becomes empty to save CPU cycles and reduce log noise
- If persisted events exist, they remain in queue until a new event triggers the timer
- Maintains same API while improving efficiency for apps with persisted events

### Graceful Shutdown Enhancement

- Added `StopWithoutFlush()` and `DisposeWithoutFlush()` methods for graceful shutdown without flushing events
- Fixed playground client exit behavior to persist events without sending to server

### Error Handling Improvement

- Changed `NewClient()` to return `(*Client, error)` instead of panicking on invalid configuration
- Libraries should never panic as it crashes the entire application and can't be handled by users
- Configuration validation errors are now properly returnable and handleable

### Go Version Upgrade

- Upgraded from Go 1.23 to Go 1.25
- Replaced manual `wg.Add(1)` + `go func()` + `defer wg.Done()` with cleaner `wg.Go()` method
- Reduces boilerplate code and eliminates WaitGroup management errors
