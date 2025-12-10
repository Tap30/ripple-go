package main

import (
	"fmt"
	"time"

	ripple "github.com/Tap30/ripple-go"
)

func main() {
	client := ripple.NewClient(ripple.ClientConfig{
		APIKey:        "your-api-key",
		Endpoint:      "https://api.example.com/events",
		FlushInterval: 5 * time.Second,
		MaxBatchSize:  10,
		MaxRetries:    3,
	})

	if err := client.Init(); err != nil {
		panic(err)
	}
	defer client.Dispose()

	client.SetContext("userId", "123")
	client.SetContext("appVersion", "1.0.0")

	client.Track("page_view", map[string]interface{}{
		"page": "/home",
	}, nil)

	client.Track("user_action", map[string]interface{}{
		"button": "submit",
	}, &ripple.EventMetadata{
		SchemaVersion: "1.0.0",
	})

	client.Flush()

	fmt.Println("Events tracked successfully")
}
