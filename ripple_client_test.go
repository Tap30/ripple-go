package ripple

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Tap30/ripple-go/adapters"
)

func createTestConfig() ClientConfig {
	return ClientConfig{
		APIKey:         "test-key",
		Endpoint:       "http://test.com",
		HTTPAdapter:    &mockHTTPAdapter{},
		StorageAdapter: &mockStorageAdapter{},
	}
}

func createTestClient() *Client {
	client, err := NewClient(createTestConfig())
	if err != nil {
		panic(err)
	}
	return client
}

func TestClient_ConfigValidation(t *testing.T) {
	t.Run("should return error if APIKey is missing", func(t *testing.T) {
		_, err := NewClient(ClientConfig{
			Endpoint:       "http://test.com",
			HTTPAdapter:    &mockHTTPAdapter{},
			StorageAdapter: &mockStorageAdapter{},
		})
		if err == nil {
			t.Fatal("expected error for missing APIKey")
		}
	})

	t.Run("should return error if Endpoint is missing", func(t *testing.T) {
		_, err := NewClient(ClientConfig{
			APIKey:         "test-key",
			HTTPAdapter:    &mockHTTPAdapter{},
			StorageAdapter: &mockStorageAdapter{},
		})
		if err == nil {
			t.Fatal("expected error for missing Endpoint")
		}
	})

	t.Run("should return error if HTTPAdapter is missing", func(t *testing.T) {
		_, err := NewClient(ClientConfig{
			APIKey:         "test-key",
			Endpoint:       "http://test.com",
			StorageAdapter: &mockStorageAdapter{},
		})
		if err == nil {
			t.Fatal("expected error for missing HTTPAdapter")
		}
	})

	t.Run("should return error if StorageAdapter is missing", func(t *testing.T) {
		_, err := NewClient(ClientConfig{
			APIKey:      "test-key",
			Endpoint:    "http://test.com",
			HTTPAdapter: &mockHTTPAdapter{},
		})
		if err == nil {
			t.Fatal("expected error for missing StorageAdapter")
		}
	})

	t.Run("should return error for negative FlushInterval", func(t *testing.T) {
		_, err := NewClient(ClientConfig{
			APIKey:         "test-key",
			Endpoint:       "http://test.com",
			HTTPAdapter:    &mockHTTPAdapter{},
			StorageAdapter: &mockStorageAdapter{},
			FlushInterval:  -1 * time.Second,
		})
		if err == nil {
			t.Fatal("expected error for negative FlushInterval")
		}
	})

	t.Run("should return error for FlushInterval less than 1ms", func(t *testing.T) {
		_, err := NewClient(ClientConfig{
			APIKey:         "test-key",
			Endpoint:       "http://test.com",
			HTTPAdapter:    &mockHTTPAdapter{},
			StorageAdapter: &mockStorageAdapter{},
			FlushInterval:  500 * time.Nanosecond,
		})
		if err == nil {
			t.Fatal("expected error for FlushInterval < 1ms")
		}
	})

	t.Run("should return error for negative MaxBatchSize", func(t *testing.T) {
		_, err := NewClient(ClientConfig{
			APIKey:         "test-key",
			Endpoint:       "http://test.com",
			HTTPAdapter:    &mockHTTPAdapter{},
			StorageAdapter: &mockStorageAdapter{},
			MaxBatchSize:   -5,
		})
		if err == nil {
			t.Fatal("expected error for negative MaxBatchSize")
		}
	})

	t.Run("should return error for negative MaxRetries", func(t *testing.T) {
		_, err := NewClient(ClientConfig{
			APIKey:         "test-key",
			Endpoint:       "http://test.com",
			HTTPAdapter:    &mockHTTPAdapter{},
			StorageAdapter: &mockStorageAdapter{},
			MaxRetries:     -1,
		})
		if err == nil {
			t.Fatal("expected error for negative MaxRetries")
		}
	})

	t.Run("should return error for negative MaxBufferSize", func(t *testing.T) {
		_, err := NewClient(ClientConfig{
			APIKey:         "test-key",
			Endpoint:       "http://test.com",
			HTTPAdapter:    &mockHTTPAdapter{},
			StorageAdapter: &mockStorageAdapter{},
			MaxBufferSize:  -1,
		})
		if err == nil {
			t.Fatal("expected error for negative MaxBufferSize")
		}
	})

	t.Run("should return error when MaxBufferSize < MaxBatchSize", func(t *testing.T) {
		_, err := NewClient(ClientConfig{
			APIKey:         "test-key",
			Endpoint:       "http://test.com",
			HTTPAdapter:    &mockHTTPAdapter{},
			StorageAdapter: &mockStorageAdapter{},
			MaxBatchSize:   100,
			MaxBufferSize:  50,
		})
		if err == nil {
			t.Fatal("expected error when MaxBufferSize < MaxBatchSize")
		}
	})
}

func TestClient_TrackAutoInit(t *testing.T) {
	t.Run("should auto-init when Track is called", func(t *testing.T) {
		client := createTestClient()
		defer client.Dispose()

		err := client.Track("test_event", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("should allow tracking after explicit Init", func(t *testing.T) {
		client := createTestClient()

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

func TestClient_DisposedBehavior(t *testing.T) {
	t.Run("should silently drop events after dispose", func(t *testing.T) {
		client := createTestClient()

		if err := client.Init(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}

		client.Dispose()

		// Track after dispose should return nil (silently dropped)
		err := client.Track("test_event", nil, nil)
		if err != nil {
			t.Fatalf("expected nil error after dispose, got: %v", err)
		}
	})

	t.Run("should re-enable after explicit Init", func(t *testing.T) {
		client := createTestClient()

		if err := client.Init(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}

		client.Dispose()

		// Re-init should work
		if err := client.Init(); err != nil {
			t.Fatalf("failed to re-init: %v", err)
		}
		defer client.Dispose()

		err := client.Track("test_event", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error after re-init: %v", err)
		}
	})

	t.Run("should work before initialization", func(t *testing.T) {
		client := createTestClient()
		// Should not panic when called before init
		client.Dispose()
	})

	t.Run("should work multiple times", func(t *testing.T) {
		client := createTestClient()

		if err := client.Init(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}

		client.Dispose()
		client.Dispose()
	})

	t.Run("dispose should clear metadata", func(t *testing.T) {
		client := createTestClient()

		if err := client.Init(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}

		client.SetMetadata("key", "value")
		client.Dispose()

		metadata := client.GetMetadata()
		if len(metadata) != 0 {
			t.Fatal("expected metadata to be cleared after dispose")
		}
	})
}

func TestClient_TrackValidation(t *testing.T) {
	t.Run("should reject empty event name", func(t *testing.T) {
		client := createTestClient()

		err := client.Track("", nil, nil)
		if err == nil {
			t.Fatal("expected error for empty event name")
		}
		if err.Error() != "event name cannot be empty" {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

func TestClient_MetadataManagement(t *testing.T) {
	client := createTestClient()

	t.Run("should set and get metadata", func(t *testing.T) {
		client.SetMetadata("userId", "123")
		client.SetMetadata("sessionId", "abc")

		metadata := client.GetMetadata()
		if metadata["userId"] != "123" {
			t.Fatal("expected userId to be 123")
		}
		if metadata["sessionId"] != "abc" {
			t.Fatal("expected sessionId to be abc")
		}
	})

	t.Run("should return empty map when no metadata is set", func(t *testing.T) {
		newClient := createTestClient()
		metadata := newClient.GetMetadata()
		if len(metadata) != 0 {
			t.Fatal("expected empty metadata when none is set")
		}
	})
}

func TestClient_FlushEdgeCases(t *testing.T) {
	t.Run("should work with empty queue", func(t *testing.T) {
		client := createTestClient()

		if err := client.Init(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}
		defer client.Dispose()

		client.Flush()
	})

	t.Run("should work before initialization", func(t *testing.T) {
		client := createTestClient()
		client.Flush()
	})
}

func TestClient_GetSessionId(t *testing.T) {
	client := createTestClient()

	sessionID := client.GetSessionId()
	if sessionID != nil {
		t.Fatalf("expected nil session ID for server environment, got %v", *sessionID)
	}
}

func TestClient_Track(t *testing.T) {
	client := createTestClient()

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	client.SetMetadata("userId", "123")
	client.Track("page_view", map[string]any{"page": "/home"}, nil)

	time.Sleep(100 * time.Millisecond)

	if client.dispatcher.queue.Len() == 0 {
		t.Fatal("expected event to be tracked")
	}
}

func TestClient_TrackWithMetadata(t *testing.T) {
	client := createTestClient()

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	metadata := map[string]any{"schemaVersion": "1.0.0"}
	client.Track("user_signup", map[string]any{"email": "test@example.com"}, metadata)

	time.Sleep(100 * time.Millisecond)

	if client.dispatcher.queue.Len() == 0 {
		t.Fatal("expected event with metadata to be tracked")
	}
}

func TestClient_Flush(t *testing.T) {
	mockHTTP := &mockHTTPAdapter{}
	client, _ := NewClient(ClientConfig{
		APIKey:         "test-key",
		Endpoint:       "http://test.com",
		HTTPAdapter:    mockHTTP,
		StorageAdapter: &mockStorageAdapter{},
	})

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	client.Track("test_event", nil, nil)
	client.Flush()

	if mockHTTP.getCalls() != 1 {
		t.Fatalf("expected 1 HTTP call, got %d", mockHTTP.getCalls())
	}
}

func TestClient_DefaultConfig(t *testing.T) {
	client := createTestClient()

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

func TestClient_InitEdgeCases(t *testing.T) {
	t.Run("should handle init when already initialized", func(t *testing.T) {
		client := createTestClient()

		err := client.Init()
		if err != nil {
			t.Fatalf("unexpected error on first init: %v", err)
		}
		defer client.Dispose()

		err = client.Init()
		if err != nil {
			t.Fatalf("unexpected error on second init: %v", err)
		}
	})

	t.Run("should handle concurrent init calls safely", func(t *testing.T) {
		client := createTestClient()
		defer client.Dispose()

		done := make(chan struct{})
		for i := 0; i < 10; i++ {
			go func() {
				client.Init()
				done <- struct{}{}
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify initialization by tracking an event (uses public API only)
		err := client.Track("test_event", nil, nil)
		if err != nil {
			t.Fatalf("expected initialized client to track events, got error: %v", err)
		}
	})

	t.Run("should use provided LoggerAdapter", func(t *testing.T) {
		config := createTestConfig()
		customLogger := adapters.NewNoOpLoggerAdapter()
		config.LoggerAdapter = customLogger

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if client.loggerAdapter != customLogger {
			t.Fatal("expected custom logger to be used")
		}
	})
}

func TestClient_SharedMetadataMerging(t *testing.T) {
	client := createTestClient()

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	client.SetMetadata("userId", "123")
	client.SetMetadata("appVersion", "1.0.0")

	client.Track("test_event", map[string]any{"action": "click"}, map[string]any{"schemaVersion": "2.0.0"})

	time.Sleep(50 * time.Millisecond)

	if client.dispatcher.queue.Len() > 0 {
		event, ok := client.dispatcher.queue.Dequeue()
		if !ok {
			t.Error("failed to dequeue event")
			return
		}

		if event.Metadata["userId"] != "123" {
			t.Errorf("expected userId to be 123, got %v", event.Metadata["userId"])
		}
		if event.Metadata["appVersion"] != "1.0.0" {
			t.Errorf("expected appVersion to be 1.0.0, got %v", event.Metadata["appVersion"])
		}
		if event.Metadata["schemaVersion"] != "2.0.0" {
			t.Errorf("expected schemaVersion to be 2.0.0, got %v", event.Metadata["schemaVersion"])
		}
	} else {
		t.Error("expected event to be in queue")
	}
}

func TestClient_SharedMetadataOverride(t *testing.T) {
	client := createTestClient()

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	client.SetMetadata("environment", "test")
	client.SetMetadata("version", "1.0.0")

	client.Track("test_event", map[string]any{"action": "click"}, map[string]any{"version": "2.0.0", "source": "button"})

	time.Sleep(50 * time.Millisecond)

	if client.dispatcher.queue.Len() > 0 {
		event, ok := client.dispatcher.queue.Dequeue()
		if !ok {
			t.Error("failed to dequeue event")
			return
		}

		if event.Metadata["environment"] != "test" {
			t.Errorf("expected environment to be test, got %v", event.Metadata["environment"])
		}
		if event.Metadata["version"] != "2.0.0" {
			t.Errorf("expected version to be 2.0.0 (overridden), got %v", event.Metadata["version"])
		}
		if event.Metadata["source"] != "button" {
			t.Errorf("expected source to be button, got %v", event.Metadata["source"])
		}
	} else {
		t.Error("expected event to be in queue")
	}
}

func TestClient_TrackWithOnlySharedMetadata(t *testing.T) {
	client := createTestClient()

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	client.SetMetadata("userId", "123")
	client.Track("test_event", nil, nil)

	time.Sleep(50 * time.Millisecond)

	if client.dispatcher.queue.Len() > 0 {
		event, ok := client.dispatcher.queue.Dequeue()
		if !ok {
			t.Error("failed to dequeue event")
			return
		}

		if event.Metadata["userId"] != "123" {
			t.Errorf("expected userId to be 123, got %v", event.Metadata["userId"])
		}
		if len(event.Metadata) != 1 {
			t.Errorf("expected 1 metadata field, got %d", len(event.Metadata))
		}
	} else {
		t.Error("expected event to be in queue")
	}
}

func TestClient_TrackWithNoMetadata(t *testing.T) {
	client := createTestClient()

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	client.Track("test_event", nil, nil)

	time.Sleep(50 * time.Millisecond)

	if client.dispatcher.queue.Len() > 0 {
		event, ok := client.dispatcher.queue.Dequeue()
		if !ok {
			t.Error("failed to dequeue event")
			return
		}

		if len(event.Metadata) != 0 {
			t.Errorf("expected metadata to be empty, got %v", event.Metadata)
		}
	} else {
		t.Error("expected event to be in queue")
	}
}

func TestClient_MetadataManager_IsEmpty(t *testing.T) {
	client := createTestClient()

	if !client.metadataManager.IsEmpty() {
		t.Fatal("expected metadata manager to be empty")
	}

	client.SetMetadata("key", "value")
	if client.metadataManager.IsEmpty() {
		t.Fatal("expected metadata manager to not be empty")
	}
}

func TestClient_MetadataManager_Clear(t *testing.T) {
	client := createTestClient()

	client.SetMetadata("key1", "value1")
	client.SetMetadata("key2", "value2")

	client.metadataManager.Clear()

	if !client.metadataManager.IsEmpty() {
		t.Fatal("expected metadata manager to be empty after clear")
	}
}

func TestClient_StorageAdapterFailures(t *testing.T) {
	storageAdapter := &mockStorageAdapter{err: errors.New("storage error")}

	client, err := NewClient(ClientConfig{
		APIKey:         "test-key",
		Endpoint:       "https://api.example.com",
		FlushInterval:  100 * time.Millisecond,
		MaxBatchSize:   10,
		MaxRetries:     3,
		MaxBufferSize:  100,
		HTTPAdapter:    &mockHTTPAdapter{},
		StorageAdapter: storageAdapter,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Init should succeed even with storage error (restore logs, doesn't fail)
	err = client.Init()
	if err != nil {
		t.Fatalf("Init should succeed even with storage error: %v", err)
	}

	// Track should work even if storage fails
	storageAdapter.err = errors.New("save error")
	if err := client.Track("test_event", nil, nil); err != nil {
		t.Errorf("Track should not fail even if storage fails: %v", err)
	}

	client.Dispose()
}

func TestFileStorageAdapter_EdgeCases(t *testing.T) {
	t.Run("should handle load when file does not exist", func(t *testing.T) {
		adapter := adapters.NewFileStorageAdapter("nonexistent_file.json")

		events, err := adapter.Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(events) != 0 {
			t.Fatalf("expected empty events, got %d", len(events))
		}
	})

	t.Run("should handle load with invalid JSON", func(t *testing.T) {
		filename := "invalid.json"
		adapter := adapters.NewFileStorageAdapter(filename)

		err := os.WriteFile(filename, []byte("invalid json"), 0o644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		defer os.Remove(filename)

		events, err := adapter.Load()
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}

		if events != nil {
			t.Fatal("expected nil events on error")
		}
	})
}

func TestClient_Close(t *testing.T) {
	client, _ := NewClient(ClientConfig{
		APIKey:         "test-key",
		Endpoint:       "http://localhost:8080",
		HTTPAdapter:    &mockHTTPAdapter{},
		StorageAdapter: &mockStorageAdapter{},
	})

	client.Init()
	client.Close()

	if !client.disposed {
		t.Error("Close should dispose the client")
	}
}
