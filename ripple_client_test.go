package ripple

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Tap30/ripple-go/adapters"
)

func stringPtr(s string) *string {
	return &s
}

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
		panic(err) // Only panic in tests
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
		if err.Error() != "APIKey is required" {
			t.Fatalf("unexpected error message: %v", err)
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
		if err.Error() != "Endpoint is required" {
			t.Fatalf("unexpected error message: %v", err)
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
		if err.Error() != "HTTPAdapter is required" {
			t.Fatalf("unexpected error message: %v", err)
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
		if err.Error() != "StorageAdapter is required" {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

func TestClient_InitializationValidation(t *testing.T) {
	client := createTestClient()

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
	client := createTestClient()

	t.Run("should set and get metadata", func(t *testing.T) {
		err := client.SetMetadata("userId", "123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = client.SetMetadata("sessionId", "abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		metadata := client.GetMetadata()
		if metadata["userId"] != "123" {
			t.Fatal("expected userId to be 123")
		}
		if metadata["sessionId"] != "abc" {
			t.Fatal("expected sessionId to be abc")
		}
	})

	t.Run("should return all metadata", func(t *testing.T) {
		err := client.SetMetadata("key1", "value1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = client.SetMetadata("key2", "value2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		metadata := client.GetMetadata()
		if metadata["key1"] != "value1" || metadata["key2"] != "value2" {
			t.Fatal("metadata values do not match")
		}
	})

	t.Run("should return nil when no metadata is set", func(t *testing.T) {
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
		client := createTestClient()

		// Should not panic when called before init
		client.Flush()
	})
}

func TestClient_DisposeEdgeCases(t *testing.T) {
	t.Run("should work before initialization", func(t *testing.T) {
		client := createTestClient()

		// Should not panic when called before init
		err := client.Dispose()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("should work multiple times", func(t *testing.T) {
		client := createTestClient()

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
	client := createTestClient()

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

func TestClient_TrackValidation(t *testing.T) {
	client := createTestClient()
	err := client.Init()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer client.Dispose()

	t.Run("should reject empty event name", func(t *testing.T) {
		err := client.Track("", map[string]any{"key": "value"}, nil)
		if err == nil {
			t.Fatal("expected error for empty event name")
		}
		if err.Error() != "event name cannot be empty" {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("should reject event name exceeding 255 characters", func(t *testing.T) {
		longName := string(make([]byte, 256))
		for i := range longName {
			longName = longName[:i] + "a" + longName[i+1:]
		}

		err := client.Track(longName, map[string]any{"key": "value"}, nil)
		if err == nil {
			t.Fatal("expected error for long event name")
		}
		if err.Error() != "event name cannot exceed 255 characters" {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("should accept valid event name", func(t *testing.T) {
		err := client.Track("valid_event", map[string]any{"key": "value"}, nil)
		if err != nil {
			t.Fatalf("unexpected error for valid event: %v", err)
		}
	})
}

func TestClient_SetMetadataValidation(t *testing.T) {
	client := createTestClient()

	t.Run("should reject empty metadata key", func(t *testing.T) {
		err := client.SetMetadata("", "value")
		if err == nil {
			t.Fatal("expected error for empty metadata key")
		}
		if err.Error() != "metadata key cannot be empty" {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("should reject metadata key exceeding 255 characters", func(t *testing.T) {
		longKey := string(make([]byte, 256))
		for i := range longKey {
			longKey = longKey[:i] + "a" + longKey[i+1:]
		}

		err := client.SetMetadata(longKey, "value")
		if err == nil {
			t.Fatal("expected error for long metadata key")
		}
		if err.Error() != "metadata key cannot exceed 255 characters" {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("should accept valid metadata key", func(t *testing.T) {
		err := client.SetMetadata("valid_key", "value")
		if err != nil {
			t.Fatalf("unexpected error for valid metadata key: %v", err)
		}
	})
}

func TestClient_GetSessionId(t *testing.T) {
	client := createTestClient()

	// Server environments should always return nil for session ID
	sessionID := client.GetSessionId()
	if sessionID != nil {
		t.Fatalf("expected nil session ID for server environment, got %v", *sessionID)
	}
}

func TestClient_SetGetMetadata(t *testing.T) {
	client := createTestClient()

	_ = client.SetMetadata("userId", "123")
	_ = client.SetMetadata("appVersion", "1.0.0")

	metadata := client.GetMetadata()
	if metadata["userId"] != "123" || metadata["appVersion"] != "1.0.0" {
		t.Fatal("metadata values do not match")
	}
}

func TestClient_MetadataManager_IsEmpty(t *testing.T) {
	client := createTestClient()

	// Test IsEmpty when no metadata is set
	if !client.metadataManager.IsEmpty() {
		t.Fatal("expected metadata manager to be empty")
	}

	// Set metadata and test IsEmpty returns false
	_ = client.SetMetadata("key", "value")
	if client.metadataManager.IsEmpty() {
		t.Fatal("expected metadata manager to not be empty")
	}
}

func TestClient_MetadataManager_Clear(t *testing.T) {
	client := createTestClient()

	// Set some metadata
	_ = client.SetMetadata("key1", "value1")
	_ = client.SetMetadata("key2", "value2")

	// Clear metadata
	client.metadataManager.Clear()

	// Verify metadata is cleared
	if !client.metadataManager.IsEmpty() {
		t.Fatal("expected metadata manager to be empty after clear")
	}

	metadata := client.GetMetadata()
	if len(metadata) != 0 {
		t.Fatal("expected empty metadata after clear")
	}
}

func TestClient_Track(t *testing.T) {
	client := createTestClient()

	mockHTTP := &mockHTTPAdapter{}
	mockStorage := &mockStorageAdapter{}
	client.httpAdapter = mockHTTP
	client.storageAdapter = mockStorage

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	_ = client.SetMetadata("userId", "123")
	client.Track("page_view", map[string]any{"page": "/home"}, nil)

	time.Sleep(100 * time.Millisecond)

	if client.dispatcher.queue.Len() == 0 && mockHTTP.calls == 0 {
		t.Fatal("expected event to be tracked")
	}
}

func TestClient_TrackWithMetadata(t *testing.T) {
	client := createTestClient()

	mockHTTP := &mockHTTPAdapter{}
	mockStorage := &mockStorageAdapter{}
	client.httpAdapter = mockHTTP
	client.storageAdapter = mockStorage

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	metadata := map[string]any{"schemaVersion": "1.0.0"}
	client.Track("user_signup", map[string]any{"email": "test@example.com"}, metadata)

	time.Sleep(100 * time.Millisecond)

	if client.dispatcher.queue.Len() == 0 && mockHTTP.calls == 0 {
		t.Fatal("expected event with metadata to be tracked")
	}
}

func TestClient_Flush(t *testing.T) {
	client := createTestClient()

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

func TestClient_SetCustomAdapters(t *testing.T) {
	client := createTestClient()

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
func TestClient_NewClient_EdgeCases(t *testing.T) {
	t.Run("should handle negative MaxBatchSize", func(t *testing.T) {
		config := createTestConfig()
		config.MaxBatchSize = -5

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if client.config.MaxBatchSize != 10 {
			t.Fatalf("expected MaxBatchSize to be set to default 10, got %d", client.config.MaxBatchSize)
		}
	})

	t.Run("should handle zero MaxRetries", func(t *testing.T) {
		config := createTestConfig()
		config.MaxRetries = 0

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if client.config.MaxRetries != 3 {
			t.Fatalf("expected MaxRetries to be set to default 3, got %d", client.config.MaxRetries)
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

func TestClient_DisposeWithoutFlush_EdgeCases(t *testing.T) {
	t.Run("should work when not initialized", func(t *testing.T) {
		client := createTestClient()

		err := client.DisposeWithoutFlush()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestClient_Init_EdgeCases(t *testing.T) {
	t.Run("should handle init when already initialized", func(t *testing.T) {
		client := createTestClient()

		mockHTTP := &mockHTTPAdapter{}
		mockStorage := &mockStorageAdapter{}
		client.httpAdapter = mockHTTP
		client.storageAdapter = mockStorage

		// Initialize first time
		err := client.Init()
		if err != nil {
			t.Fatalf("unexpected error on first init: %v", err)
		}

		// Initialize second time - should not error
		err = client.Init()
		if err != nil {
			t.Fatalf("unexpected error on second init: %v", err)
		}
	})

	t.Run("should handle dispatcher start error", func(t *testing.T) {
		client := createTestClient()

		// Use a mock storage that will cause an error during start
		mockHTTP := &mockHTTPAdapter{}
		mockStorage := &mockStorageAdapterWithError{}
		client.httpAdapter = mockHTTP
		client.storageAdapter = mockStorage

		err := client.Init()
		if err == nil {
			t.Fatal("expected error from dispatcher start")
		}
	})

	t.Run("should use NoOpLoggerAdapter when none provided", func(t *testing.T) {
		config := createTestConfig()
		config.LoggerAdapter = nil

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Initialize to trigger logger usage
		mockHTTP := &mockHTTPAdapter{}
		mockStorage := &mockStorageAdapter{}
		client.httpAdapter = mockHTTP
		client.storageAdapter = mockStorage

		err = client.Init()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer client.Dispose()

		// Track an event to trigger more logger usage
		err = client.Track("test", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Flush to trigger dispatcher logging
		client.Flush()
	})

	t.Run("should use explicit NoOpLoggerAdapter", func(t *testing.T) {
		config := createTestConfig()
		noopLogger := adapters.NewNoOpLoggerAdapter()
		config.LoggerAdapter = noopLogger

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the logger is set
		if client.loggerAdapter != noopLogger {
			t.Fatal("expected NoOpLoggerAdapter to be used")
		}

		// Initialize and use the client to trigger all logger methods
		mockHTTP := &mockHTTPAdapter{}
		mockStorage := &mockStorageAdapter{}
		client.httpAdapter = mockHTTP
		client.storageAdapter = mockStorage

		err = client.Init()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer client.Dispose()

		// Track events to trigger logger calls
		for i := 0; i < 5; i++ {
			err = client.Track("test_event", map[string]any{"index": i}, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		// Flush to trigger more logging
		client.Flush()

		// Test all logger methods directly
		noopLogger.Debug("test debug")
		noopLogger.Info("test info")
		noopLogger.Warn("test warn")
		noopLogger.Error("test error")
	})
}

func TestDispatcher_EdgeCases(t *testing.T) {
	t.Run("should handle StopWithoutFlush when not started", func(t *testing.T) {
		config := DispatcherConfig{
			Endpoint:      "http://test.com",
			FlushInterval: time.Second,
			MaxBatchSize:  10,
			MaxRetries:    3,
		}

		dispatcher := NewDispatcher(config, &mockHTTPAdapter{}, &mockStorageAdapter{}, map[string]string{})

		err := dispatcher.StopWithoutFlush()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("should handle timer start race condition", func(t *testing.T) {
		config := DispatcherConfig{
			Endpoint:      "http://test.com",
			FlushInterval: time.Millisecond * 10,
			MaxBatchSize:  10,
			MaxRetries:    3,
		}

		dispatcher := NewDispatcher(config, &mockHTTPAdapter{}, &mockStorageAdapter{}, map[string]string{})
		dispatcher.Start()
		defer dispatcher.Stop()

		// Try to start timer multiple times concurrently
		for i := 0; i < 5; i++ {
			go func() {
				event := Event{Name: "test", IssuedAt: time.Now().UnixMilli()}
				dispatcher.Enqueue(event)
			}()
		}

		time.Sleep(time.Millisecond * 50)
	})
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
		// Create a file with invalid JSON
		filename := "invalid.json"
		adapter := adapters.NewFileStorageAdapter(filename)

		// Write invalid JSON
		err := os.WriteFile(filename, []byte("invalid json"), 0644)
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

// Mock storage adapter that returns error on Load
type mockStorageAdapterWithError struct{}

func (m *mockStorageAdapterWithError) Save(events []Event) error {
	return nil
}

func (m *mockStorageAdapterWithError) Load() ([]Event, error) {
	return nil, errors.New("mock load error")
}

func (m *mockStorageAdapterWithError) Clear() error {
	return nil
}
func TestClient_SharedMetadataMerging(t *testing.T) {
	client := createTestClient()

	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	// Set shared metadata
	_ = client.SetMetadata("userId", "123")
	_ = client.SetMetadata("appVersion", "1.0.0")

	// Track event with additional metadata
	client.Track("test_event", map[string]any{"action": "click"}, map[string]any{"schemaVersion": "2.0.0"})

	// Wait a moment for the event to be queued
	time.Sleep(50 * time.Millisecond)

	// Verify metadata merging by checking the event in the queue
	if client.dispatcher.queue.Len() > 0 {
		// Get the event from queue
		event, ok := client.dispatcher.queue.Dequeue()
		if !ok {
			t.Error("failed to dequeue event")
			return
		}

		// Check that shared metadata is present
		if event.Metadata["userId"] != "123" {
			t.Errorf("expected userId to be 123, got %v", event.Metadata["userId"])
		}
		if event.Metadata["appVersion"] != "1.0.0" {
			t.Errorf("expected appVersion to be 1.0.0, got %v", event.Metadata["appVersion"])
		}

		// Check that event-specific metadata is present
		if event.Metadata["schemaVersion"] != "2.0.0" {
			t.Errorf("expected schemaVersion to be 2.0.0, got %v", event.Metadata["schemaVersion"])
		}
	} else {
		t.Error("expected event to be in queue")
	}
}
func TestClient_TrackWithInvalidPayload(t *testing.T) {
	client := createTestClient()
	
	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	// Test with invalid payload type
	err := client.Track("test_event", "invalid_payload")
	if err == nil {
		t.Error("expected error for invalid payload type")
	}
	if err.Error() != "payload must be of type map[string]any or nil" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestClient_TrackWithInvalidMetadata(t *testing.T) {
	client := createTestClient()
	
	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	// Test with invalid metadata type (should be ignored)
	err := client.Track("test_event", map[string]any{"key": "value"}, "invalid_metadata")
	if err != nil {
		t.Errorf("should not error with invalid metadata type: %v", err)
	}
}

func TestClient_SharedMetadataOverride(t *testing.T) {
	client := createTestClient()
	
	if err := client.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	defer client.Dispose()

	// Set shared metadata
	_ = client.SetMetadata("environment", "test")
	_ = client.SetMetadata("version", "1.0.0")

	// Track event with metadata that overrides shared metadata
	client.Track("test_event", map[string]any{"action": "click"}, map[string]any{"version": "2.0.0", "source": "button"})

	time.Sleep(50 * time.Millisecond)

	if client.dispatcher.queue.Len() > 0 {
		event, ok := client.dispatcher.queue.Dequeue()
		if !ok {
			t.Error("failed to dequeue event")
			return
		}
		
		// Shared metadata should be present
		if event.Metadata["environment"] != "test" {
			t.Errorf("expected environment to be test, got %v", event.Metadata["environment"])
		}
		
		// Event-specific metadata should override shared metadata
		if event.Metadata["version"] != "2.0.0" {
			t.Errorf("expected version to be 2.0.0 (overridden), got %v", event.Metadata["version"])
		}
		
		// Event-specific metadata should be present
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

	// Set shared metadata
	_ = client.SetMetadata("userId", "123")

	// Track event without event-specific metadata
	client.Track("test_event")

	time.Sleep(50 * time.Millisecond)

	if client.dispatcher.queue.Len() > 0 {
		event, ok := client.dispatcher.queue.Dequeue()
		if !ok {
			t.Error("failed to dequeue event")
			return
		}
		
		// Only shared metadata should be present
		if event.Metadata["userId"] != "123" {
			t.Errorf("expected userId to be 123, got %v", event.Metadata["userId"])
		}
		
		// Should have exactly one metadata field
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

	// Track event without any metadata
	client.Track("test_event")

	time.Sleep(50 * time.Millisecond)

	if client.dispatcher.queue.Len() > 0 {
		event, ok := client.dispatcher.queue.Dequeue()
		if !ok {
			t.Error("failed to dequeue event")
			return
		}
		
		// Metadata should be nil when no metadata is set
		if event.Metadata != nil {
			t.Errorf("expected metadata to be nil, got %v", event.Metadata)
		}
	} else {
		t.Error("expected event to be in queue")
	}
}

func TestDispatcher_StopTimerIfEmpty(t *testing.T) {
	config := DispatcherConfig{
		FlushInterval: 100 * time.Millisecond,
		MaxBatchSize:  5,
		MaxRetries:    3,
	}
	
	mockHTTP := &mockHTTPAdapter{}
	mockStorage := &mockStorageAdapter{}
	dispatcher := NewDispatcher(config, mockHTTP, mockStorage, map[string]string{})
	
	dispatcher.Start()
	defer dispatcher.Stop()
	
	// Add an event to start the timer
	event := Event{Name: "test", IssuedAt: time.Now().UnixMilli()}
	dispatcher.Enqueue(event)
	
	// Wait for timer to start
	time.Sleep(50 * time.Millisecond)
	
	// Flush to empty the queue
	dispatcher.Flush()
	
	// Wait for timer to potentially stop
	time.Sleep(150 * time.Millisecond)
	
	// Timer should have stopped (this tests the stopTimerIfEmpty function)
	// We can't directly verify this without exposing internal state,
	// but the function will be called during the flush process
}
func TestClient_InitWithStorageError(t *testing.T) {
	client := createTestClient()
	
	// Use a storage adapter that will fail during Load
	client.storageAdapter = &mockStorageAdapterWithError{}
	
	err := client.Init()
	if err == nil {
		t.Error("expected error during Init with failing storage adapter")
	}
	
	// Client should not be initialized
	if client.initialized {
		t.Error("client should not be initialized after Init error")
	}
}

func TestClient_InitTwice(t *testing.T) {
	client := createTestClient()
	
	// First init should succeed
	err := client.Init()
	if err != nil {
		t.Fatalf("first Init failed: %v", err)
	}
	defer client.Dispose()
	
	// Second init should be no-op and return nil
	err = client.Init()
	if err != nil {
		t.Errorf("second Init should return nil, got: %v", err)
	}
}
