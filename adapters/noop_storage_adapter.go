package adapters

// NoOpStorageAdapter is a storage adapter that performs no operations.
// Useful for scenarios where event persistence is not required.
type NoOpStorageAdapter struct{}

// NewNoOpStorageAdapter creates a new NoOpStorageAdapter instance.
func NewNoOpStorageAdapter() *NoOpStorageAdapter {
	return &NoOpStorageAdapter{}
}

// Save does nothing and always returns nil.
func (n *NoOpStorageAdapter) Save(events []Event) error {
	return nil
}

// Load returns an empty slice and nil error.
func (n *NoOpStorageAdapter) Load() ([]Event, error) {
	return []Event{}, nil
}

// Clear does nothing and always returns nil.
func (n *NoOpStorageAdapter) Clear() error {
	return nil
}

// Close does nothing and always returns nil.
func (n *NoOpStorageAdapter) Close() error {
	return nil
}
