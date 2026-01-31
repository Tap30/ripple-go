package ripple

import (
	"errors"
	"sync"
	"time"

	"github.com/Tap30/ripple-go/adapters"
)

var (
	serverPlatform = &Platform{Type: "server"}
	eventPool      = sync.Pool{
		New: func() any {
			return &Event{}
		},
	}
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

// NewClient creates a new type-safe Ripple client
func NewClient(config ClientConfig) (*Client, error) {
	// Validate required fields
	if config.APIKey == "" {
		return nil, errors.New("APIKey is required")
	}
	if config.Endpoint == "" {
		return nil, errors.New("Endpoint is required")
	}
	if config.HTTPAdapter == nil {
		return nil, errors.New("HTTPAdapter is required")
	}
	if config.StorageAdapter == nil {
		return nil, errors.New("StorageAdapter is required")
	}

	// Set defaults
	if config.FlushInterval == 0 {
		config.FlushInterval = 5 * time.Second
	}
	if config.MaxBatchSize <= 0 {
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

func (c *Client) SetMetadata(key string, value any) error {
	keyLen := len(key)
	if keyLen == 0 {
		return errors.New("metadata key cannot be empty")
	}
	if keyLen > 255 {
		return errors.New("metadata key cannot exceed 255 characters")
	}

	c.metadataManager.Set(key, value)
	return nil
}

func (c *Client) GetMetadata() map[string]any {
	return c.metadataManager.GetAll()
}

func (c *Client) GetSessionId() *string {
	// Server environments don't use session IDs
	return nil
}

func (c *Client) Track(name string, args ...any) error {
	// Validate event name (optimized single check)
	nameLen := len(name)
	if nameLen == 0 {
		return errors.New("event name cannot be empty")
	}
	if nameLen > 255 {
		return errors.New("event name cannot exceed 255 characters")
	}

	// Parse optional arguments
	var payload any
	var metadata map[string]any

	if len(args) > 0 {
		payload = args[0]
	}
	if len(args) > 1 {
		if meta, ok := args[1].(map[string]any); ok {
			metadata = meta
		}
	}

	c.mu.RLock()
	initialized := c.initialized
	c.mu.RUnlock()

	if !initialized {
		return errors.New("client not initialized. Call Init() before tracking events")
	}

	// Convert payload to map[string]any if provided
	var eventPayload map[string]any
	if payload != nil {
		if p, ok := payload.(map[string]any); ok {
			eventPayload = p
		} else {
			return errors.New("payload must be of type map[string]any or nil")
		}
	}

	// Use only the provided metadata (no context merging)
	now := time.Now().UnixMilli()
	event := eventPool.Get().(*Event)
	*event = Event{
		Name:      name,
		Payload:   eventPayload,
		Metadata:  metadata,
		IssuedAt:  now,
		SessionID: nil, // Server environments don't use session IDs
		Platform:  serverPlatform,
	}

	c.loggerAdapter.Debug("Tracking event: %s", name)
	c.dispatcher.Enqueue(*event)
	eventPool.Put(event)
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
