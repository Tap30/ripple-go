# Ripple Go Adapters

This package contains the adapter interfaces and default implementations for the Ripple Go SDK.

## Interfaces

### HTTPAdapter

Interface for HTTP communication. Implement this to use custom HTTP clients.

```go
type HTTPAdapter interface {
    Send(endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error)
    SendWithContext(ctx context.Context, endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error)
}
```

**Default Implementation:** `NetHTTPAdapter`

- Uses Go's standard `net/http` package
- Sends events as JSON POST requests
- Supports custom headers and context cancellation

### StorageAdapter

Interface for event persistence. Implement this to use custom storage backends.

```go
type StorageAdapter interface {
    Save(events []Event) error
    Load() ([]Event, error)
    Clear() error
}
```

**NoOp Implementation:** `NoOpStorageAdapter`

- Performs no storage operations
- Default choice for most use cases
- Useful when persistence is not required

### LoggerAdapter

Interface for internal SDK logging.

```go
type LoggerAdapter interface {
    Debug(message string, args ...any)
    Info(message string, args ...any)
    Warn(message string, args ...any)
    Error(message string, args ...any)
}
```

**Default Implementation:** `PrintLoggerAdapter` (configurable log level)
**NoOp Implementation:** `NoOpLoggerAdapter` (silent)

## Types

### StorageQuotaExceededError

Storage adapters should return this error when they cannot save events due to quota limits. The dispatcher logs it as a warning instead of an error.

```go
type StorageQuotaExceededError struct {
    Message string
}
```

## Custom Implementations

### Example: Custom HTTP Adapter

```go
package main

import (
    "context"
    "github.com/Tap30/ripple-go/adapters"
)

type MyHTTPAdapter struct{}

func (a *MyHTTPAdapter) Send(endpoint string, events []adapters.Event, headers map[string]string) (*adapters.HTTPResponse, error) {
    return a.SendWithContext(context.Background(), endpoint, events, headers)
}

func (a *MyHTTPAdapter) SendWithContext(ctx context.Context, endpoint string, events []adapters.Event, headers map[string]string) (*adapters.HTTPResponse, error) {
    // your custom HTTP logic
    return &adapters.HTTPResponse{Status: 200}, nil
}
```

### Example: Custom Storage Adapter

```go
package main

import (
    "encoding/json"
    "os"
    "github.com/Tap30/ripple-go/adapters"
)

type FileStorage struct {
    filepath string
}

func (f *FileStorage) Save(events []adapters.Event) error {
    data, err := json.Marshal(events)
    if err != nil {
        return err
    }
    return os.WriteFile(f.filepath, data, 0o644)
}

func (f *FileStorage) Load() ([]adapters.Event, error) {
    data, err := os.ReadFile(f.filepath)
    if err != nil {
        if os.IsNotExist(err) {
            return []adapters.Event{}, nil
        }
        return nil, err
    }
    var events []adapters.Event
    if err := json.Unmarshal(data, &events); err != nil {
        return nil, err
    }
    return events, nil
}

func (f *FileStorage) Clear() error {
    err := os.Remove(f.filepath)
    if err != nil && !os.IsNotExist(err) {
        return err
    }
    return nil
}
```

## Usage with Client

Adapters are configured via `ClientConfig` in `NewClient()`:

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
        StorageAdapter: adapters.NewNoOpStorageAdapter(),
        LoggerAdapter:  adapters.NewPrintLoggerAdapter(adapters.LogLevelDebug),
    })
    if err != nil {
        panic(err)
    }
    defer client.Dispose()

    client.Track("event", nil, nil)
}
```
