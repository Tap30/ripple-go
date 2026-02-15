package ripple

import "sync"

// TODO: remove and use go native solutions

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

// Release forcefully unlocks the mutex if held, used during disposal.
func (m *Mutex) Release() {
	// TryLock + Unlock ensures we release without panicking if not held
	if m.mu.TryLock() {
		m.mu.Unlock()
	}
}
