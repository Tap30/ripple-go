package ripple

import (
	"errors"
	"sync"
	"time"

	"github.com/Tap30/ripple-go/adapters"
)

type Client struct {
	config          ClientConfig
	metadataManager *MetadataManager
	dispatcher      *Dispatcher
	httpAdapter     HTTPAdapter
	storageAdapter  StorageAdapter
	loggerAdapter   LoggerAdapter
	initialized     bool
	mu              sync.RWMutex
}

func NewClient(config ClientConfig) (*Client, error) {
	// Validate required fields
	if config.APIKey == "" {
		return nil, errors.New("apiKey must be provided in config")
	}
	if config.Endpoint == "" {
		return nil, errors.New("endpoint must be provided in config")
	}
	if config.HTTPAdapter == nil || config.StorageAdapter == nil {
		return nil, errors.New("both HTTPAdapter and StorageAdapter must be provided in config")
	}

	// Set defaults
	if config.FlushInterval == 0 {
		config.FlushInterval = 5 * time.Second
	}
	if !(config.MaxBatchSize > 0) {
		config.MaxBatchSize = 10
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	client := &Client{
		config:          config,
		metadataManager: NewMetadataManager(),
		httpAdapter:     config.HTTPAdapter,
		storageAdapter:  config.StorageAdapter,
	}

	// Use provided logger or default
	if config.LoggerAdapter != nil {
		client.loggerAdapter = config.LoggerAdapter
	} else {
		client.loggerAdapter = adapters.NewPrintLoggerAdapter(adapters.LogLevelWarn)
	}

	return client, nil
}

// SetHTTPAdapter sets a custom HTTP adapter.
// Must be called before Init().
func (c *Client) SetHTTPAdapter(adapter HTTPAdapter) {
	c.httpAdapter = adapter
}

// SetStorageAdapter sets a custom storage adapter.
// Must be called before Init().
func (c *Client) SetStorageAdapter(adapter StorageAdapter) {
	c.storageAdapter = adapter
}

func (c *Client) Init() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	apiKeyHeader := "X-API-Key"
	if c.config.APIKeyHeader != nil {
		apiKeyHeader = *c.config.APIKeyHeader
	}

	headers := map[string]string{
		apiKeyHeader: c.config.APIKey,
	}

	dispatcherConfig := DispatcherConfig{
		APIKey:        c.config.APIKey,
		APIKeyHeader:  apiKeyHeader,
		Endpoint:      c.config.Endpoint,
		FlushInterval: c.config.FlushInterval,
		MaxBatchSize:  c.config.MaxBatchSize,
		MaxRetries:    c.config.MaxRetries,
	}

	c.dispatcher = NewDispatcher(dispatcherConfig, c.httpAdapter, c.storageAdapter, headers)
	c.dispatcher.SetLoggerAdapter(c.loggerAdapter)
	err := c.dispatcher.Start()
	if err != nil {
		return err
	}

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

func (c *Client) Track(name string, payload map[string]any, metadata *EventMetadata) error {
	c.mu.RLock()
	initialized := c.initialized
	c.mu.RUnlock()

	if !initialized {
		return errors.New("client not initialized. Call Init() before tracking events")
	}

	// Use only the provided metadata (no context merging)
	event := Event{
		Name:      name,
		Payload:   payload,
		Metadata:  metadata,
		IssuedAt:  time.Now().UnixMilli(),
		SessionID: nil, // Server platform doesn't use session ID
		Platform:  &Platform{Type: "server"},
	}

	c.loggerAdapter.Debug("Tracking event: %s", name)
	c.dispatcher.Enqueue(event)
	return nil
}

func (c *Client) Flush() {
	c.mu.RLock()
	initialized := c.initialized
	c.mu.RUnlock()

	if !initialized {
		c.loggerAdapter.Warn("Flush called before initialization")
		return
	}

	c.loggerAdapter.Debug("Flushing events")
	c.dispatcher.Flush()
}

func (c *Client) Dispose() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil
	}

	c.loggerAdapter.Info("Disposing client")
	err := c.dispatcher.Stop()
	c.initialized = false
	return err
}

// DisposeWithoutFlush stops the client and persists events to storage without flushing to server
func (c *Client) DisposeWithoutFlush() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil
	}

	c.loggerAdapter.Info("Disposing client without flush")
	err := c.dispatcher.StopWithoutFlush()
	c.initialized = false
	return err
}
