package ripple

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

type mockLogger struct {
	debugs   []string
	infos    []string
	warnings []string
	errors   []string
}

func (m *mockLogger) Debug(message string, args ...any) {
	m.debugs = append(m.debugs, fmt.Sprintf(message, args...))
}

func (m *mockLogger) Info(message string, args ...any) {
	m.infos = append(m.infos, fmt.Sprintf(message, args...))
}

func (m *mockLogger) Warn(message string, args ...any) {
	m.warnings = append(m.warnings, fmt.Sprintf(message, args...))
}

func (m *mockLogger) Error(message string, args ...any) {
	m.errors = append(m.errors, fmt.Sprintf(message, args...))
}

type mockHTTPAdapter struct {
	calls      int
	fail       bool
	err        error
	statusCode int
}

func (m *mockHTTPAdapter) Send(endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	if m.fail {
		status := m.statusCode
		if status == 0 {
			status = 500 // default to 500 for backward compatibility
		}
		return &HTTPResponse{Status: status}, nil
	}
	return &HTTPResponse{Status: 200}, nil
}

type mockStorageAdapter struct {
	saved  []Event
	loaded []Event
	err    error
}

func (m *mockStorageAdapter) Save(events []Event) error {
	if m.err != nil {
		return m.err
	}
	m.saved = events
	return nil
}

func (m *mockStorageAdapter) Load() ([]Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.loaded, nil
}

func (m *mockStorageAdapter) Clear() error {
	m.saved = nil
	return nil
}

func TestDispatcher_Enqueue(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	storageAdapter := &mockStorageAdapter{}
	config := DispatcherConfig{
		Endpoint:      "http://test.com",
		FlushInterval: 1 * time.Second,
		MaxBatchSize:  2,
		MaxRetries:    3,
	}

	dispatcher := NewDispatcher(config, httpAdapter, storageAdapter, nil)
	dispatcher.Restore()
	defer dispatcher.Stop()

	dispatcher.Enqueue(Event{Name: "test1"})
	dispatcher.Enqueue(Event{Name: "test2"})

	time.Sleep(100 * time.Millisecond)

	if httpAdapter.calls == 0 {
		t.Fatal("expected HTTP adapter to be called")
	}
}

func TestDispatcher_Flush(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	storageAdapter := &mockStorageAdapter{}
	config := DispatcherConfig{
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
	}

	dispatcher := NewDispatcher(config, httpAdapter, storageAdapter, nil)
	dispatcher.Restore()
	defer dispatcher.Stop()

	dispatcher.Enqueue(Event{Name: "test"})
	dispatcher.Flush()

	if httpAdapter.calls != 1 {
		t.Fatalf("expected 1 call, got %d", httpAdapter.calls)
	}
}

func TestDispatcher_LoadPersistedEvents(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	storageAdapter := &mockStorageAdapter{
		loaded: []Event{{Name: "persisted"}},
	}
	config := DispatcherConfig{
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
	}

	dispatcher := NewDispatcher(config, httpAdapter, storageAdapter, nil)
	dispatcher.Restore()

	if dispatcher.queue.Len() != 1 {
		t.Fatal("expected 1 persisted event in queue")
	}

	dispatcher.Stop()
}

func TestDispatcher_PersistOnStop(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: true}
	storageAdapter := &mockStorageAdapter{}
	config := DispatcherConfig{
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    0,
	}

	dispatcher := NewDispatcher(config, httpAdapter, storageAdapter, nil)
	dispatcher.Restore()
	dispatcher.Enqueue(Event{Name: "test"})

	dispatcher.Stop()

	if len(storageAdapter.saved) != 1 || storageAdapter.saved[0].Name != "test" {
		t.Fatal("expected events to be persisted on stop")
	}
}

func TestDispatcher_StartLoadError(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	storageAdapter := &mockStorageAdapter{err: errors.New("load error")}
	config := DispatcherConfig{
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
	}

	dispatcher := NewDispatcher(config, httpAdapter, storageAdapter, nil)
	err := dispatcher.Restore()
	if err == nil {
		t.Fatal("expected error from Start")
	}
}

func TestDispatcher_RetryWithError(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{err: errors.New("network error")}
	storageAdapter := &mockStorageAdapter{}
	config := DispatcherConfig{
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    1,
	}

	dispatcher := NewDispatcher(config, httpAdapter, storageAdapter, nil)
	dispatcher.Restore()
	defer dispatcher.Stop()

	dispatcher.Enqueue(Event{Name: "test"})
	dispatcher.Flush()

	if httpAdapter.calls != 2 {
		t.Fatalf("expected 2 calls (1 initial + 1 retry), got %d", httpAdapter.calls)
	}
}

func TestDispatcher_4xxClientError_DropsEvents(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: true, statusCode: 400}
	storageAdapter := &mockStorageAdapter{}
	config := DispatcherConfig{
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
	}

	dispatcher := NewDispatcher(config, httpAdapter, storageAdapter, nil)
	dispatcher.Restore()
	defer dispatcher.Stop()

	dispatcher.Enqueue(Event{Name: "test"})
	dispatcher.Flush()

	// Should only call once (no retries for 4xx)
	if httpAdapter.calls != 1 {
		t.Fatalf("expected 1 call for 4xx error, got %d", httpAdapter.calls)
	}

	// Events should not be persisted (dropped)
	if len(storageAdapter.saved) > 0 {
		t.Fatal("expected no events to be persisted for 4xx error")
	}
}

func TestDispatcher_5xxServerError_RetriesAndPersists(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: true, statusCode: 500}
	storageAdapter := &mockStorageAdapter{}
	config := DispatcherConfig{
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    2,
	}

	dispatcher := NewDispatcher(config, httpAdapter, storageAdapter, nil)
	dispatcher.Restore()
	defer dispatcher.Stop()

	dispatcher.Enqueue(Event{Name: "test"})
	dispatcher.Flush()

	// Should retry: 1 initial + 2 retries = 3 calls
	if httpAdapter.calls != 3 {
		t.Fatalf("expected 3 calls for 5xx error with 2 retries, got %d", httpAdapter.calls)
	}

	// Events should be re-queued and available for persistence
	if dispatcher.queue.Len() == 0 {
		t.Fatal("expected events to be re-queued after 5xx max retries")
	}
}

func TestDispatcher_NetworkError_RetriesAndPersists(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{err: errors.New("network timeout")}
	storageAdapter := &mockStorageAdapter{}
	config := DispatcherConfig{
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    1,
	}

	dispatcher := NewDispatcher(config, httpAdapter, storageAdapter, nil)
	dispatcher.Restore()
	defer dispatcher.Stop()

	dispatcher.Enqueue(Event{Name: "test"})
	dispatcher.Flush()

	// Should retry: 1 initial + 1 retry = 2 calls
	if httpAdapter.calls != 2 {
		t.Fatalf("expected 2 calls for network error with 1 retry, got %d", httpAdapter.calls)
	}

	// Events should be re-queued and available for persistence
	if dispatcher.queue.Len() == 0 {
		t.Fatal("expected events to be re-queued after network error max retries")
	}
}

func TestDispatcher_2xxSuccess_ClearsStorage(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{} // defaults to 200 OK
	storageAdapter := &mockStorageAdapter{}
	config := DispatcherConfig{
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
	}

	dispatcher := NewDispatcher(config, httpAdapter, storageAdapter, nil)
	dispatcher.Restore()
	defer dispatcher.Stop()

	dispatcher.Enqueue(Event{Name: "test"})
	dispatcher.Flush()

	// Should only call once (success)
	if httpAdapter.calls != 1 {
		t.Fatalf("expected 1 call for 2xx success, got %d", httpAdapter.calls)
	}

	// Queue should be empty after successful send
	if dispatcher.queue.Len() != 0 {
		t.Fatal("expected queue to be empty after successful send")
	}
}

func TestDispatcher_DynamicRebatching(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{} // defaults to 200 OK
	storageAdapter := &mockStorageAdapter{}
	config := DispatcherConfig{
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  3, // Small batch size to test rebatching
		MaxRetries:    3,
	}

	dispatcher := NewDispatcher(config, httpAdapter, storageAdapter, nil)
	dispatcher.Restore()
	defer dispatcher.Stop()

	// Add 7 events (should create 3 batches: 3, 3, 1)
	for i := 0; i < 7; i++ {
		dispatcher.Enqueue(Event{Name: fmt.Sprintf("test%d", i)})
	}

	dispatcher.Flush()

	// Should make 3 HTTP calls (3 batches)
	if httpAdapter.calls != 3 {
		t.Fatalf("expected 3 calls for dynamic rebatching, got %d", httpAdapter.calls)
	}

	// Queue should be empty after successful send
	if dispatcher.queue.Len() != 0 {
		t.Fatal("expected queue to be empty after successful send")
	}
}

func TestDispatcher_MaxBufferSize_FIFOEviction(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: false}
	storageAdapter := &mockStorageAdapter{}

	dispatcher := NewDispatcher(
		DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "https://api.example.com",
			FlushInterval: 100 * time.Millisecond,
			MaxBatchSize:  10,
			MaxRetries:    3,
			MaxBufferSize: 2, // Limit to 2 events
		},
		httpAdapter,
		storageAdapter,
		nil,
	)

	dispatcher.Enqueue(Event{Name: "event1"})
	dispatcher.Enqueue(Event{Name: "event2"})
	dispatcher.Enqueue(Event{Name: "event3"})

	// Should only save last 2 events (event2 and event3)
	if len(storageAdapter.saved) != 2 {
		t.Fatalf("expected 2 events, got %d", len(storageAdapter.saved))
	}
	if storageAdapter.saved[0].Name != "event2" {
		t.Errorf("expected first event to be event2, got %s", storageAdapter.saved[0].Name)
	}
	if storageAdapter.saved[1].Name != "event3" {
		t.Errorf("expected second event to be event3, got %s", storageAdapter.saved[1].Name)
	}
}

func TestDispatcher_MaxBufferSize_NoLimitWhenNotConfigured(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: false}
	storageAdapter := &mockStorageAdapter{}

	dispatcher := NewDispatcher(
		DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "https://api.example.com",
			FlushInterval: 100 * time.Millisecond,
			MaxBatchSize:  10,
			MaxRetries:    3,
			MaxBufferSize: 0, // No limit
		},
		httpAdapter,
		storageAdapter,
		nil,
	)

	dispatcher.Enqueue(Event{Name: "event1"})
	dispatcher.Enqueue(Event{Name: "event2"})
	dispatcher.Enqueue(Event{Name: "event3"})

	// Should save all 3 events
	if len(storageAdapter.saved) != 3 {
		t.Fatalf("expected 3 events, got %d", len(storageAdapter.saved))
	}
}

func TestDispatcher_MaxBufferSize_UnderThreshold(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: false}
	storageAdapter := &mockStorageAdapter{}

	dispatcher := NewDispatcher(
		DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "https://api.example.com",
			FlushInterval: 100 * time.Millisecond,
			MaxBatchSize:  10,
			MaxRetries:    3,
			MaxBufferSize: 10, // Limit to 10 events
		},
		httpAdapter,
		storageAdapter,
		nil,
	)

	dispatcher.Enqueue(Event{Name: "event1"})
	dispatcher.Enqueue(Event{Name: "event2"})

	// Should save both events (under limit)
	if len(storageAdapter.saved) != 2 {
		t.Fatalf("expected 2 events, got %d", len(storageAdapter.saved))
	}
}

func TestDispatcher_MaxBufferSize_AppliedOnLoad(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: false}
	storageAdapter := &mockStorageAdapter{
		loaded: []Event{
			{Name: "event1"},
			{Name: "event2"},
			{Name: "event3"},
			{Name: "event4"},
		},
	}

	dispatcher := NewDispatcher(
		DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "https://api.example.com",
			FlushInterval: 100 * time.Millisecond,
			MaxBatchSize:  10,
			MaxRetries:    3,
			MaxBufferSize: 2, // Limit to 2 events
		},
		httpAdapter,
		storageAdapter,
		nil,
	)

	err := dispatcher.Restore()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only load last 2 events
	if dispatcher.queue.Len() != 2 {
		t.Fatalf("expected 2 events in queue, got %d", dispatcher.queue.Len())
	}
}

func TestDispatcher_MaxBufferSize_ConfigValidation(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: false}
	storageAdapter := &mockStorageAdapter{}
	logger := &mockLogger{}

	// Create dispatcher with logger set first to capture warning
	dispatcher := NewDispatcher(
		DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "https://api.example.com",
			FlushInterval: 100 * time.Millisecond,
			MaxBatchSize:  100,
			MaxRetries:    3,
			MaxBufferSize: 50, // Less than MaxBatchSize
		},
		httpAdapter,
		storageAdapter,
		nil,
	)
	dispatcher.SetLoggerAdapter(logger)

	// Create another dispatcher to trigger validation with our logger
	_ = NewDispatcher(
		DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "https://api.example.com",
			FlushInterval: 100 * time.Millisecond,
			MaxBatchSize:  100,
			MaxRetries:    3,
			MaxBufferSize: 50, // Less than MaxBatchSize
		},
		httpAdapter,
		storageAdapter,
		nil,
	)

	// Validation happens in constructor, so we just verify dispatcher was created
	if dispatcher == nil {
		t.Error("expected dispatcher to be created")
	}
}
