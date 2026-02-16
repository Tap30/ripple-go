package ripple

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

type mockLogger struct {
	mu        sync.Mutex
	debugs    []string
	infos     []string
	warnings  []string
	errs      []string
	warnCount int
	errCount  int
}

func (m *mockLogger) Debug(message string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugs = append(m.debugs, fmt.Sprintf(message, args...))
}

func (m *mockLogger) Info(message string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infos = append(m.infos, fmt.Sprintf(message, args...))
}

func (m *mockLogger) Warn(message string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.warnCount++
	m.warnings = append(m.warnings, fmt.Sprintf(message, args...))
}

func (m *mockLogger) Error(message string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errCount++
	m.errs = append(m.errs, fmt.Sprintf(message, args...))
}

type mockHTTPAdapter struct {
	mu           sync.Mutex
	calls        int
	fail         bool
	err          error
	statusCode   int
	networkError bool
}

func (m *mockHTTPAdapter) Send(endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error) {
	return m.SendWithContext(context.Background(), endpoint, events, headers)
}

func (m *mockHTTPAdapter) SendWithContext(ctx context.Context, endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error) {
	m.mu.Lock()
	m.calls++
	fail := m.fail
	err := m.err
	statusCode := m.statusCode
	networkError := m.networkError
	m.mu.Unlock()

	if err != nil {
		return nil, err
	}
	if fail {
		if networkError {
			return nil, &HTTPError{Status: 0}
		}
		status := statusCode
		if status == 0 {
			status = 500
		}
		return &HTTPResponse{Status: status}, nil
	}
	// Return custom status code if set, otherwise 200
	if statusCode != 0 {
		return &HTTPResponse{Status: statusCode}, nil
	}
	return &HTTPResponse{Status: 200}, nil
}

func (m *mockHTTPAdapter) getCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

type mockStorageAdapter struct {
	mu         sync.Mutex
	saved      []Event
	loaded     []Event
	err        error
	clearCalls int
	clearErr   error
	closeErr   error
	closeCalls int
}

func (m *mockStorageAdapter) Save(events []Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.saved = events
	return nil
}

func (m *mockStorageAdapter) Load() ([]Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	return m.loaded, nil
}

func (m *mockStorageAdapter) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clearCalls++
	if m.clearErr != nil {
		return m.clearErr
	}
	m.saved = nil
	return nil
}

func (m *mockStorageAdapter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalls++
	return m.closeErr
}

func (m *mockStorageAdapter) getSaved() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Event, len(m.saved))
	copy(result, m.saved)
	return result
}

func newTestDispatcher(httpAdapter *mockHTTPAdapter, storageAdapter *mockStorageAdapter) *Dispatcher {
	return NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
	}, httpAdapter, storageAdapter, &mockLogger{})
}

func TestDispatcher_Enqueue(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	storageAdapter := &mockStorageAdapter{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 1 * time.Second,
		MaxBatchSize:  2,
		MaxRetries:    3,
	}, httpAdapter, storageAdapter, &mockLogger{})

	d.Restore()
	defer d.Dispose()

	d.Enqueue(Event{Name: "test1"})
	d.Enqueue(Event{Name: "test2"})

	time.Sleep(100 * time.Millisecond)

	if httpAdapter.getCalls() == 0 {
		t.Fatal("expected HTTP adapter to be called")
	}
}

func TestDispatcher_Flush(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	storageAdapter := &mockStorageAdapter{}
	d := newTestDispatcher(httpAdapter, storageAdapter)

	d.Restore()
	defer d.Dispose()

	d.Enqueue(Event{Name: "test"})
	d.Flush()

	if httpAdapter.getCalls() != 1 {
		t.Fatalf("expected 1 call, got %d", httpAdapter.getCalls())
	}
}

func TestDispatcher_LoadPersistedEvents(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	storageAdapter := &mockStorageAdapter{
		loaded: []Event{{Name: "persisted"}},
	}
	d := newTestDispatcher(httpAdapter, storageAdapter)

	d.Restore()

	if d.queue.Len() != 1 {
		t.Fatal("expected 1 persisted event in queue")
	}

	d.Dispose()
}

func TestDispatcher_RestoreLogsError(t *testing.T) {
	logger := &mockLogger{}
	storageAdapter := &mockStorageAdapter{err: errors.New("load error")}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
	}, &mockHTTPAdapter{}, storageAdapter, logger)

	// Restore should NOT return error — it logs and continues
	d.Restore()

	if len(logger.errs) == 0 {
		t.Fatal("expected error to be logged")
	}
}

func TestDispatcher_EnqueueAfterDispose(t *testing.T) {
	logger := &mockLogger{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
	}, &mockHTTPAdapter{}, &mockStorageAdapter{}, logger)

	d.Dispose()
	d.Enqueue(Event{Name: "test"})

	if d.queue.Len() != 0 {
		t.Fatal("expected queue to be empty after dispose")
	}
	if len(logger.warnings) == 0 {
		t.Fatal("expected warning about enqueue after dispose")
	}
}

func TestDispatcher_RetryWithError(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{err: errors.New("network error")}
	storageAdapter := &mockStorageAdapter{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    1,
	}, httpAdapter, storageAdapter, &mockLogger{})

	d.Restore()
	defer d.Dispose()

	d.Enqueue(Event{Name: "test"})
	d.Flush()

	if httpAdapter.getCalls() != 2 {
		t.Fatalf("expected 2 calls (1 initial + 1 retry), got %d", httpAdapter.getCalls())
	}
}

func TestDispatcher_4xxClientError_DropsEvents(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: true, statusCode: 400}
	storageAdapter := &mockStorageAdapter{}
	d := newTestDispatcher(httpAdapter, storageAdapter)

	d.Restore()
	defer d.Dispose()

	d.Enqueue(Event{Name: "test"})
	d.Flush()

	if httpAdapter.getCalls() != 1 {
		t.Fatalf("expected 1 call for 4xx error, got %d", httpAdapter.getCalls())
	}

	saved := storageAdapter.getSaved()
	if len(saved) > 0 {
		t.Fatal("expected no events to be persisted for 4xx error")
	}
}

func TestDispatcher_5xxServerError_RetriesAndPersists(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: true, statusCode: 500}
	storageAdapter := &mockStorageAdapter{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    2,
	}, httpAdapter, storageAdapter, &mockLogger{})

	d.Restore()
	defer d.Dispose()

	d.Enqueue(Event{Name: "test"})
	d.Flush()

	if httpAdapter.getCalls() != 3 {
		t.Fatalf("expected 3 calls for 5xx error with 2 retries, got %d", httpAdapter.getCalls())
	}

	if d.queue.Len() == 0 {
		t.Fatal("expected events to be re-queued after 5xx max retries")
	}
}

func TestDispatcher_NetworkError_RetriesAndPersists(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{err: errors.New("network timeout")}
	storageAdapter := &mockStorageAdapter{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    1,
	}, httpAdapter, storageAdapter, &mockLogger{})

	d.Restore()
	defer d.Dispose()

	d.Enqueue(Event{Name: "test"})
	d.Flush()

	if httpAdapter.getCalls() != 2 {
		t.Fatalf("expected 2 calls for network error with 1 retry, got %d", httpAdapter.getCalls())
	}

	if d.queue.Len() == 0 {
		t.Fatal("expected events to be re-queued after network error max retries")
	}
}

func TestDispatcher_2xxSuccess_ClearsStorage(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	storageAdapter := &mockStorageAdapter{}
	d := newTestDispatcher(httpAdapter, storageAdapter)

	d.Restore()
	defer d.Dispose()

	d.Enqueue(Event{Name: "test"})
	d.Flush()

	if httpAdapter.getCalls() != 1 {
		t.Fatalf("expected 1 call for 2xx success, got %d", httpAdapter.getCalls())
	}

	if d.queue.Len() != 0 {
		t.Fatal("expected queue to be empty after successful send")
	}
}

func TestDispatcher_DynamicRebatching(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	storageAdapter := &mockStorageAdapter{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  3,
		MaxRetries:    3,
	}, httpAdapter, storageAdapter, &mockLogger{})

	d.Restore()
	defer d.Dispose()

	for i := 0; i < 7; i++ {
		d.Enqueue(Event{Name: fmt.Sprintf("test%d", i)})
	}

	d.Flush()

	if httpAdapter.getCalls() < 3 {
		t.Fatalf("expected at least 3 calls for dynamic rebatching, got %d", httpAdapter.getCalls())
	}

	if d.queue.Len() != 0 {
		t.Fatal("expected queue to be empty after successful send")
	}
}

func TestDispatcher_MaxBufferSize_FIFOEviction(t *testing.T) {
	storageAdapter := &mockStorageAdapter{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 100 * time.Millisecond,
		MaxBatchSize:  10,
		MaxRetries:    3,
		MaxBufferSize: 2,
	}, &mockHTTPAdapter{}, storageAdapter, &mockLogger{})
	defer d.Dispose()

	d.Enqueue(Event{Name: "event1"})
	d.Enqueue(Event{Name: "event2"})
	d.Enqueue(Event{Name: "event3"})

	saved := storageAdapter.getSaved()
	if len(saved) != 2 {
		t.Fatalf("expected 2 events, got %d", len(saved))
	}
	if saved[0].Name != "event2" {
		t.Errorf("expected first event to be event2, got %s", saved[0].Name)
	}
	if saved[1].Name != "event3" {
		t.Errorf("expected second event to be event3, got %s", saved[1].Name)
	}
}

func TestDispatcher_MaxBufferSize_NoLimitWhenNotConfigured(t *testing.T) {
	storageAdapter := &mockStorageAdapter{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 100 * time.Millisecond,
		MaxBatchSize:  10,
		MaxRetries:    3,
		MaxBufferSize: 0,
	}, &mockHTTPAdapter{}, storageAdapter, &mockLogger{})
	defer d.Dispose()

	d.Enqueue(Event{Name: "event1"})
	d.Enqueue(Event{Name: "event2"})
	d.Enqueue(Event{Name: "event3"})

	saved := storageAdapter.getSaved()
	if len(saved) != 3 {
		t.Fatalf("expected 3 events, got %d", len(saved))
	}
}

func TestDispatcher_MaxBufferSize_AppliedOnLoad(t *testing.T) {
	storageAdapter := &mockStorageAdapter{
		loaded: []Event{
			{Name: "event1"},
			{Name: "event2"},
			{Name: "event3"},
			{Name: "event4"},
		},
	}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 100 * time.Millisecond,
		MaxBatchSize:  10,
		MaxRetries:    3,
		MaxBufferSize: 2,
	}, &mockHTTPAdapter{}, storageAdapter, &mockLogger{})

	d.Restore()

	if d.queue.Len() != 2 {
		t.Fatalf("expected 2 events in queue, got %d", d.queue.Len())
	}
}

func TestDispatcher_ConcurrentFlush(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 1 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
		MaxBufferSize: 100,
	}, httpAdapter, &mockStorageAdapter{}, &mockLogger{})

	for i := 0; i < 20; i++ {
		d.Enqueue(Event{Name: fmt.Sprintf("event_%d", i)})
	}

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			d.Flush()
			done <- true
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	if httpAdapter.getCalls() == 0 {
		t.Error("expected HTTP adapter to be called")
	}

	d.Dispose()
}

func TestDispatcher_NoTimerLeak(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	storageAdapter := &mockStorageAdapter{}

	for i := 0; i < 100; i++ {
		d := NewDispatcher(DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "http://test.com",
			FlushInterval: 10 * time.Millisecond,
			MaxBatchSize:  10,
			MaxRetries:    3,
			MaxBufferSize: 100,
		}, httpAdapter, storageAdapter, &mockLogger{})

		d.Enqueue(Event{Name: "test"})
		d.Dispose()
	}

	time.Sleep(50 * time.Millisecond)
}

func TestDispatcher_DisposeAbortsRetries(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: true, statusCode: 500}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    10, // high retries so dispose can interrupt
	}, httpAdapter, &mockStorageAdapter{}, &mockLogger{})

	d.Enqueue(Event{Name: "test"})

	// Start flush in background
	done := make(chan struct{})
	go func() {
		d.Flush()
		close(done)
	}()

	// Give flush time to start
	time.Sleep(50 * time.Millisecond)

	// Dispose should abort retries
	d.Dispose()

	// Flush should complete quickly after dispose
	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("flush did not complete after dispose — retries were not aborted")
	}
}

func TestDispatcher_OneShotTimer(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 50 * time.Millisecond,
		MaxBatchSize:  100,
		MaxRetries:    3,
	}, httpAdapter, &mockStorageAdapter{}, &mockLogger{})

	d.Restore()
	defer d.Dispose()

	d.Enqueue(Event{Name: "test"})

	// Wait for timer to fire and flush to complete
	time.Sleep(200 * time.Millisecond)

	calls := httpAdapter.getCalls()
	if calls != 1 {
		t.Fatalf("expected 1 flush from timer, got %d", calls)
	}

	// Timer should be nil now (one-shot), no more flushes
	time.Sleep(200 * time.Millisecond)

	calls = httpAdapter.getCalls()
	if calls != 1 {
		t.Fatalf("expected still 1 flush (one-shot timer), got %d", calls)
	}
}

func TestDispatcher_HandleUnexpectedStatusCode(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{statusCode: 150} // unexpected status < 200
	storageAdapter := &mockStorageAdapter{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
	}, httpAdapter, storageAdapter, &mockLogger{})

	d.Enqueue(Event{Name: "test"})
	d.Flush()

	if storageAdapter.clearCalls != 1 {
		t.Errorf("expected storage to be cleared for unexpected status, got %d clear calls", storageAdapter.clearCalls)
	}
}

func TestDispatcher_CalculateBackoffCap(t *testing.T) {
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    10,
	}, &mockHTTPAdapter{}, &mockStorageAdapter{}, &mockLogger{})

	// High attempt should cap at 30s
	backoff := d.calculateBackoff(10)
	if backoff > 31*time.Second { // 30s + max jitter (1s)
		t.Errorf("expected backoff <= 31s, got %v", backoff)
	}
	if backoff < 30*time.Second {
		t.Errorf("expected backoff >= 30s, got %v", backoff)
	}
}

func TestDispatcher_NetworkErrorContextCancellation(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: true, networkError: true}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    5,
	}, httpAdapter, &mockStorageAdapter{}, &mockLogger{})

	d.Enqueue(Event{Name: "test"})

	done := make(chan struct{})
	go func() {
		d.Flush()
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	d.Dispose()

	select {
	case <-done:
		// success - context cancellation stopped retries
	case <-time.After(3 * time.Second):
		t.Fatal("flush did not complete after dispose during network error retries")
	}
}

func TestDispatcher_ClearStorageErrors(t *testing.T) {
	t.Run("clear error on 2xx success", func(t *testing.T) {
		httpAdapter := &mockHTTPAdapter{statusCode: 200}
		storageAdapter := &mockStorageAdapter{clearErr: errors.New("clear failed")}
		logger := &mockLogger{}
		d := NewDispatcher(DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "http://test.com",
			FlushInterval: 10 * time.Second,
			MaxBatchSize:  10,
			MaxRetries:    3,
		}, httpAdapter, storageAdapter, logger)

		d.Restore()
		d.Enqueue(Event{Name: "test"})
		d.Flush()

		if storageAdapter.clearCalls != 1 {
			t.Errorf("expected 1 clear call, got %d", storageAdapter.clearCalls)
		}
		if logger.errCount == 0 {
			t.Error("expected error log for clear failure")
		}
	})

	t.Run("clear error on 4xx client error", func(t *testing.T) {
		httpAdapter := &mockHTTPAdapter{statusCode: 400}
		storageAdapter := &mockStorageAdapter{clearErr: errors.New("clear failed")}
		logger := &mockLogger{}
		d := NewDispatcher(DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "http://test.com",
			FlushInterval: 10 * time.Second,
			MaxBatchSize:  10,
			MaxRetries:    3,
		}, httpAdapter, storageAdapter, logger)

		d.Restore()
		d.Enqueue(Event{Name: "test"})
		d.Flush()

		if storageAdapter.clearCalls != 1 {
			t.Errorf("expected 1 clear call, got %d", storageAdapter.clearCalls)
		}
		if logger.errCount == 0 {
			t.Error("expected error log for clear failure")
		}
	})

	t.Run("clear error on unexpected status", func(t *testing.T) {
		httpAdapter := &mockHTTPAdapter{statusCode: 150}
		storageAdapter := &mockStorageAdapter{clearErr: errors.New("clear failed")}
		logger := &mockLogger{}
		d := NewDispatcher(DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "http://test.com",
			FlushInterval: 10 * time.Second,
			MaxBatchSize:  10,
			MaxRetries:    3,
		}, httpAdapter, storageAdapter, logger)

		d.Restore()
		d.Enqueue(Event{Name: "test"})
		d.Flush()

		if storageAdapter.clearCalls != 1 {
			t.Errorf("expected 1 clear call, got %d", storageAdapter.clearCalls)
		}
		if logger.errCount == 0 {
			t.Error("expected error log for clear failure")
		}
	})

	t.Run("clear error on 3xx redirect (unexpected status)", func(t *testing.T) {
		httpAdapter := &mockHTTPAdapter{statusCode: 301}
		storageAdapter := &mockStorageAdapter{clearErr: errors.New("clear failed")}
		logger := &mockLogger{}
		d := NewDispatcher(DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "http://test.com",
			FlushInterval: 10 * time.Second,
			MaxBatchSize:  10,
			MaxRetries:    3,
		}, httpAdapter, storageAdapter, logger)

		d.Restore()
		d.Enqueue(Event{Name: "test"})
		d.Flush()

		httpAdapter.mu.Lock()
		httpCalls := httpAdapter.calls
		httpAdapter.mu.Unlock()

		logger.mu.Lock()
		warnCount := logger.warnCount
		errCount := logger.errCount
		warnings := logger.warnings
		logger.mu.Unlock()

		clearCalls := storageAdapter.clearCalls

		if httpCalls == 0 {
			t.Error("expected HTTP adapter to be called")
		}
		if clearCalls != 1 {
			t.Errorf("expected 1 clear call, got %d", clearCalls)
		}
		if warnCount == 0 {
			t.Errorf("expected warn log for unexpected status, got %d warnings: %v", warnCount, warnings)
		}
		if errCount == 0 {
			t.Errorf("expected error log for clear failure, got %d errors", errCount)
		}
	})
}

func TestDispatcher_RequeueWithStorageError(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{fail: true, statusCode: 500}
	storageAdapter := &mockStorageAdapter{err: errors.New("save failed")}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    1,
	}, httpAdapter, storageAdapter, &mockLogger{})

	d.Enqueue(Event{Name: "test"})
	d.Flush()

	// Should attempt to save after requeue despite error
	if len(storageAdapter.getSaved()) != 0 {
		t.Error("storage should not have saved events due to error")
	}
}

func TestDispatcher_LogStorageErrorWithQuotaExceeded(t *testing.T) {
	httpAdapter := &mockHTTPAdapter{}
	storageAdapter := &mockStorageAdapter{err: &StorageQuotaExceededError{Message: "quota exceeded"}}
	logger := &mockLogger{}
	d := NewDispatcher(DispatcherConfig{
		APIKey:        "test-key",
		APIKeyHeader:  "X-API-Key",
		Endpoint:      "http://test.com",
		FlushInterval: 10 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
	}, httpAdapter, storageAdapter, logger)

	d.Enqueue(Event{Name: "test"})

	// Storage error should be logged as warning for quota errors
	if logger.warnCount == 0 {
		t.Error("expected warning log for quota exceeded error")
	}
}

func TestDispatcher_CloseStorageOnDispose(t *testing.T) {
	t.Run("should call Close on storage adapter", func(t *testing.T) {
		httpAdapter := &mockHTTPAdapter{}
		storageAdapter := &mockStorageAdapter{}
		logger := &mockLogger{}
		d := NewDispatcher(DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "http://test.com",
			FlushInterval: 10 * time.Second,
			MaxBatchSize:  10,
			MaxRetries:    3,
		}, httpAdapter, storageAdapter, logger)

		d.Restore()
		d.Dispose()

		if storageAdapter.closeCalls != 1 {
			t.Errorf("expected 1 close call, got %d", storageAdapter.closeCalls)
		}
	})

	t.Run("should log error if Close fails", func(t *testing.T) {
		httpAdapter := &mockHTTPAdapter{}
		storageAdapter := &mockStorageAdapter{closeErr: errors.New("close failed")}
		logger := &mockLogger{}
		d := NewDispatcher(DispatcherConfig{
			APIKey:        "test-key",
			APIKeyHeader:  "X-API-Key",
			Endpoint:      "http://test.com",
			FlushInterval: 10 * time.Second,
			MaxBatchSize:  10,
			MaxRetries:    3,
		}, httpAdapter, storageAdapter, logger)

		d.Restore()
		d.Dispose()

		if storageAdapter.closeCalls != 1 {
			t.Errorf("expected 1 close call, got %d", storageAdapter.closeCalls)
		}

		if logger.errCount == 0 {
			t.Error("expected error log for close failure")
		}
	})
}
