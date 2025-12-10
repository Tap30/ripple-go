package adapters

// Event represents a tracked event.
type Event struct {
	Name     string                 `json:"name"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
	IssuedAt int64                  `json:"issuedAt"`
	Context  map[string]interface{} `json:"context,omitempty"`
	Metadata *EventMetadata         `json:"metadata,omitempty"`
	Platform *Platform              `json:"platform,omitempty"`
}

// EventMetadata contains optional event metadata.
type EventMetadata struct {
	SchemaVersion string `json:"schemaVersion,omitempty"`
}

// Platform identifies the runtime environment.
type Platform struct {
	Type string `json:"type"`
}
