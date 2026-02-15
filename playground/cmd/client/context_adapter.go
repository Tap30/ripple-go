package main

import (
	"context"
	"time"

	"github.com/Tap30/ripple-go/adapters"
)

// ContextAwareHTTPAdapter wraps the standard adapter with configurable context
type ContextAwareHTTPAdapter struct {
	adapter adapters.HTTPAdapter
	timeout time.Duration
}

func NewContextAwareHTTPAdapter(timeout time.Duration) *ContextAwareHTTPAdapter {
	return &ContextAwareHTTPAdapter{
		adapter: adapters.NewNetHTTPAdapter(),
		timeout: timeout,
	}
}

func (c *ContextAwareHTTPAdapter) Send(endpoint string, events []adapters.Event, headers map[string]string, apiKeyHeader string) (*adapters.HTTPResponse, error) {
	return c.adapter.Send(endpoint, events, headers, apiKeyHeader)
}

func (c *ContextAwareHTTPAdapter) SendWithContext(ctx context.Context, endpoint string, events []adapters.Event, headers map[string]string, apiKeyHeader string) (*adapters.HTTPResponse, error) {
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	return c.adapter.SendWithContext(ctx, endpoint, events, headers, apiKeyHeader)
}

func (c *ContextAwareHTTPAdapter) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}
