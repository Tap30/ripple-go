package adapters

// Event represents a tracked event.
type Event struct {
	Name      string         `json:"name"`
	Payload   map[string]any `json:"payload"`
	Metadata  map[string]any `json:"metadata"`
	IssuedAt  int64          `json:"issuedAt"`
	SessionID *string        `json:"sessionId"`
	Platform  *Platform      `json:"platform"`
}

// EventMetadata contains optional event metadata.
type EventMetadata = map[string]any

// Platform represents server platform information.
type Platform struct {
	Type string `json:"type"`
}
