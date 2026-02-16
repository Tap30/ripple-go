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

var client *ripple.Client
var scanner *bufio.Scanner
var metadataCounter int
var eventCounter int
var httpAdapter *ContextAwareHTTPAdapter

func main() {
	scanner = bufio.NewScanner(os.Stdin)

	httpAdapter = NewContextAwareHTTPAdapter(10 * time.Second) // Default 10s timeout

	var err error
	client, err = ripple.NewClient(ripple.ClientConfig{
		APIKey:         "test-api-key",
		Endpoint:       "http://localhost:3000/events",
		FlushInterval:  5 * time.Second,
		MaxBatchSize:   5,
		MaxRetries:     3,
		HTTPAdapter:    httpAdapter,
		StorageAdapter: NewFileStorageAdapter("ripple_events.json"),
		LoggerAdapter:  adapters.NewPrintLoggerAdapter(adapters.LogLevelDebug),
	})

	if err != nil {
		fmt.Printf("❌ Failed to create client: %v\n", err)
		return
	}

	if err := client.Init(); err != nil {
		fmt.Printf("❌ Failed to initialize client: %v\n", err)
		return
	}

	fmt.Println("🎯 Ripple Interactive Client")
	fmt.Println("Connected to: http://localhost:3000/events")
	fmt.Println("⏱️  HTTP Timeout: 10s (configurable)")
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
			viewMetadata()
		case "8":
			trackMultipleEvents()
		case "9":
			flush()
		case "10":
			trackEventWithError()
		case "11":
			testInvalidEndpoint()
		case "12":
			setHTTPTimeout()
		case "13":
			testTimeoutScenario()
		case "14":
			initClient()
		case "15":
			disposeClient()
		case "16":
			fmt.Println("👋 Goodbye!")
			client.Dispose()
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
	fmt.Println("7. View Current Metadata")
	fmt.Println()
	fmt.Println("📦 Batch and Flush")
	fmt.Println("8. Track Multiple Events (Batch Test)")
	fmt.Println("9. Manual Flush")
	fmt.Println()
	fmt.Println("⚠️  Error Handling")
	fmt.Println("10. Test Retry Logic (Error Event)")
	fmt.Println("11. Test Invalid Endpoint")
	fmt.Println()
	fmt.Println("⏱️  Context & Timeout Control")
	fmt.Println("12. Set HTTP Timeout")
	fmt.Println("13. Test Timeout Scenario")
	fmt.Println()
	fmt.Println("🔄 Lifecycle Management")
	fmt.Println("14. Initialize Client")
	fmt.Println("15. Dispose Client")
	fmt.Println("16. Exit")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func readInput(prompt string) string {
	fmt.Print(prompt)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func trackSimpleEvent() {
	fmt.Println("\n📊 Track Simple Event")
	if err := client.Track("button_click", nil, nil); err != nil {
		fmt.Printf("❌ Error tracking event: %v\n\n", err)
		return
	}
	fmt.Println("✅ Tracked: button_click\n")
}

func trackEventWithPayload() {
	fmt.Println("\n📊 Track Event with Payload")
	payload := map[string]any{
		"action":    "click",
		"target":    "button",
		"timestamp": time.Now().Unix(),
	}
	if err := client.Track("user_action", payload, nil); err != nil {
		fmt.Printf("❌ Error tracking event: %v\n\n", err)
		return
	}
	fmt.Println("✅ Tracked: user_action with payload\n")
}

func trackEventWithMetadata() {
	fmt.Println("\n📊 Track Event with Metadata")
	payload := map[string]any{
		"formId": "contact-form",
		"fields": 5,
	}
	metadata := map[string]any{"schemaVersion": "1.0.0"}
	if err := client.Track("form_submit", payload, metadata); err != nil {
		fmt.Printf("❌ Error tracking event: %v\n\n", err)
		return
	}
	fmt.Println("✅ Tracked: form_submit with metadata\n")
}

func trackEventWithCustomMetadata() {
	fmt.Println("\n📊 Track Event with Custom Metadata")
	payload := map[string]any{
		"orderId": "order-123",
		"amount":  99.99,
	}
	metadata := map[string]any{"schemaVersion": "2.1.0"}
	if err := client.Track("purchase_completed", payload, metadata); err != nil {
		fmt.Printf("❌ Error tracking event: %v\n\n", err)
		return
	}
	fmt.Println("✅ Tracked: purchase_completed with rich metadata\n")
}

func setSharedMetadata() {
	fmt.Println("\n🏷️  Set Shared Metadata")
	metadataCounter++
	key := fmt.Sprintf("key_%d", metadataCounter)
	value := fmt.Sprintf("value_%d", metadataCounter)

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
		payload := map[string]any{"index": i}
		client.Track("batch_event", payload, nil)
	}
	fmt.Println("✅ Tracked 10 events (should auto-flush at batch size 5)\n")
}

func testInvalidEndpoint() {
	fmt.Println("\n⚠️  Test Invalid Endpoint")

	// Create a new client with invalid endpoint
	errorClient, err := ripple.NewClient(ripple.ClientConfig{
		APIKey:         "test-key",
		Endpoint:       "http://localhost:9999/invalid",
		FlushInterval:  5 * time.Second,
		MaxBatchSize:   5,
		MaxRetries:     2,
		HTTPAdapter:    adapters.NewNetHTTPAdapter(),
		StorageAdapter: NewFileStorageAdapter("error_events.json"),
		LoggerAdapter:  adapters.NewPrintLoggerAdapter(adapters.LogLevelWarn),
	})

	if err != nil {
		fmt.Printf("❌ Failed to create error client: %v\n\n", err)
		return
	}

	if err := errorClient.Init(); err != nil {
		fmt.Printf("❌ Failed to init error client: %v\n\n", err)
		return
	}

	errorClient.Track("error_test", map[string]any{"shouldFail": true}, nil)
	fmt.Println("✅ Tracked event to invalid endpoint (check console for retries)\n")
}

func initClient() {
	fmt.Println("\n🔄 Initialize Client")
	if err := client.Init(); err != nil {
		fmt.Printf("❌ Error initializing client: %v\n\n", err)
		return
	}
	fmt.Println("✅ Client initialized\n")
}

func disposeClient() {
	fmt.Println("\n🔄 Dispose Client")
	client.Dispose()
	fmt.Println("✅ Client disposed\n")
}

func setMetadata() {
	fmt.Println("\n📝 Set Metadata")
	metadataCounter++
	key := fmt.Sprintf("key_%d", metadataCounter)
	value := fmt.Sprintf("value_%d", metadataCounter)

	client.SetMetadata(key, value)
	fmt.Printf("✅ Metadata set: %s = %s\n\n", key, value)
}

func viewMetadata() {
	fmt.Println("\n👀 Current Metadata")
	metadata := client.GetMetadata()
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
	payload := map[string]any{
		"action":    fmt.Sprintf("action_%d", eventCounter),
		"timestamp": time.Now().Unix(),
		"data": map[string]any{
			"count": eventCounter,
			"type":  "sample",
		},
	}

	metadata := map[string]any{"schemaVersion": "1.0.0"}

	client.Track(name, payload, metadata)
	fmt.Printf("✅ Event '%s' tracked with sample payload\n\n", name)
}

func trackEventWithError() {
	fmt.Println("\n⚠️  Track Event with Error (Test Retry)")
	eventCounter++
	name := fmt.Sprintf("error_event_%d", eventCounter)

	// Payload with error trigger
	payload := map[string]any{
		"action":        fmt.Sprintf("error_action_%d", eventCounter),
		"timestamp":     time.Now().Unix(),
		"trigger_error": true, // This will cause server to return 500
		"data": map[string]any{
			"count": eventCounter,
			"type":  "error_test",
		},
	}

	metadata := map[string]any{"schemaVersion": "1.0.0"}

	client.Track(name, payload, metadata)
	fmt.Printf("✅ Error event '%s' tracked - will trigger retry logic\n\n", name)
}

func flush() {
	fmt.Println("\n🔄 Flushing events...")
	client.Flush()
	fmt.Println("✅ Events flushed\n")
}

func setHTTPTimeout() {
	fmt.Println("\n⏱️  Set HTTP Timeout")
	fmt.Println("Current timeout: ", httpAdapter.timeout)
	input := readInput("Enter timeout in seconds (e.g., 5): ")

	var seconds int
	if _, err := fmt.Sscanf(input, "%d", &seconds); err != nil {
		fmt.Printf("❌ Invalid input: %v\n\n", err)
		return
	}

	httpAdapter.SetTimeout(time.Duration(seconds) * time.Second)
	fmt.Printf("✅ HTTP timeout set to %d seconds\n\n", seconds)
}

func testTimeoutScenario() {
	fmt.Println("\n⏱️  Test Timeout Scenario")
	fmt.Println("Setting timeout to 1ms to force timeout...")

	originalTimeout := httpAdapter.timeout
	httpAdapter.SetTimeout(1 * time.Millisecond)

	client.Track("timeout_test", map[string]any{"shouldTimeout": true}, nil)
	client.Flush()

	fmt.Println("✅ Timeout test completed (check logs for timeout errors)")

	// Restore original timeout
	httpAdapter.SetTimeout(originalTimeout)
	fmt.Printf("✅ Restored timeout to %v\n\n", originalTimeout)
}
