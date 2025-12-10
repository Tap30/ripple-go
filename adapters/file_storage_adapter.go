package adapters

import (
	"encoding/json"
	"os"
)

// FileStorageAdapter is the default storage adapter implementation using file system.
// Stores events as JSON in a file.
type FileStorageAdapter struct {
	filepath string
}

// Ensure FileStorageAdapter implements StorageAdapter interface
var _ StorageAdapter = (*FileStorageAdapter)(nil)

// NewFileStorageAdapter creates a new FileStorageAdapter instance.
//
// Parameters:
//   - filepath: Path to the file where events will be stored
func NewFileStorageAdapter(filepath string) StorageAdapter {
	return &FileStorageAdapter{filepath: filepath}
}

// Save persists events to a JSON file.
func (f *FileStorageAdapter) Save(events []Event) error {
	data, err := json.Marshal(events)
	if err != nil {
		return err
	}
	return os.WriteFile(f.filepath, data, 0644)
}

// Load retrieves events from a JSON file.
// Returns empty array if file doesn't exist.
func (f *FileStorageAdapter) Load() ([]Event, error) {
	data, err := os.ReadFile(f.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Event{}, nil
		}
		return nil, err
	}
	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// Clear removes the storage file.
func (f *FileStorageAdapter) Clear() error {
	return os.Remove(f.filepath)
}
