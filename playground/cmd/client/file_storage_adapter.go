package main

import (
	"encoding/json"
	"os"

	"github.com/Tap30/ripple-go/adapters"
)

// FileStorageAdapter stores events as JSON in a file.
type FileStorageAdapter struct {
	filepath string
}

// Ensure FileStorageAdapter implements StorageAdapter interface
var _ adapters.StorageAdapter = (*FileStorageAdapter)(nil)

// NewFileStorageAdapter creates a new FileStorageAdapter instance.
func NewFileStorageAdapter(filepath string) adapters.StorageAdapter {
	return &FileStorageAdapter{filepath: filepath}
}

// Save persists events to a JSON file.
func (f *FileStorageAdapter) Save(events []adapters.Event) error {
	data, err := json.Marshal(events)
	if err != nil {
		return err
	}
	return os.WriteFile(f.filepath, data, 0o644)
}

// Load retrieves events from a JSON file.
// Returns empty array if file doesn't exist.
func (f *FileStorageAdapter) Load() ([]adapters.Event, error) {
	data, err := os.ReadFile(f.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return []adapters.Event{}, nil
		}
		return nil, err
	}
	var events []adapters.Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// Clear removes the storage file.
func (f *FileStorageAdapter) Clear() error {
	err := os.Remove(f.filepath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Close does nothing for file storage (no persistent connections).
func (f *FileStorageAdapter) Close() error {
	return nil
}
