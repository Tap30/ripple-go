package adapters

import (
	"testing"
)

func TestNoOpStorageAdapter_Save(t *testing.T) {
	adapter := NewNoOpStorageAdapter()

	events := []Event{
		{Name: "test_event", Payload: map[string]any{"key": "value"}},
	}

	err := adapter.Save(events)
	if err != nil {
		t.Errorf("Save should always return nil, got: %v", err)
	}
}

func TestNoOpStorageAdapter_Load(t *testing.T) {
	adapter := NewNoOpStorageAdapter()

	events, err := adapter.Load()
	if err != nil {
		t.Errorf("Load should return nil error, got: %v", err)
	}

	if events == nil {
		t.Error("Load should return empty slice, not nil")
	}

	if len(events) != 0 {
		t.Errorf("Load should return empty slice, got %d events", len(events))
	}
}

func TestNoOpStorageAdapter_Clear(t *testing.T) {
	adapter := NewNoOpStorageAdapter()

	err := adapter.Clear()
	if err != nil {
		t.Errorf("Clear should always return nil, got: %v", err)
	}
}

func TestNoOpStorageAdapter_Interface(t *testing.T) {
	var _ StorageAdapter = (*NoOpStorageAdapter)(nil)
}
