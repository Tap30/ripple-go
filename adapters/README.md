# Ripple Go Adapters

This package contains the adapter interfaces and default implementations for the Ripple Go SDK.

## Interfaces

### HTTPAdapter

Interface for HTTP communication. Implement this to use custom HTTP clients.

```go
type HTTPAdapter interface {
    Send(endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error)
}
```

**Default Implementation:** `NetHTTPAdapter`

- Uses Go's standard `net/http` package
- Sends events as JSON POST requests
- Supports custom headers

### StorageAdapter

Interface for event persistence. Implement this to use custom storage backends.

```go
type StorageAdapter interface {
    Save(events []Event) error
    Load() ([]Event, error)
    Clear() error
}
```

**Default Implementation:** `FileStorageAdapter`

- Stores events as JSON in a file
- Default file: `ripple_events.json`
- Suitable for server environments

**NoOp Implementation:** `NoOpStorageAdapter`

- Performs no storage operations
- Save and Clear do nothing
- Load returns empty array
- Useful when persistence is not required

## Custom Implementations

### Example: Custom HTTP Adapter

```go
package main

import "github.com/Tap30/ripple-go/adapters"

type MyHTTPAdapter struct {
    // your custom fields
}

func (a *MyHTTPAdapter) Send(endpoint string, events []adapters.Event, headers map[string]string) (*adapters.HTTPResponse, error) {
    // your custom HTTP logic
    // e.g., using gRPC, custom retry logic, etc.
    return &adapters.HTTPResponse{OK: true, Status: 200}, nil
}
```

### Example: Redis Storage Adapter

```go
package main

import (
    "encoding/json"
    "github.com/Tap30/ripple-go/adapters"
    "github.com/redis/go-redis/v9"
)

type RedisStorageAdapter struct {
    client *redis.Client
    key    string
}

func NewRedisStorageAdapter(client *redis.Client, key string) *RedisStorageAdapter {
    return &RedisStorageAdapter{client: client, key: key}
}

func (r *RedisStorageAdapter) Save(events []adapters.Event) error {
    data, err := json.Marshal(events)
    if err != nil {
        return err
    }
    return r.client.Set(ctx, r.key, data, 0).Err()
}

func (r *RedisStorageAdapter) Load() ([]adapters.Event, error) {
    data, err := r.client.Get(ctx, r.key).Result()
    if err == redis.Nil {
        return []adapters.Event{}, nil
    }
    if err != nil {
        return nil, err
    }
    
    var events []adapters.Event
    if err := json.Unmarshal([]byte(data), &events); err != nil {
        return nil, err
    }
    return events, nil
}

func (r *RedisStorageAdapter) Clear() error {
    return r.client.Del(ctx, r.key).Err()
}
```

### Example: Database Storage Adapter

```go
package main

import (
    "database/sql"
    "encoding/json"
    "github.com/Tap30/ripple-go/adapters"
)

type DatabaseStorageAdapter struct {
    db *sql.DB
}

func NewDatabaseStorageAdapter(db *sql.DB) *DatabaseStorageAdapter {
    return &DatabaseStorageAdapter{db: db}
}

func (d *DatabaseStorageAdapter) Save(events []adapters.Event) error {
    tx, err := d.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    for _, event := range events {
        data, _ := json.Marshal(event)
        _, err := tx.Exec("INSERT INTO events (data) VALUES (?)", data)
        if err != nil {
            return err
        }
    }
    
    return tx.Commit()
}

func (d *DatabaseStorageAdapter) Load() ([]adapters.Event, error) {
    rows, err := d.db.Query("SELECT data FROM events")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var events []adapters.Event
    for rows.Next() {
        var data []byte
        if err := rows.Scan(&data); err != nil {
            return nil, err
        }
        var event adapters.Event
        if err := json.Unmarshal(data, &event); err != nil {
            return nil, err
        }
        events = append(events, event)
    }
    
    return events, nil
}

func (d *DatabaseStorageAdapter) Clear() error {
    _, err := d.db.Exec("DELETE FROM events")
    return err
}
```

## Usage with Client

```go
package main

import (
    ripple "github.com/Tap30/ripple-go"
    "github.com/Tap30/ripple-go/adapters"
)

func main() {
    client := ripple.NewClient(ripple.ClientConfig{
        APIKey:   "your-api-key",
        Endpoint: "https://api.example.com/events",
    })
    
    // Set custom adapters before Init()
    client.SetHTTPAdapter(&MyHTTPAdapter{})
    client.SetStorageAdapter(adapters.NewDefaultStorageAdapter("custom_path.json"))
    
    client.Init()
    defer client.Dispose()
    
    // Use the client normally
    client.Track("event", nil, nil)
}
```

## Design Philosophy

The adapter pattern allows you to:

1. **Swap implementations** without changing core SDK code
2. **Test easily** by using mock adapters
3. **Extend functionality** for specific use cases
4. **Maintain compatibility** across different environments

This matches the TypeScript implementation's approach while following Go idioms.
