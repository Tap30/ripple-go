package ripple

import "sync"

// MetadataManager manages global metadata attached to all events
type MetadataManager struct {
	metadata map[string]interface{}
	mu       sync.RWMutex
}

// NewMetadataManager creates a new metadata manager
func NewMetadataManager() *MetadataManager {
	return &MetadataManager{
		metadata: make(map[string]interface{}),
	}
}

// Set sets a metadata value
func (m *MetadataManager) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadata[key] = value
}

// Get gets a metadata value
func (m *MetadataManager) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metadata[key]
}

// GetAll returns all metadata as a copy
func (m *MetadataManager) GetAll() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.metadata) == 0 {
		return nil
	}

	result := make(map[string]interface{}, len(m.metadata))
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
	m.metadata = make(map[string]interface{})
}
