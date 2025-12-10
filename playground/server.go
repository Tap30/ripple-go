package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const PORT = 3000

type EventsPayload struct {
	Events []map[string]interface{} `json:"events"`
}

func main() {
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		apiKey := r.URL.Query().Get("apiKey")
		if apiKey == "" {
			apiKey = r.Header.Get("Authorization")
		}

		log.Printf("🔑 API Key: %s", apiKey)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("❌ Failed to read body")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read body"})
			return
		}

		var payload EventsPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("❌ Invalid JSON")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
			return
		}

		prettyJSON, _ := json.MarshalIndent(payload, "", "  ")
		log.Printf("📊 Received events:\n%s", string(prettyJSON))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"received": len(payload.Events),
		})
	})

	log.Printf("🚀 Event tracking server running at http://localhost:%d", PORT)
	log.Printf("📍 Endpoint: http://localhost:%d/events", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil))
}
