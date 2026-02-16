package ripple

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

const (
	maxBackoffDuration = 30 * time.Second
	maxJitterMs        = 1000
)

// Dispatcher manages event queuing, batching, flushing, and retry logic.
type Dispatcher struct {
	config         DispatcherConfig
	queue          *Queue
	httpAdapter    HTTPAdapter
	storageAdapter StorageAdapter
	loggerAdapter  LoggerAdapter
	headers        map[string]string
	timer          *time.Timer
	flushMu        sync.Mutex
	retryCancel    context.CancelFunc
	disposed       bool
	mu             sync.Mutex
}

// NewDispatcher creates a new Dispatcher instance.
func NewDispatcher(config DispatcherConfig, httpAdapter HTTPAdapter, storageAdapter StorageAdapter, loggerAdapter LoggerAdapter) *Dispatcher {
	return &Dispatcher{
		config:         config,
		queue:          NewQueue(),
		httpAdapter:    httpAdapter,
		storageAdapter: storageAdapter,
		loggerAdapter:  loggerAdapter,
		headers: map[string]string{
			config.APIKeyHeader: config.APIKey,
			"Content-Type":      "application/json",
		},
	}
}

// Enqueue adds an event to the queue.
func (d *Dispatcher) Enqueue(event Event) {
	d.mu.Lock()
	if d.disposed {
		d.mu.Unlock()
		d.loggerAdapter.Warn("Cannot enqueue event: Dispatcher has been disposed")
		return
	}
	d.mu.Unlock()

	d.queue.Enqueue(event)

	// Apply buffer limit and persist
	eventsToSave := d.applyQueueLimit(d.queue.ToSlice())
	if len(eventsToSave) < d.queue.Len() {
		d.queue.Clear()
		d.queue.LoadFromSlice(eventsToSave)
	}

	if err := d.storageAdapter.Save(eventsToSave); err != nil {
		d.logStorageError("Failed to persist events to storage", err, map[string]any{
			"queueSize": d.queue.Len(),
		})
	}

	if d.queue.Len() >= d.config.MaxBatchSize {
		d.Flush()
	} else {
		d.scheduleFlush()
	}
}

// Flush immediately flushes all queued events.
func (d *Dispatcher) Flush() {
	d.flushMu.Lock()
	defer d.flushMu.Unlock()

	d.stopTimer()

	if d.queue.IsEmpty() {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.mu.Lock()
	d.retryCancel = cancel
	d.mu.Unlock()
	defer cancel()

	allEvents := d.queue.ToSlice()
	d.queue.Clear()

	for i := 0; i < len(allEvents); i += d.config.MaxBatchSize {
		end := i + d.config.MaxBatchSize
		if end > len(allEvents) {
			end = len(allEvents)
		}
		d.sendWithRetry(ctx, allEvents[i:end], 0)
	}
}

// Restore loads persisted events from storage.
func (d *Dispatcher) Restore() {
	d.mu.Lock()
	d.disposed = false
	d.mu.Unlock()

	events, err := d.storageAdapter.Load()
	if err != nil {
		d.loggerAdapter.Error("Failed to restore events from storage", map[string]any{
			"error": err.Error(),
		})
		return
	}

	limited := d.applyQueueLimit(events)
	d.queue.LoadFromSlice(limited)

	if d.queue.Len() > 0 {
		d.scheduleFlush()
	}
}

// Dispose cleans up resources: aborts retries, clears queue, releases mutex.
func (d *Dispatcher) Dispose() {
	d.mu.Lock()
	d.disposed = true
	cancel := d.retryCancel
	d.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	d.stopTimer()
	d.queue.Clear()

	if err := d.storageAdapter.Close(); err != nil {
		d.loggerAdapter.Error("failed to close storage adapter", map[string]any{
			"error": err.Error(),
		})
	}
}

// applyQueueLimit applies the maxBufferSize limit using FIFO eviction.
func (d *Dispatcher) applyQueueLimit(events []Event) []Event {
	if d.config.MaxBufferSize > 0 && len(events) > d.config.MaxBufferSize {
		return events[len(events)-d.config.MaxBufferSize:]
	}
	return events
}

// sendWithRetry sends events with exponential backoff retry logic.
// Note: This method never logs headers to prevent API key exposure.
func (d *Dispatcher) sendWithRetry(ctx context.Context, events []Event, attempt int) {
	resp, err := d.httpAdapter.SendWithContext(ctx, d.config.Endpoint, events, d.headers)

	if err != nil {
		d.handleNetworkError(ctx, err, events, attempt)
	} else {
		d.handleResponse(ctx, resp, events, attempt)
	}
}

func (d *Dispatcher) handleResponse(ctx context.Context, resp *HTTPResponse, events []Event, attempt int) {
	if resp.Status >= 200 && resp.Status < 300 {
		if err := d.storageAdapter.Clear(); err != nil {
			d.loggerAdapter.Error("Failed to clear storage after successful send", map[string]any{
				"error": err.Error(),
			})
		}
	} else if resp.Status >= 400 && resp.Status < 500 {
		d.loggerAdapter.Warn("4xx client error, dropping events", map[string]any{
			"status":      resp.Status,
			"eventsCount": len(events),
		})
		if err := d.storageAdapter.Clear(); err != nil {
			d.loggerAdapter.Error("Failed to clear storage after 4xx error", map[string]any{
				"error": err.Error(),
			})
		}
	} else if resp.Status >= 500 {
		d.handleServerError(ctx, resp.Status, events, attempt)
	} else {
		d.loggerAdapter.Warn("Unexpected status code, dropping events", map[string]any{
			"status":      resp.Status,
			"eventsCount": len(events),
		})
		if err := d.storageAdapter.Clear(); err != nil {
			d.loggerAdapter.Error("Failed to clear storage after unexpected status", map[string]any{
				"error": err.Error(),
			})
		}
	}
}

func (d *Dispatcher) handleServerError(ctx context.Context, status int, events []Event, attempt int) {
	if attempt < d.config.MaxRetries {
		d.loggerAdapter.Warn("5xx server error, retrying", map[string]any{
			"status":     status,
			"attempt":    attempt + 1,
			"maxRetries": d.config.MaxRetries,
		})

		if !d.delay(ctx, d.calculateBackoff(attempt)) {
			return
		}
		d.sendWithRetry(ctx, events, attempt+1)
	} else {
		d.loggerAdapter.Error("5xx server error, max retries reached", map[string]any{
			"status":      status,
			"maxRetries":  d.config.MaxRetries,
			"eventsCount": len(events),
		})
		d.requeueEvents(events)
	}
}

func (d *Dispatcher) handleNetworkError(ctx context.Context, err error, events []Event, attempt int) {
	d.loggerAdapter.Error("Network error occurred", map[string]any{"error": err.Error()})

	if attempt < d.config.MaxRetries {
		d.loggerAdapter.Warn("Network error, retrying", map[string]any{
			"attempt":    attempt + 1,
			"maxRetries": d.config.MaxRetries,
			"error":      err.Error(),
		})

		if !d.delay(ctx, d.calculateBackoff(attempt)) {
			return
		}
		d.sendWithRetry(ctx, events, attempt+1)
	} else {
		d.loggerAdapter.Error("Network error, max retries reached", map[string]any{
			"maxRetries":  d.config.MaxRetries,
			"eventsCount": len(events),
			"error":       err.Error(),
		})
		d.requeueEvents(events)
	}
}

func (d *Dispatcher) requeueEvents(events []Event) {
	currentQueue := d.queue.ToSlice()
	events = append(events, currentQueue...)
	limited := d.applyQueueLimit(events)
	d.queue.Clear()
	d.queue.LoadFromSlice(limited)

	if err := d.storageAdapter.Save(limited); err != nil {
		d.logStorageError("Failed to persist events after requeue", err, nil)
	}
}

// scheduleFlush schedules a one-shot flush after the configured interval.
func (d *Dispatcher) scheduleFlush() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.disposed || d.timer != nil {
		return
	}

	d.timer = time.AfterFunc(d.config.FlushInterval, func() {
		d.mu.Lock()
		d.timer = nil
		d.mu.Unlock()
		d.Flush()
	})
}

func (d *Dispatcher) stopTimer() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}

// logStorageError logs storage errors, using warn level for StorageQuotaExceededError.
func (d *Dispatcher) logStorageError(message string, err error, extra map[string]any) {
	args := map[string]any{"error": err.Error()}
	for k, v := range extra {
		args[k] = v
	}
	var quotaErr *StorageQuotaExceededError
	if errors.As(err, &quotaErr) {
		d.loggerAdapter.Warn(message, args)
	} else {
		d.loggerAdapter.Error(message, args)
	}
}

// calculateBackoff computes exponential backoff with jitter.
// Formula: (2^attempt seconds) + random jitter, capped at 30s.
// Example progression: 1s, 2s, 4s, 8s, 16s, 30s (capped).
func (d *Dispatcher) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: 2^attempt seconds
	backoff := time.Duration(1<<attempt) * time.Second
	if backoff > maxBackoffDuration {
		backoff = maxBackoffDuration
	}
	// Add random jitter (0-1000ms) to prevent thundering herd
	jitter := time.Duration(rand.Intn(maxJitterMs)) * time.Millisecond
	return backoff + jitter
}

// delay waits for the given duration or until context is cancelled.
// Returns true if the delay completed, false if cancelled.
func (d *Dispatcher) delay(ctx context.Context, duration time.Duration) bool {
	select {
	case <-time.After(duration):
		return true
	case <-ctx.Done():
		return false
	}
}
