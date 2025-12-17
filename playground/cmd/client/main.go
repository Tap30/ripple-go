package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	ripple "github.com/Tap30/ripple-go"
	"github.com/Tap30/ripple-go/adapters"
)

func stringPtr(s string) *string {
	return &s
}

var client *ripple.Client
var scanner *bufio.Scanner
var contextCounter int
var eventCounter int

func main() {
	scanner = bufio.NewScanner(os.Stdin)

	client = ripple.NewClient(ripple.ClientConfig{
		APIKey:        "test-api-key",
		Endpoint:      "http://localhost:3000/events",
		FlushInterval: 5 * time.Second,
		MaxBatchSize:  5,
		MaxRetries:    3,
		Adapters: struct {
			HTTPAdapter    ripple.HTTPAdapter
			StorageAdapter ripple.StorageAdapter
			LoggerAdapter  ripple.LoggerAdapter
		}{
			HTTPAdapter:    adapters.NewNetHTTPAdapter(),
			StorageAdapter: adapters.NewFileStorageAdapter("ripple_events.json"),
			LoggerAdapter:  adapters.NewPrintLoggerAdapter(adapters.LogLevelInfo),
		},
	})

	if err := client.Init(); err != nil {
		fmt.Printf("❌ Failed to initialize client: %v\n", err)
		return
	}

	fmt.Println("🎯 Ripple Interactive Client")
	fmt.Println("Connected to: http://localhost:3000/events")
	fmt.Println()

	for {
		showMenu()
		choice := readInput("Choose an option: ")

		switch choice {
		case "1":
			trackSimpleEvent()
		case "2":
			trackEventWithPayload()
		case "3":
			trackEventWithMetadata()
		case "4":
			trackEventWithCustomMetadata()
		case "5":
			setSharedMetadata()
		case "6":
			trackWithSharedMetadata()
		case "7":
			viewContext()
		case "8":
			trackMultipleEvents()
		case "9":
			flush()
		case "10":
			trackEventWithError()
		case "11":
			testInvalidEndpoint()
		case "12":
			disposeClient()
		case "13":
			fmt.Println("👋 Goodbye!")
			// Persist events to storage without flushing to server
			client.DisposeWithoutFlush()
			return
		default:
			fmt.Println("❌ Invalid option. Please try again.\n")
		}
	}
}

func showMenu() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Basic Event Tracking")
	fmt.Println("1. Track Simple Event")
	fmt.Println("2. Track Event with Payload")
	fmt.Println("3. Track Event with Metadata")
	fmt.Println("4. Track Event with Custom Metadata")
	fmt.Println()
	fmt.Println("🏷️  Metadata Management")
	fmt.Println("5. Set Shared Metadata")
	fmt.Println("6. Track with Shared Metadata")
	fmt.Println("7. View Current Context/Metadata")
	fmt.Println()
	fmt.Println("📦 Batch and Flush")
	fmt.Println("8. Track Multiple Events (Batch Test)")
	fmt.Println("9. Manual Flush")
	fmt.Println()
	fmt.Println("⚠️  Error Handling")
	fmt.Println("10. Test Retry Logic (Error Event)")
	fmt.Println("11. Test Invalid Endpoint")
	fmt.Println()
	fmt.Println("🔄 Lifecycle Management")
	fmt.Println("12. Dispose Client")
	fmt.Println("13. Exit")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func readInput(prompt string) string {
	fmt.Print(prompt)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func trackSimpleEvent() {
	fmt.Println("\n📊 Track Simple Event")
	client.Track("button_click", nil, nil)
	fmt.Println("✅ Tracked: button_click\n")
}

func trackEventWithPayload() {
	fmt.Println("\n📊 Track Event with Payload")
	payload := map[string]interface{}{
		"action":    "click",
		"target":    "button",
		"timestamp": time.Now().Unix(),
	}
	client.Track("user_action", payload, nil)
	fmt.Println("✅ Tracked: user_action with payload\n")
}

func trackEventWithMetadata() {
	fmt.Println("\n📊 Track Event with Metadata")
	payload := map[string]interface{}{
		"formId": "contact-form",
		"fields": 5,
	}
	metadata := &ripple.EventMetadata{SchemaVersion: stringPtr("1.0.0")}
	client.Track("form_submit", payload, metadata)
	fmt.Println("✅ Tracked: form_submit with metadata\n")
}

func trackEventWithCustomMetadata() {
	fmt.Println("\n📊 Track Event with Custom Metadata")
	payload := map[string]interface{}{
		"orderId": "order-123",
		"amount":  99.99,
	}
	metadata := &ripple.EventMetadata{SchemaVersion: stringPtr("2.1.0")}
	client.Track("purchase_completed", payload, metadata)
	fmt.Println("✅ Tracked: purchase_completed with rich metadata\n")
}

func setSharedMetadata() {
	fmt.Println("\n🏷️  Set Shared Metadata")
	contextCounter++
	key := fmt.Sprintf("key_%d", contextCounter)
	value := fmt.Sprintf("value_%d", contextCounter)

	client.SetMetadata(key, value)
	fmt.Printf("✅ Shared metadata set: %s = %s\n\n", key, value)
}

func trackWithSharedMetadata() {
	fmt.Println("\n🏷️  Track with Shared Metadata")
	client.Track("metadata_test", nil, nil)
	fmt.Println("✅ Tracked event with shared metadata\n")
}

func trackMultipleEvents() {
	fmt.Println("\n📦 Track Multiple Events (Batch Test)")
	for i := 0; i < 10; i++ {
		payload := map[string]interface{}{"index": i}
		client.Track("batch_event", payload, nil)
	}
	fmt.Println("✅ Tracked 10 events (should auto-flush at batch size 5)\n")
}

func testInvalidEndpoint() {
	fmt.Println("\n⚠️  Test Invalid Endpoint")

	// Create a new client with invalid endpoint
	errorClient := ripple.NewClient(ripple.ClientConfig{
		APIKey:        "test-key",
		Endpoint:      "http://localhost:9999/invalid",
		FlushInterval: 5 * time.Second,
		MaxBatchSize:  5,
		MaxRetries:    2,
		Adapters: struct {
			HTTPAdapter    ripple.HTTPAdapter
			StorageAdapter ripple.StorageAdapter
			LoggerAdapter  ripple.LoggerAdapter
		}{
			HTTPAdapter:    adapters.NewNetHTTPAdapter(),
			StorageAdapter: adapters.NewFileStorageAdapter("error_events.json"),
			LoggerAdapter:  adapters.NewPrintLoggerAdapter(adapters.LogLevelWarn),
		},
	})

	if err := errorClient.Init(); err != nil {
		fmt.Printf("❌ Failed to init error client: %v\n\n", err)
		return
	}

	errorClient.Track("error_test", map[string]interface{}{"shouldFail": true}, nil)
	fmt.Println("✅ Tracked event to invalid endpoint (check console for retries)\n")
}

func disposeClient() {
	fmt.Println("\n🔄 Dispose Client")
	client.Dispose()
	fmt.Println("✅ Client disposed\n")
}

func setContext() {
	fmt.Println("\n📝 Set Metadata")
	contextCounter++
	key := fmt.Sprintf("key_%d", contextCounter)
	value := fmt.Sprintf("value_%d", contextCounter)

	client.SetMetadata(key, value)
	fmt.Printf("✅ Metadata set: %s = %s\n\n", key, value)
}

func viewContext() {
	fmt.Println("\n👀 Current Metadata")
	metadata := client.GetAllMetadata()
	if len(metadata) == 0 {
		fmt.Println("(empty)")
	} else {
		for k, v := range metadata {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
	fmt.Println()
}

func trackEvent() {
	fmt.Println("\n📊 Track Event")
	eventCounter++
	name := fmt.Sprintf("event_%d", eventCounter)

	// Mock sample payload
	payload := map[string]interface{}{
		"action":    fmt.Sprintf("action_%d", eventCounter),
		"timestamp": time.Now().Unix(),
		"data": map[string]interface{}{
			"count": eventCounter,
			"type":  "sample",
		},
	}

	metadata := &ripple.EventMetadata{SchemaVersion: stringPtr("1.0.0")}

	client.Track(name, payload, metadata)
	fmt.Printf("✅ Event '%s' tracked with sample payload\n\n", name)
}

func trackEventWithError() {
	fmt.Println("\n⚠️  Track Event with Error (Test Retry)")
	eventCounter++
	name := fmt.Sprintf("error_event_%d", eventCounter)

	// Payload with error trigger
	payload := map[string]interface{}{
		"action":        fmt.Sprintf("error_action_%d", eventCounter),
		"timestamp":     time.Now().Unix(),
		"trigger_error": true, // This will cause server to return 500
		"data": map[string]interface{}{
			"count": eventCounter,
			"type":  "error_test",
		},
	}

	metadata := &ripple.EventMetadata{SchemaVersion: stringPtr("1.0.0")}

	client.Track(name, payload, metadata)
	fmt.Printf("✅ Error event '%s' tracked - will trigger retry logic\n\n", name)
}

func flush() {
	fmt.Println("\n🔄 Flushing events...")
	client.Flush()
	fmt.Println("✅ Events flushed\n")
}
