package ripple

import (
	"container/list"
	"sync"
)

// Queue represents a thread-safe FIFO queue for Event items.
type Queue struct {
	mu   sync.Mutex
	list *list.List
}

// NewQueue creates and returns a new empty Queue.
func NewQueue() *Queue {
	return &Queue{list: list.New()}
}

// Enqueue adds an Event to the end of the queue.
func (q *Queue) Enqueue(event Event) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.list.PushBack(event)
}

// Dequeue removes and returns the front Event in the queue.
// It returns false if the queue is empty.
func (q *Queue) Dequeue() (Event, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.list.Len() == 0 {
		return Event{}, false
	}
	front := q.list.Front()
	q.list.Remove(front)
	return front.Value.(Event), true
}

// IsEmpty reports whether the queue has no elements.
func (q *Queue) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.list.Len() == 0
}

// Len returns the number of Events currently in the queue.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.list.Len()
}

// Clear removes all Events from the queue.
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.list.Init()
}

// ToSlice returns all Events in the queue as a slice, preserving order.
func (q *Queue) ToSlice() []Event {
	q.mu.Lock()
	defer q.mu.Unlock()
	events := make([]Event, 0, q.list.Len())
	for e := q.list.Front(); e != nil; e = e.Next() {
		events = append(events, e.Value.(Event))
	}
	return events
}

// LoadFromSlice replaces the queue contents with Events from the provided slice.
func (q *Queue) LoadFromSlice(events []Event) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.list.Init()
	for _, event := range events {
		q.list.PushBack(event)
	}
}
