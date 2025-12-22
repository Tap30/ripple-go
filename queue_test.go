package ripple

import "testing"

func TestQueue_EnqueueDequeue(t *testing.T) {
	q := NewQueue()
	event := Event{Name: "test"}
	q.Enqueue(event)

	dequeued, ok := q.Dequeue()
	if !ok || dequeued.Name != "test" {
		t.Fatal("expected to dequeue event")
	}
}

func TestQueue_IsEmpty(t *testing.T) {
	q := NewQueue()
	if !q.IsEmpty() {
		t.Fatal("expected queue to be empty")
	}
	q.Enqueue(Event{Name: "test"})
	if q.IsEmpty() {
		t.Fatal("expected queue not to be empty")
	}
}

func TestQueue_Len(t *testing.T) {
	q := NewQueue()
	if q.Len() != 0 {
		t.Fatal("expected length 0")
	}
	q.Enqueue(Event{Name: "test1"})
	q.Enqueue(Event{Name: "test2"})
	if q.Len() != 2 {
		t.Fatal("expected length 2")
	}
}

func TestQueue_Clear(t *testing.T) {
	q := NewQueue()
	q.Enqueue(Event{Name: "test"})
	q.Clear()
	if !q.IsEmpty() {
		t.Fatal("expected queue to be empty after clear")
	}
}

func TestQueue_ToSlice(t *testing.T) {
	q := NewQueue()
	q.Enqueue(Event{Name: "test1"})
	q.Enqueue(Event{Name: "test2"})

	slice := q.ToSlice()
	if len(slice) != 2 || slice[0].Name != "test1" || slice[1].Name != "test2" {
		t.Fatal("expected slice with 2 events in order")
	}
}

func TestQueue_LoadFromSlice(t *testing.T) {
	q := NewQueue()
	events := []Event{{Name: "test1"}, {Name: "test2"}}
	q.LoadFromSlice(events)

	if q.Len() != 2 {
		t.Fatal("expected length 2")
	}
	dequeued, _ := q.Dequeue()
	if dequeued.Name != "test1" {
		t.Fatal("expected first event to be test1")
	}
}

func TestQueue_DequeueEmpty(t *testing.T) {
	q := NewQueue()
	_, ok := q.Dequeue()
	if ok {
		t.Fatal("expected dequeue to fail on empty queue")
	}
}
