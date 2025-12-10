package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	ripple "github.com/Tap30/ripple-go"
)

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
			setContext()
		case "2":
			viewContext()
		case "3":
			trackEvent()
		case "4":
			flush()
		case "5":
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
	fmt.Println("1. Set Context")
	fmt.Println("2. View Context")
	fmt.Println("3. Track Event")
	fmt.Println("4. Flush Events")
	fmt.Println("5. Exit")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func readInput(prompt string) string {
	fmt.Print(prompt)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func setContext() {
	fmt.Println("\n📝 Set Context")
	contextCounter++
	key := fmt.Sprintf("key_%d", contextCounter)
	value := fmt.Sprintf("value_%d", contextCounter)

	client.SetContext(key, value)
	fmt.Printf("✅ Context set: %s = %s\n\n", key, value)
}

func viewContext() {
	fmt.Println("\n👀 Current Context")
	ctx := client.GetContext()
	if len(ctx) == 0 {
		fmt.Println("(empty)")
	} else {
		for k, v := range ctx {
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

	metadata := &ripple.EventMetadata{SchemaVersion: "1.0.0"}

	client.Track(name, payload, metadata)
	fmt.Printf("✅ Event '%s' tracked with sample payload\n\n", name)
}

func flush() {
	fmt.Println("\n🔄 Flushing events...")
	client.Flush()
	fmt.Println("✅ Events flushed\n")
}
