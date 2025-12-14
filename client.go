package ripple

import (
	"sync"
	"time"

	"github.com/Tap30/ripple-go/adapters"
)

type Client struct {
	config         ClientConfig
	context        map[string]interface{}
	contextMu      sync.RWMutex
	dispatcher     *Dispatcher
	httpAdapter    HTTPAdapter
	storageAdapter StorageAdapter
}

func NewClient(config ClientConfig) *Client {
	if config.FlushInterval == 0 {
		config.FlushInterval = 5 * time.Second
	}
	if config.MaxBatchSize == 0 {
		config.MaxBatchSize = 10
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	client := &Client{
		config:  config,
		context: make(map[string]interface{}),
	}

	// Use provided adapters or defaults
	if config.Adapters.HTTPAdapter != nil {
		client.httpAdapter = config.Adapters.HTTPAdapter
	} else {
		client.httpAdapter = adapters.NewNetHTTPAdapter()
	}

	if config.Adapters.StorageAdapter != nil {
		client.storageAdapter = config.Adapters.StorageAdapter
	} else {
		client.storageAdapter = adapters.NewFileStorageAdapter("ripple_events.json")
	}

	return client
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
	return c.dispatcher.Start()
}

func (c *Client) SetContext(key string, value interface{}) {
	c.contextMu.Lock()
	defer c.contextMu.Unlock()
	c.context[key] = value
}

func (c *Client) GetContext() map[string]interface{} {
	c.contextMu.RLock()
	defer c.contextMu.RUnlock()
	ctx := make(map[string]interface{}, len(c.context))
	for k, v := range c.context {
		ctx[k] = v
	}
	return ctx
}

func (c *Client) Track(name string, payload map[string]interface{}, metadata *EventMetadata) {
	event := Event{
		Name:      name,
		Payload:   payload,
		Metadata:  metadata,
		IssuedAt:  time.Now().UnixMilli(),
		Context:   c.GetContext(),
		SessionID: nil, // Server platform doesn't use session ID
		Platform:  &Platform{Type: "server"},
	}
	c.dispatcher.Enqueue(event)
}

func (c *Client) Flush() {
	c.dispatcher.Flush()
}

func (c *Client) Dispose() error {
	return c.dispatcher.Stop()
}

// DisposeWithoutFlush stops the client and persists events to storage without flushing to server
func (c *Client) DisposeWithoutFlush() error {
	return c.dispatcher.StopWithoutFlush()
}
