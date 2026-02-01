package ripple

import (
	"reflect"
	"testing"
	"time"
)

// TestContractCompliance validates that all API signatures match the contract specification exactly
func TestContractCompliance(t *testing.T) {
	t.Run("Client type signature", func(t *testing.T) {
		// Verify Client type exists
		var client *Client
		clientType := reflect.TypeOf(client).Elem()

		// Check type name
		typeName := clientType.Name()
		if typeName != "Client" {
			t.Errorf("Expected Client type name, got %s", typeName)
		}

		// Verify it has fields
		numFields := clientType.NumField()
		if numFields == 0 {
			t.Error("Client should have fields")
		}
	})

	t.Run("NewClient signature", func(t *testing.T) {
		// Verify NewClient function signature
		newClientFunc := reflect.ValueOf(NewClient)
		funcType := newClientFunc.Type()

		// Should take 1 parameter (ClientConfig) and return 2 values (*Client, error)
		if funcType.NumIn() != 1 {
			t.Errorf("NewClient should take 1 parameter, got %d", funcType.NumIn())
		}
		if funcType.NumOut() != 2 {
			t.Errorf("NewClient should return 2 values, got %d", funcType.NumOut())
		}

		// Second return value should be error
		if funcType.Out(1).Name() != "error" {
			t.Errorf("NewClient second return should be error, got %s", funcType.Out(1).Name())
		}
	})

	t.Run("Required methods exist", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		clientValue := reflect.ValueOf(client)
		clientType := clientValue.Type()

		if clientType.Kind() != reflect.Ptr {
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

		// Test Init() error
		initMethod := clientValue.MethodByName("Init")
		initType := initMethod.Type()
		if initType.NumIn() != 0 || initType.NumOut() != 1 {
			t.Error("Init should take no parameters and return error")
		}

		// Test Track(string, ...any) error
		trackMethod := clientValue.MethodByName("Track")
		trackType := trackMethod.Type()
		if trackType.NumIn() != 2 || trackType.NumOut() != 1 {
			t.Error("Track should take 2 parameters (name string, args ...any) and return error")
		}
		if !trackType.IsVariadic() {
			t.Error("Track should be variadic")
		}

		// Test SetMetadata(string, any) error
		setMetadataMethod := clientValue.MethodByName("SetMetadata")
		setMetadataType := setMetadataMethod.Type()
		if setMetadataType.NumIn() != 2 || setMetadataType.NumOut() != 1 {
			t.Error("SetMetadata should take 2 parameters and return error")
		}

		// Test GetMetadata() map[string]any
		getMetadataMethod := clientValue.MethodByName("GetMetadata")
		getMetadataType := getMetadataMethod.Type()
		if getMetadataType.NumIn() != 0 || getMetadataType.NumOut() != 1 {
			t.Error("GetMetadata should take no parameters and return map[string]any")
		}

		// Test GetSessionId() *string
		getSessionIdMethod := clientValue.MethodByName("GetSessionId")
		getSessionIdType := getSessionIdMethod.Type()
		if getSessionIdType.NumIn() != 0 || getSessionIdType.NumOut() != 1 {
			t.Error("GetSessionId should take no parameters and return *string")
		}

		// Test Flush() (no return)
		flushMethod := clientValue.MethodByName("Flush")
		flushType := flushMethod.Type()
		if flushType.NumIn() != 0 || flushType.NumOut() != 0 {
			t.Error("Flush should take no parameters and return nothing")
		}

		// Test Dispose() error
		disposeMethod := clientValue.MethodByName("Dispose")
		disposeType := disposeMethod.Type()
		if disposeType.NumIn() != 0 || disposeType.NumOut() != 1 {
			t.Error("Dispose should take no parameters and return error")
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

	t.Run("Track requires Init", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		err := client.Track("test", nil, nil)
		if err == nil {
			t.Error("Track should return error when called before Init")
		}
	})

	t.Run("Event validation", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		client.Init()
		defer client.Dispose()

		// Empty name should fail
		err := client.Track("", nil, nil)
		if err == nil {
			t.Error("Track should reject empty event name")
		}

		// Long name should fail
		longName := string(make([]rune, 256))
		for i := range longName {
			longName = string([]rune(longName)[:i]) + "a" + string([]rune(longName)[i+1:])
		}
		err = client.Track(longName, nil, nil)
		if err == nil {
			t.Error("Track should reject event name > 255 characters")
		}
	})

	t.Run("Metadata validation", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())

		// Empty key should fail
		err := client.SetMetadata("", "value")
		if err == nil {
			t.Error("SetMetadata should reject empty key")
		}

		// Long key should fail
		longKey := string(make([]rune, 256))
		for i := range longKey {
			longKey = string([]rune(longKey)[:i]) + "a" + string([]rune(longKey)[i+1:])
		}
		err = client.SetMetadata(longKey, "value")
		if err == nil {
			t.Error("SetMetadata should reject key > 255 characters")
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

		// Test concurrent Track calls
		done := make(chan bool, 100)
		for i := 0; i < 100; i++ {
			go func(id int) {
				defer func() { done <- true }()
				for j := 0; j < 10; j++ {
					client.Track("concurrent_test", map[string]any{"id": id, "iteration": j}, nil)
				}
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 100; i++ {
			<-done
		}
	})

	t.Run("Concurrent metadata operations", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())

		done := make(chan bool, 50)

		// Concurrent SetMetadata
		for i := 0; i < 25; i++ {
			go func(id int) {
				defer func() { done <- true }()
				for j := 0; j < 10; j++ {
					client.SetMetadata("key"+string(rune(id)), "value")
				}
			}(i)
		}

		// Concurrent GetMetadata
		for i := 0; i < 25; i++ {
			go func() {
				defer func() { done <- true }()
				for j := 0; j < 10; j++ {
					client.GetMetadata()
				}
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 50; i++ {
			<-done
		}
	})

	t.Run("Memory stability", func(t *testing.T) {
		client, _ := NewClient(createTestConfig())
		client.Init()
		defer client.Dispose()

		// Track many events to test memory stability
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

	// Measure performance over multiple iterations
	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		client.Track("perf_test", map[string]any{"iteration": i}, nil)
	}

	duration := time.Since(start)
	avgNsPerOp := duration.Nanoseconds() / int64(iterations)

	// Should maintain reasonable performance (< 5000ns per operation under load)
	if avgNsPerOp > 5000 {
		t.Errorf("Performance degraded: %d ns/op > 5000 ns/op threshold", avgNsPerOp)
	}

	t.Logf("Performance stability: %d ns/op average over %d operations", avgNsPerOp, iterations)
}
