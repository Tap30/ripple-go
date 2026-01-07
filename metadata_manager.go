package ripple

import "sync"

// MetadataManager manages global metadata attached to all events
type MetadataManager struct {
	metadata map[string]any
	mu       sync.RWMutex
}

// NewMetadataManager creates a new metadata manager
func NewMetadataManager() *MetadataManager {
	return &MetadataManager{
		metadata: make(map[string]any),
	}
}

// Set sets a metadata value
func (m *MetadataManager) Set(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadata[key] = value
}

// GetAll returns all metadata as a copy
func (m *MetadataManager) GetAll() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]any, len(m.metadata))
	for k, v := range m.metadata {
		result[k] = v
	}
	return result
}

// IsEmpty returns true if no metadata is set
func (m *MetadataManager) IsEmpty() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.metadata) == 0
}

// Clear removes all metadata
func (m *MetadataManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadata = make(map[string]any)
}
