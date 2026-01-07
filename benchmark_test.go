package ripple

import (
	"testing"
	"time"
)

// Benchmark client creation
func BenchmarkNewClient(b *testing.B) {
	config := ClientConfig{
		APIKey:         "test-key",
		Endpoint:       "http://test.com",
		HTTPAdapter:    &benchHTTPAdapter{},
		StorageAdapter: &benchStorageAdapter{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, _ := NewDefaultClient(config)
		_ = client
	}
}

// Benchmark metadata operations
func BenchmarkSetMetadata(b *testing.B) {
	client, _ := NewDefaultClient(ClientConfig{
		APIKey:         "test-key",
		Endpoint:       "http://test.com",
		HTTPAdapter:    &benchHTTPAdapter{},
		StorageAdapter: &benchStorageAdapter{},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.SetMetadata("key", "value")
	}
}

func BenchmarkGetMetadata(b *testing.B) {
	client, _ := NewDefaultClient(ClientConfig{
		APIKey:         "test-key",
		Endpoint:       "http://test.com",
		HTTPAdapter:    &benchHTTPAdapter{},
		StorageAdapter: &benchStorageAdapter{},
	})
	_ = client.SetMetadata("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.GetMetadata()
	}
}

// Benchmark event tracking
func BenchmarkTrack(b *testing.B) {
	client, _ := NewDefaultClient(ClientConfig{
		APIKey:         "test-key",
		Endpoint:       "http://test.com",
		HTTPAdapter:    &benchHTTPAdapter{},
		StorageAdapter: &benchStorageAdapter{},
	})
	client.Init()
	defer client.Dispose()

	payload := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Track("test_event", payload, nil)
	}
}

func BenchmarkTrackWithMetadata(b *testing.B) {
	client, _ := NewDefaultClient(ClientConfig{
		APIKey:         "test-key",
		Endpoint:       "http://test.com",
		HTTPAdapter:    &benchHTTPAdapter{},
		StorageAdapter: &benchStorageAdapter{},
	})
	client.Init()
	defer client.Dispose()

	payload := map[string]any{
		"key1": "value1",
		"key2": 123,
	}
	schemaVersion := "1.0.0"
	metadata := &EventMetadata{
		SchemaVersion: &schemaVersion,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Track("test_event", payload, metadata)
	}
}

// Benchmark queue operations
func BenchmarkQueueEnqueue(b *testing.B) {
	queue := NewQueue()
	event := Event{
		Name:     "test",
		Payload:  map[string]any{"key": "value"},
		IssuedAt: time.Now().UnixMilli(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.Enqueue(event)
	}
}

func BenchmarkQueueDequeue(b *testing.B) {
	queue := NewQueue()
	event := Event{
		Name:     "test",
		Payload:  map[string]any{"key": "value"},
		IssuedAt: time.Now().UnixMilli(),
	}

	// Pre-fill queue
	for i := 0; i < b.N; i++ {
		queue.Enqueue(event)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = queue.Dequeue()
	}
}

// Benchmark adapters for performance testing
type benchHTTPAdapter struct{}

func (a *benchHTTPAdapter) Send(endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error) {
	return &HTTPResponse{OK: true, Status: 200}, nil
}

type benchStorageAdapter struct{}

func (a *benchStorageAdapter) Save(events []Event) error {
	return nil
}

func (a *benchStorageAdapter) Load() ([]Event, error) {
	return nil, nil
}

func (a *benchStorageAdapter) Clear() error {
	return nil
}

// Performance regression tests
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression tests in short mode")
	}

	// Test Track performance doesn't regress below acceptable threshold
	result := testing.Benchmark(BenchmarkTrack)
	nsPerOp := result.NsPerOp()
	allocsPerOp := result.AllocsPerOp()

	// Ensure Track operations stay under 2000ns and 10 allocs
	if nsPerOp > 2000 {
		t.Errorf("Track performance regression: %d ns/op > 2000 ns/op threshold", nsPerOp)
	}
	if allocsPerOp > 10 {
		t.Errorf("Track allocation regression: %d allocs/op > 10 allocs/op threshold", allocsPerOp)
	}

	t.Logf("Track performance: %d ns/op, %d allocs/op", nsPerOp, allocsPerOp)
}
