package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// NetHTTPAdapter is the standard HTTP adapter implementation using net/http package.
type NetHTTPAdapter struct {
	client *http.Client
}

// Ensure NetHTTPAdapter implements HTTPAdapter interface
var _ HTTPAdapter = (*NetHTTPAdapter)(nil)

// NewNetHTTPAdapter creates a new NetHTTPAdapter instance.
func NewNetHTTPAdapter() HTTPAdapter {
	return &NetHTTPAdapter{
		client: &http.Client{},
	}
}

// Send sends events to the specified endpoint with the given headers.
func (h *NetHTTPAdapter) Send(endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error) {
	payload := map[string]any{
		"events": events,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal events: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return &HTTPResponse{
		Status: resp.StatusCode,
		OK:     resp.StatusCode >= 200 && resp.StatusCode < 300,
	}, nil
}
