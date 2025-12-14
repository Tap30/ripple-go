package ripple

import (
	"testing"
	"time"
)

func stringPtr(s string) *string {
	return &s
}

func TestClient_SetGetContext(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		Endpoint: "http://test.com",
	})

	client.SetContext("userId", "123")
	client.SetContext("appVersion", "1.0.0")

	ctx := client.GetContext()
	if ctx["userId"] != "123" || ctx["appVersion"] != "1.0.0" {
		t.Fatal("context values do not match")
	}
}

func TestClient_Track(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		Endpoint: "http://test.com",
	})

	mockHTTP := &mockHTTPAdapter{}
	mockStorage := &mockStorageAdapter{}
	client.httpAdapter = mockHTTP
	client.storageAdapter = mockStorage

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	client.SetContext("userId", "123")
	client.Track("page_view", map[string]interface{}{"page": "/home"}, nil)

	time.Sleep(100 * time.Millisecond)

	if client.dispatcher.queue.Len() == 0 && mockHTTP.calls == 0 {
		t.Fatal("expected event to be tracked")
	}
}

func TestClient_TrackWithMetadata(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		Endpoint: "http://test.com",
	})

	mockHTTP := &mockHTTPAdapter{}
	mockStorage := &mockStorageAdapter{}
	client.httpAdapter = mockHTTP
	client.storageAdapter = mockStorage

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	metadata := &EventMetadata{SchemaVersion: stringPtr("1.0.0")}
	client.Track("user_signup", map[string]interface{}{"email": "test@example.com"}, metadata)

	time.Sleep(100 * time.Millisecond)

	if client.dispatcher.queue.Len() == 0 && mockHTTP.calls == 0 {
		t.Fatal("expected event with metadata to be tracked")
	}
}

func TestClient_Flush(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		Endpoint: "http://test.com",
	})

	mockHTTP := &mockHTTPAdapter{}
	mockStorage := &mockStorageAdapter{}
	client.httpAdapter = mockHTTP
	client.storageAdapter = mockStorage

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	client.Track("test_event", nil, nil)
	client.Flush()

	if mockHTTP.calls != 1 {
		t.Fatalf("expected 1 HTTP call, got %d", mockHTTP.calls)
	}
}

func TestClient_DefaultConfig(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		Endpoint: "http://test.com",
	})

	if client.config.FlushInterval != 5*time.Second {
		t.Fatal("expected default flush interval of 5s")
	}
	if client.config.MaxBatchSize != 10 {
		t.Fatal("expected default max batch size of 10")
	}
	if client.config.MaxRetries != 3 {
		t.Fatal("expected default max retries of 3")
	}
}

func TestClient_SetCustomAdapters(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		Endpoint: "http://test.com",
	})

	customHTTP := &mockHTTPAdapter{}
	customStorage := &mockStorageAdapter{}

	client.SetHTTPAdapter(customHTTP)
	client.SetStorageAdapter(customStorage)

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	client.Track("test", nil, nil)
	client.Flush()

	if customHTTP.calls == 0 {
		t.Fatal("expected custom HTTP adapter to be used")
	}
}
