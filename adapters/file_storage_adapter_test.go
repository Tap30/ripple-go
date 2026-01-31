package adapters

import (
	"os"
	"testing"
)

func TestFileStorageAdapter_SaveLoad(t *testing.T) {
	filepath := "test_events.json"
	defer os.Remove(filepath)

	adapter := NewFileStorageAdapter(filepath)
	events := []Event{{Name: "test1"}, {Name: "test2"}}

	if err := adapter.Save(events); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := adapter.Load()
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if len(loaded) != 2 || loaded[0].Name != "test1" || loaded[1].Name != "test2" {
		t.Fatal("loaded events do not match saved events")
	}
}

func TestFileStorageAdapter_LoadNonExistent(t *testing.T) {
	adapter := NewFileStorageAdapter("nonexistent.json")
	loaded, err := adapter.Load()
	if err != nil {
		t.Fatalf("expected no error for nonexistent file: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatal("expected empty slice for nonexistent file")
	}
}

func TestFileStorageAdapter_Clear(t *testing.T) {
	filepath := "test_clear.json"
	adapter := NewFileStorageAdapter(filepath)
	adapter.Save([]Event{{Name: "test"}})

	if err := adapter.Clear(); err != nil {
		t.Fatalf("failed to clear: %v", err)
	}

	if _, err := os.Stat(filepath); !os.IsNotExist(err) {
		t.Fatal("expected file to be deleted")
	}
}

func TestFileStorageAdapter_SaveError(t *testing.T) {
	adapter := NewFileStorageAdapter("/invalid/path/test.json")
	err := adapter.Save([]Event{{Name: "test"}})
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestFileStorageAdapter_LoadInvalidJSON(t *testing.T) {
	filepath := "test_invalid.json"
	defer os.Remove(filepath)

	os.WriteFile(filepath, []byte("invalid json"), 0644)

	adapter := NewFileStorageAdapter(filepath)
	_, err := adapter.Load()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFileStorageAdapter_SaveMarshalError(t *testing.T) {
	filepath := "test_marshal.json"
	defer os.Remove(filepath)

	adapter := NewFileStorageAdapter(filepath)
	events := []Event{{
		Name:    "test",
		Payload: map[string]any{"invalid": make(chan int)},
	}}
	err := adapter.Save(events)
	if err == nil {
		t.Fatal("expected error for unmarshalable data")
	}
}

func TestFileStorageAdapter_LoadPermissionError(t *testing.T) {
	// Create a file in a directory that doesn't exist
	adapter := NewFileStorageAdapter("/nonexistent/directory/file.json")

	// This should return empty array for nonexistent file/directory
	events, err := adapter.Load()
	if err != nil {
		// If there's an error, it should be handled gracefully
		if !os.IsNotExist(err) {
			t.Errorf("unexpected error type: %v", err)
		}
	} else {
		// Should return empty array
		if len(events) != 0 {
			t.Errorf("expected empty array, got %d events", len(events))
		}
	}
}
