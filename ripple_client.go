package ripple

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Tap30/ripple-go/adapters"
)

var (
	// serverPlatform is a shared pointer used by all events.
	serverPlatform = &Platform{Type: "server"}
)

type Client struct {
	config          ClientConfig
	metadataManager *MetadataManager
	dispatcher      *Dispatcher
	loggerAdapter   LoggerAdapter
	initialized     bool
	disposed        bool
	initMu          sync.Mutex
}

// NewClient creates a new Ripple client
func NewClient(config ClientConfig) (*Client, error) {
	// Validate required fields
	if config.APIKey == "" {
		return nil, errors.New("api key is required")
	}
	if config.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	if config.HTTPAdapter == nil {
		return nil, errors.New("http adapter is required")
	}
	if config.StorageAdapter == nil {
		return nil, errors.New("storage adapter is required")
	}

	// Validate numeric config values
	if config.FlushInterval < 0 || (config.FlushInterval > 0 && config.FlushInterval < time.Millisecond) {
		return nil, errors.New("flush interval must be a positive duration")
	}
	if config.MaxBatchSize < 0 {
		return nil, errors.New("max batch size must be a positive number")
	}
	if config.MaxRetries < 0 {
		return nil, errors.New("max retries must be a non-negative number")
	}
	if config.MaxBufferSize < 0 {
		return nil, errors.New("max buffer size must be a positive number")
	}

	// Set defaults
	if config.FlushInterval == 0 {
		config.FlushInterval = 5 * time.Second
	}
	if config.MaxBatchSize == 0 {
		config.MaxBatchSize = 10
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	apiKeyHeader := "X-API-Key"
	if config.APIKeyHeader != nil {
		apiKeyHeader = *config.APIKeyHeader
	}

	loggerAdapter := LoggerAdapter(adapters.NewPrintLoggerAdapter(adapters.LogLevelWarn))
	if config.LoggerAdapter != nil {
		loggerAdapter = config.LoggerAdapter
	}

	dispatcherConfig := DispatcherConfig{
		APIKey:        config.APIKey,
		APIKeyHeader:  apiKeyHeader,
		Endpoint:      config.Endpoint,
		FlushInterval: config.FlushInterval,
		MaxBatchSize:  config.MaxBatchSize,
		MaxRetries:    config.MaxRetries,
		MaxBufferSize: config.MaxBufferSize,
	}

	// Validate buffer vs batch
	if config.MaxBufferSize > 0 && config.MaxBufferSize < config.MaxBatchSize {
		return nil, fmt.Errorf("max buffer size (%d) must be greater than or equal to max batch size (%d)", config.MaxBufferSize, config.MaxBatchSize)
	}

	dispatcher := NewDispatcher(dispatcherConfig, config.HTTPAdapter, config.StorageAdapter, loggerAdapter)

	client := &Client{
		config:          config,
		metadataManager: NewMetadataManager(),
		dispatcher:      dispatcher,
		loggerAdapter:   loggerAdapter,
	}

	return client, nil
}

func (c *Client) Init() error {
	c.initMu.Lock()
	defer c.initMu.Unlock()

	if c.initialized {
		return nil
	}

	c.dispatcher.Restore()
	c.disposed = false
	c.initialized = true
	c.loggerAdapter.Info("Client initialized successfully")
	return nil
}

func (c *Client) SetMetadata(key string, value any) {
	c.metadataManager.Set(key, value)
}

func (c *Client) GetMetadata() map[string]any {
	return c.metadataManager.GetAll()
}

func (c *Client) GetSessionId() *string {
	return nil
}

// Track tracks an event with optional payload and metadata.
// Automatically initializes the client if not already initialized.
// If the client is disposed, events are silently dropped.
//
// Parameters:
//   - name: Event name/identifier (required, cannot be empty)
//   - payload: Event data payload (optional, pass nil if not needed)
//   - metadata: Event-specific metadata (optional, pass nil if not needed)
func (c *Client) Track(name string, payload map[string]any, metadata map[string]any) error {
	if name == "" {
		return errors.New("event name cannot be empty")
	}

	if c.disposed {
		c.loggerAdapter.Warn("Cannot track event: Client has been disposed")
		return nil
	}

	if err := c.Init(); err != nil {
		return err
	}

	// Merge shared metadata with event-specific metadata
	eventMetadata := c.metadataManager.GetAll()
	if len(metadata) > 0 {
		if len(eventMetadata) == 0 {
			eventMetadata = metadata
		} else {
			for k, v := range metadata {
				eventMetadata[k] = v
			}
		}
	}

	event := Event{
		Name:      name,
		Payload:   payload,
		Metadata:  eventMetadata,
		IssuedAt:  time.Now().UnixMilli(),
		SessionID: nil,
		Platform:  serverPlatform,
	}

	c.loggerAdapter.Debug("Tracking event: %s", name)
	c.dispatcher.Enqueue(event)
	return nil
}

func (c *Client) Flush() {
	if !c.initialized {
		c.loggerAdapter.Warn("Flush called before initialization")
		return
	}

	c.loggerAdapter.Debug("Flushing events")
	c.dispatcher.Flush()
}

// Dispose cleans up resources. Matches TS dispose() behavior:
// aborts retries, clears queue, clears metadata, resets state.
func (c *Client) Dispose() {
	c.dispatcher.Dispose()
	c.metadataManager.Clear()
	c.disposed = true
	c.initialized = false
	c.loggerAdapter.Info("Client disposed")
}

// Close is an alias for Dispose for idiomatic Go cleanup.
func (c *Client) Close() {
	c.Dispose()
}
