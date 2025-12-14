package adapters

// Event represents a tracked event.
type Event struct {
	Name      string                 `json:"name"`
	Payload   map[string]interface{} `json:"payload"`
	Metadata  *EventMetadata         `json:"metadata"`
	IssuedAt  int64                  `json:"issuedAt"`
	Context   map[string]interface{} `json:"context"`
	SessionID *string                `json:"sessionId"`
	Platform  *Platform              `json:"platform"`
}

// EventMetadata contains optional event metadata.
type EventMetadata struct {
	SchemaVersion *string `json:"schemaVersion,omitempty"`
}

// Platform represents server platform information.
type Platform struct {
	Type string `json:"type"`
}
