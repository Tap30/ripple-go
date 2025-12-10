package ripple

import (
	"errors"
	"testing"
	"time"
)

type mockHTTPAdapter struct {
	calls int
	fail  bool
	err   error
}

func (m *mockHTTPAdapter) Send(endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	if m.fail {
		return &HTTPResponse{OK: false, Status: 500}, nil
	}
	return &HTTPResponse{OK: true, Status: 200}, nil
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
	dispatcher.Start()
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
	dispatcher.Start()
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
	dispatcher.Start()

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
	dispatcher.Start()
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
	err := dispatcher.Start()
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
	dispatcher.Start()
	defer dispatcher.Stop()

	dispatcher.Enqueue(Event{Name: "test"})
	dispatcher.Flush()

	if httpAdapter.calls != 2 {
		t.Fatalf("expected 2 calls (1 initial + 1 retry), got %d", httpAdapter.calls)
	}
}

func TestDispatcher_MinFunction(t *testing.T) {
	if min(5, 3) != 3 {
		t.Fatal("expected min(5, 3) = 3")
	}
	if min(2, 8) != 2 {
		t.Fatal("expected min(2, 8) = 2")
	}
}
