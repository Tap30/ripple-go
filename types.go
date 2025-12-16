package ripple

import (
	"time"

	"github.com/Tap30/ripple-go/adapters"
)

// Re-export adapter types for convenience
type (
	Event          = adapters.Event
	EventMetadata  = adapters.EventMetadata
	Platform       = adapters.Platform
	HTTPAdapter    = adapters.HTTPAdapter
	HTTPResponse   = adapters.HTTPResponse
	StorageAdapter = adapters.StorageAdapter
	LoggerAdapter  = adapters.LoggerAdapter
	LogLevel       = adapters.LogLevel
)

type HTTPError struct {
	Status int
}

func (e *HTTPError) Error() string {
	return "HTTP request failed"
}

type ClientConfig struct {
	APIKey        string
	Endpoint      string
	APIKeyHeader  *string
	FlushInterval time.Duration
	MaxBatchSize  int
	MaxRetries    int
	Adapters      struct {
		HTTPAdapter    HTTPAdapter
		StorageAdapter StorageAdapter
		LoggerAdapter  LoggerAdapter
	}
}

type DispatcherConfig struct {
	APIKey        string
	APIKeyHeader  string
	Endpoint      string
	FlushInterval time.Duration
	MaxBatchSize  int
	MaxRetries    int
}
