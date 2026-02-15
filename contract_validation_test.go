package ripple

import (
	"reflect"
	"testing"
	"time"
)

// TestContractCompliance validates that all API signatures match the contract specification exactly
func TestContractCompliance(t *testing.T) {
	t.Run("Client type signature", func(t *testing.T) {
		var client *Client
		clientType := reflect.TypeOf(client).Elem()

		if clientType.Name() != "Client" {
			t.Errorf("Expected Client type name, got %s", clientType.Name())
		}

		if clientType.NumField() == 0 {
			t.Error("Client should have fields")
		}
	})

	t.Run("NewClient signature", func(t *testing.T) {
		newClientFunc := reflect.ValueOf(NewClient)
		funcType := newClientFunc.Type()

		if funcType.NumIn() != 1 {
			t.Errorf("NewClient should take 1 parameter, got %d", funcType.NumIn())
		}
		if funcType.NumOut() != 2 {
			t.Errorf("NewClient should return 2 values, got %d", funcType.NumOut())
		}
		if funcType.Out(1).Name() != "error" {
			t.Errorf("NewClient second return should be error, got %s", funcType.Out(1).Name())
		}
	})

	t.Run("Required methods exist", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		clientValue := reflect.ValueOf(client)

		if clientValue.Type().Kind() != reflect.Ptr {
			t.Error("Client should be a pointer type")
		}

		requiredMethods := []string{
			"Init",
			"Track",
			"SetMetadata",
			"GetMetadata",
			"GetSessionId",
			"Flush",
			"Dispose",
			"Close",
		}

		for _, methodName := range requiredMethods {
			method := clientValue.MethodByName(methodName)
			if !method.IsValid() {
				t.Errorf("Required method %s not found", methodName)
			}
		}
	})

	t.Run("Method signatures", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		clientValue := reflect.ValueOf(client)

		// Init() error
		initType := clientValue.MethodByName("Init").Type()
		if initType.NumIn() != 0 || initType.NumOut() != 1 {
			t.Error("Init should take no parameters and return error")
		}

		// Track(string, ...any) error
		trackType := clientValue.MethodByName("Track").Type()
		if trackType.NumIn() != 2 || trackType.NumOut() != 1 {
			t.Error("Track should take 2 parameters (name string, args ...any) and return error")
		}
		if !trackType.IsVariadic() {
			t.Error("Track should be variadic")
		}

		// SetMetadata(string, any) — no return
		setMetadataType := clientValue.MethodByName("SetMetadata").Type()
		if setMetadataType.NumIn() != 2 || setMetadataType.NumOut() != 0 {
			t.Error("SetMetadata should take 2 parameters and return nothing")
		}

		// GetMetadata() map[string]any
		getMetadataType := clientValue.MethodByName("GetMetadata").Type()
		if getMetadataType.NumIn() != 0 || getMetadataType.NumOut() != 1 {
			t.Error("GetMetadata should take no parameters and return map[string]any")
		}

		// GetSessionId() *string
		getSessionIdType := clientValue.MethodByName("GetSessionId").Type()
		if getSessionIdType.NumIn() != 0 || getSessionIdType.NumOut() != 1 {
			t.Error("GetSessionId should take no parameters and return *string")
		}

		// Flush() — no return
		flushType := clientValue.MethodByName("Flush").Type()
		if flushType.NumIn() != 0 || flushType.NumOut() != 0 {
			t.Error("Flush should take no parameters and return nothing")
		}

		// Dispose() — no return
		disposeType := clientValue.MethodByName("Dispose").Type()
		if disposeType.NumIn() != 0 || disposeType.NumOut() != 0 {
			t.Error("Dispose should take no parameters and return nothing")
		}
	})
}

// TestEventStructCompliance validates Event struct matches contract
func TestEventStructCompliance(t *testing.T) {
	event := Event{}
	eventType := reflect.TypeOf(event)

	requiredFields := map[string]string{
		"Name":      "string",
		"Payload":   "map[string]interface {}",
		"IssuedAt":  "int64",
		"SessionID": "*string",
		"Metadata":  "map[string]interface {}",
		"Platform":  "*adapters.Platform",
	}

	for fieldName, expectedType := range requiredFields {
		field, found := eventType.FieldByName(fieldName)
		if !found {
			t.Errorf("Required field %s not found in Event struct", fieldName)
			continue
		}

		actualType := field.Type.String()
		if actualType != expectedType {
			t.Errorf("Field %s has type %s, expected %s", fieldName, actualType, expectedType)
		}
	}
}

// TestContractBehavior validates behavior matches contract requirements
func TestContractBehavior(t *testing.T) {
	t.Run("GetSessionId returns nil for server", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		sessionId := client.GetSessionId()
		if sessionId != nil {
			t.Error("GetSessionId should return nil for server environments")
		}
	})

	t.Run("GetMetadata returns empty map when no metadata", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		metadata := client.GetMetadata()
		if metadata == nil {
			t.Error("GetMetadata should return empty map, not nil")
		}
		if len(metadata) != 0 {
			t.Error("GetMetadata should return empty map when no metadata set")
		}
	})

	t.Run("Track auto-initializes", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		defer client.Dispose()

		err := client.Track("test", nil, nil)
		if err != nil {
			t.Errorf("Track should auto-initialize, got error: %v", err)
		}
	})

	t.Run("Track silently drops after dispose", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		client.Init()
		client.Dispose()

		err := client.Track("test", nil, nil)
		if err != nil {
			t.Errorf("Track after dispose should return nil, got: %v", err)
		}
	})
}

// TestReliability performs reliability and stress testing
func TestReliability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping reliability tests in short mode")
	}

	t.Run("Concurrent operations", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		client.Init()
		defer client.Dispose()

		done := make(chan bool, 100)
		for i := 0; i < 100; i++ {
			go func(id int) {
				defer func() { done <- true }()
				for j := 0; j < 10; j++ {
					client.Track("concurrent_test", map[string]any{"id": id, "iteration": j}, nil)
				}
			}(i)
		}

		for i := 0; i < 100; i++ {
			<-done
		}
	})

	t.Run("Concurrent metadata operations", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())

		done := make(chan bool, 50)

		for i := 0; i < 25; i++ {
			go func(id int) {
				defer func() { done <- true }()
				for j := 0; j < 10; j++ {
					client.SetMetadata("key"+string(rune(id)), "value")
				}
			}(i)
		}

		for i := 0; i < 25; i++ {
			go func() {
				defer func() { done <- true }()
				for j := 0; j < 10; j++ {
					client.GetMetadata()
				}
			}()
		}

		for i := 0; i < 50; i++ {
			<-done
		}
	})

	t.Run("Memory stability", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		client.Init()
		defer client.Dispose()

		for i := 0; i < 1000; i++ {
			client.Track("memory_test", map[string]any{
				"iteration": i,
				"data":      "test data for memory stability",
			}, nil)

			if i%100 == 0 {
				client.Flush()
			}
		}
	})
}

// TestPerformanceStability ensures performance remains stable under load
func TestPerformanceStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance stability tests in short mode")
	}

	client, _ := NewClient(createTestConfig())
	client.Init()
	defer client.Dispose()

	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		client.Track("perf_test", map[string]any{"iteration": i}, nil)
	}

	duration := time.Since(start)
	avgNsPerOp := duration.Nanoseconds() / int64(iterations)

	if avgNsPerOp > 50000 {
		t.Errorf("Performance degraded: %d ns/op > 50000 ns/op threshold", avgNsPerOp)
	}

	t.Logf("Performance stability: %d ns/op average over %d operations", avgNsPerOp, iterations)
}
