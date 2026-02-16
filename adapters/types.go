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

// StorageQuotaExceededError indicates that the storage quota has been exceeded.
// Storage adapters should return this error when they cannot save events due to quota limits.
// The dispatcher will log this as a warning instead of an error.
type StorageQuotaExceededError struct {
	Message string
}

func (e *StorageQuotaExceededError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "storage quota exceeded"
}
