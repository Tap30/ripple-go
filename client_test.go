package ripple

import (
	"testing"
	"time"
)

func stringPtr(s string) *string {
	return &s
}

func TestClient_ConfigValidation(t *testing.T) {
	t.Run("should panic if APIKey is missing", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for missing APIKey")
			}
		}()
		NewClient(ClientConfig{
			Endpoint: "http://test.com",
		})
	})

	t.Run("should panic if Endpoint is missing", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for missing Endpoint")
			}
		}()
		NewClient(ClientConfig{
			APIKey: "test-key",
		})
	})
}

func TestClient_InitializationValidation(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		Endpoint: "http://test.com",
	})

	t.Run("should return error if Track called before Init", func(t *testing.T) {
		err := client.Track("test_event", nil, nil)
		if err == nil {
			t.Fatal("expected error when tracking before init")
		}
		expectedMsg := "client not initialized. Call Init() before tracking events"
		if err.Error() != expectedMsg {
			t.Fatalf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("should allow tracking after Init", func(t *testing.T) {
		mockHTTP := &mockHTTPAdapter{}
		mockStorage := &mockStorageAdapter{}
		client.httpAdapter = mockHTTP
		client.storageAdapter = mockStorage

		if err := client.Init(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}
		defer client.Dispose()

		err := client.Track("test_event", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error after init: %v", err)
		}
	})
}

func TestClient_MetadataManagement(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		Endpoint: "http://test.com",
	})

	t.Run("should set and get metadata", func(t *testing.T) {
		client.SetMetadata("userId", "123")
		client.SetMetadata("sessionId", "abc")

		if client.GetMetadata("userId") != "123" {
			t.Fatal("expected userId to be 123")
		}
		if client.GetMetadata("sessionId") != "abc" {
			t.Fatal("expected sessionId to be abc")
		}
	})

	t.Run("should return all metadata", func(t *testing.T) {
		client.SetMetadata("key1", "value1")
		client.SetMetadata("key2", "value2")

		metadata := client.GetAllMetadata()
		if metadata["key1"] != "value1" || metadata["key2"] != "value2" {
			t.Fatal("metadata values do not match")
		}
	})

	t.Run("should return nil when no metadata is set", func(t *testing.T) {
		newClient := NewClient(ClientConfig{
			APIKey:   "test-key",
			Endpoint: "http://test.com",
		})

		metadata := newClient.GetAllMetadata()
		if metadata != nil {
			t.Fatal("expected nil metadata when none is set")
		}
	})
}

func TestClient_FlushEdgeCases(t *testing.T) {
	t.Run("should work with empty queue", func(t *testing.T) {
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

		// Should not panic or error with empty queue
		client.Flush()
	})

	t.Run("should work before initialization", func(t *testing.T) {
		client := NewClient(ClientConfig{
			APIKey:   "test-key",
			Endpoint: "http://test.com",
		})

		// Should not panic when called before init
		client.Flush()
	})
}

func TestClient_DisposeEdgeCases(t *testing.T) {
	t.Run("should work before initialization", func(t *testing.T) {
		client := NewClient(ClientConfig{
			APIKey:   "test-key",
			Endpoint: "http://test.com",
		})

		// Should not panic when called before init
		err := client.Dispose()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("should work multiple times", func(t *testing.T) {
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

		// Should work multiple times without error
		client.Dispose()
		client.Dispose()
	})
}

func TestClient_DisposeWithoutFlush(t *testing.T) {
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

	// Add an event
	client.Track("test_event", nil, nil)

	// Dispose without flush should not send HTTP request
	err := client.DisposeWithoutFlush()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// HTTP adapter should not have been called
	if mockHTTP.calls > 0 {
		t.Fatal("expected no HTTP calls when disposing without flush")
	}
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
