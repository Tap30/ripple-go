package ripple

import (
	"time"

	"github.com/Tap30/ripple-go/adapters"
)

// Re-export adapter types for convenience
type (
	// Event represents a trackable analytics event.
	Event = adapters.Event

	// EventMetadata contains optional metadata associated with an event.
	EventMetadata = adapters.EventMetadata

	// Platform describes the runtime environment (e.g., server, client).
	Platform = adapters.Platform

	// HTTPAdapter defines the interface used by the client to perform HTTP requests.
	HTTPAdapter = adapters.HTTPAdapter

	// HTTPResponse represents a response returned by an HTTPAdapter.
	HTTPResponse = adapters.HTTPResponse

	// StorageAdapter defines the interface used for event persistence and retries.
	StorageAdapter = adapters.StorageAdapter

	// LoggerAdapter defines the interface used for internal SDK logging.
	LoggerAdapter = adapters.LoggerAdapter

	// LogLevel represents the severity level for logging.
	LogLevel = adapters.LogLevel
)

type HTTPError struct {
	Status int
}

func (e *HTTPError) Error() string {
	return "HTTP request failed"
}

type ClientConfig struct {
	// APIKey is the authentication key used to authorize requests.
	//
	// Required.
	APIKey string

	// Endpoint is the base HTTPS URL of the Ripple API.
	//
	// Example: https://api.ripple.io
	//
	// Required.
	Endpoint string

	// APIKeyHeader is the HTTP header name used to send the API key.
	//
	// Default: "X-API-Key"
	APIKeyHeader *string

	// FlushInterval controls how often events are automatically flushed
	// to the server.
	//
	// Default: 5 seconds.
	FlushInterval time.Duration

	// MaxBatchSize is the maximum number of events sent in a single request.
	//
	// Default: 10.
	MaxBatchSize int

	// MaxRetries is the maximum number of retry attempts for failed requests.
	//
	// Default: 3.
	MaxRetries int

	// HTTPAdapter is the transport layer used to perform HTTP requests.
	//
	// Required.
	HTTPAdapter HTTPAdapter

	// StorageAdapter is used to persist events for retry and durability.
	//
	// Required.
	StorageAdapter StorageAdapter

	// LoggerAdapter is used for internal SDK logging.
	//
	// Default: PrintLoggerAdapter with WARN level.
	LoggerAdapter LoggerAdapter
}

type DispatcherConfig struct {
	// APIKey is the authentication key used to authorize requests.
	APIKey string

	// APIKeyHeader is the HTTP header name used to send the API key.
	APIKeyHeader string

	// Endpoint is the base HTTPS URL of the Ripple API.
	Endpoint string

	// FlushInterval controls how often queued events are flushed.
	FlushInterval time.Duration

	// MaxBatchSize is the maximum number of events per batch.
	MaxBatchSize int

	// MaxRetries is the maximum number of retry attempts for failed requests.
	MaxRetries int
}
