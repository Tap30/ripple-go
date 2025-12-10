package adapters

// StorageAdapter is an interface for event persistence.
// Implement this interface to use custom storage backends (database, Redis, S3, etc.).
type StorageAdapter interface {
	// Save persists events to storage.
	//
	// Parameters:
	//   - events: Array of events to save
	//
	// Returns error if save fails.
	Save(events []Event) error

	// Load retrieves persisted events from storage.
	//
	// Returns array of events or error.
	Load() ([]Event, error)

	// Clear removes all persisted events from storage.
	//
	// Returns error if clear fails.
	Clear() error
}
