package ripple

import "sync"

// Mutex provides mutual exclusion lock for preventing race conditions
type Mutex struct {
	mu sync.Mutex
}

// NewMutex creates a new mutex
func NewMutex() *Mutex {
	return &Mutex{}
}

// RunAtomic executes a task with exclusive lock
func (m *Mutex) RunAtomic(task func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return task()
}
