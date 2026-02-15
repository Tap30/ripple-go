package ripple

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/Tap30/ripple-go/adapters"
)

const (
	maxBackoffDuration = 60 * time.Second
	maxJitterMs        = 1000
)

// Dispatcher manages event queuing, batching, flushing, and retry logic.
// Automatically flushes events based on batch size or time interval.
// Implements exponential backoff with jitter for retries.
type Dispatcher struct {
	config         DispatcherConfig
	queue          *Queue
	httpAdapter    HTTPAdapter
	storageAdapter StorageAdapter
	loggerAdapter  LoggerAdapter
	headers        map[string]string
	ticker         *time.Ticker
	stopChan       chan struct{}
	flushMutex     *Mutex
	wg             sync.WaitGroup
	timerStarted   bool
	stopped        bool
	timerMu        sync.Mutex
}

// NewDispatcher creates a new Dispatcher instance.
func NewDispatcher(config DispatcherConfig, httpAdapter HTTPAdapter, storageAdapter StorageAdapter, headers map[string]string) *Dispatcher {
	d := &Dispatcher{
		config:         config,
		queue:          NewQueue(),
		httpAdapter:    httpAdapter,
		storageAdapter: storageAdapter,
		loggerAdapter:  adapters.NewPrintLoggerAdapter(adapters.LogLevelWarn),
		headers:        headers,
		stopChan:       make(chan struct{}),
		flushMutex:     NewMutex(),
	}

	// Validate configuration
	if config.MaxBufferSize > 0 && config.MaxBufferSize < config.MaxBatchSize {
		d.loggerAdapter.Warn(
			"Configuration warning: maxBufferSize (%d) is less than maxBatchSize (%d). "+
				"This means the batch size will never be reached and events will be dropped unnecessarily. "+
				"Consider setting maxBufferSize >= maxBatchSize.",
			config.MaxBufferSize,
			config.MaxBatchSize,
		)
	}

	return d
}

// SetLoggerAdapter sets a custom logger adapter.
func (d *Dispatcher) SetLoggerAdapter(logger LoggerAdapter) {
	d.loggerAdapter = logger
}

// Enqueue adds an event to the queue.
// Triggers auto-flush if batch size threshold is reached.
func (d *Dispatcher) Enqueue(event Event) {
	d.queue.Enqueue(event)

	// Apply buffer limit and persist
	eventsToSave := d.applyQueueLimit(d.queue.ToSlice())
	if len(eventsToSave) < d.queue.Len() {
		// Queue was trimmed, reload with limited events
		d.queue.Clear()
		d.queue.LoadFromSlice(eventsToSave)
	}

	if err := d.storageAdapter.Save(eventsToSave); err != nil {
		d.loggerAdapter.Error("Failed to persist events to storage", map[string]any{
			"error":     err.Error(),
			"queueSize": d.queue.Len(),
		})
	}

	if d.queue.Len() >= d.config.MaxBatchSize {
		go d.Flush()
	} else {
		d.scheduleFlush()
	}
}

// Flush immediately flushes all queued events.
// Cancels any scheduled flush.
// Uses mutex to prevent concurrent flush operations.
// Events are sent in batches according to maxBatchSize (dynamic rebatching).
func (d *Dispatcher) Flush() {
	d.flushMutex.RunAtomic(func() error {
		d.stopTimer()

		if d.queue.IsEmpty() {
			return nil
		}

		ctx := context.Background()
		if d.config.FlushTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, d.config.FlushTimeout)
			defer cancel()
		}

		allEvents := d.queue.ToSlice()
		d.queue.Clear()

		var lastErr error
		for i := 0; i < len(allEvents); i += d.config.MaxBatchSize {
			end := i + d.config.MaxBatchSize
			if end > len(allEvents) {
				end = len(allEvents)
			}
			batch := allEvents[i:end]

			if err := d.sendWithRetry(ctx, batch, 0); err != nil {
				lastErr = err
			}
		}

		return lastErr
	})
}

// Restore restores persisted events from storage.
// Called during initialization to recover unsent events.
func (d *Dispatcher) Restore() error {
	events, err := d.storageAdapter.Load()
	if err != nil {
		d.loggerAdapter.Error("Failed to restore events from storage", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	limited := d.applyQueueLimit(events)
	d.queue.LoadFromSlice(limited)

	if d.queue.Len() > 0 {
		d.scheduleFlush()
	}

	return nil
}

// Dispose cleans up resources, cancels scheduled flushes, and clears all references.
// stopDispatcher is the common stop logic that ensures stopChan is only closed once.
func (d *Dispatcher) stopDispatcher() bool {
	d.timerMu.Lock()
	if d.stopped {
		d.timerMu.Unlock()
		return false
	}
	d.stopped = true
	d.timerMu.Unlock()

	d.stopTimer()
	close(d.stopChan)
	d.wg.Wait()
	return true
}

// Should be called when disposing the client.
func (d *Dispatcher) Dispose() {
	if !d.stopDispatcher() {
		return
	}
	d.queue.Clear()
}

// Stop stops the dispatcher and flushes all events.
func (d *Dispatcher) Stop() error {
	if !d.stopDispatcher() {
		return nil
	}

	d.Flush()

	events := d.queue.ToSlice()
	if len(events) > 0 {
		limited := d.applyQueueLimit(events)
		return d.storageAdapter.Save(limited)
	}
	return nil
}

// StopWithoutFlush stops the dispatcher and persists events to storage without flushing to server.
func (d *Dispatcher) StopWithoutFlush() error {
	if !d.stopDispatcher() {
		return nil
	}

	events := d.queue.ToSlice()
	if len(events) > 0 {
		limited := d.applyQueueLimit(events)
		return d.storageAdapter.Save(limited)
	}
	return nil
}

// applyQueueLimit applies the maxBufferSize limit to events using FIFO eviction.
// Returns the limited slice of events (keeping the most recent ones).
func (d *Dispatcher) applyQueueLimit(events []Event) []Event {
	if d.config.MaxBufferSize > 0 && len(events) > d.config.MaxBufferSize {
		return events[len(events)-d.config.MaxBufferSize:]
	}
	return events
}

// sendWithRetry sends events with exponential backoff retry logic.
func (d *Dispatcher) sendWithRetry(ctx context.Context, events []Event, attempt int) error {
	resp, err := d.httpAdapter.SendWithContext(ctx, d.config.Endpoint, events, d.headers)

	if err != nil {
		return d.handleNetworkError(ctx, err, events, attempt)
	} else {
		return d.handleResponse(ctx, resp, events, attempt)
	}
}

// handleResponse handles HTTP response based on status code.
func (d *Dispatcher) handleResponse(ctx context.Context, resp *HTTPResponse, events []Event, attempt int) error {
	if resp.Status >= 200 && resp.Status < 300 {
		if err := d.clearStorage(); err != nil {
			d.loggerAdapter.Error("Failed to clear storage after successful send", map[string]any{
				"error": err.Error(),
			})
			return err
		}
		return nil
	} else if resp.Status >= 400 && resp.Status < 500 {
		d.loggerAdapter.Warn("4xx client error, dropping events", map[string]any{
			"status":      resp.Status,
			"eventsCount": len(events),
		})
		if err := d.clearStorage(); err != nil {
			d.loggerAdapter.Error("Failed to clear storage after 4xx error", map[string]any{
				"error": err.Error(),
			})
			return err
		}
		return nil
	} else if resp.Status >= 500 {
		return d.handleServerError(ctx, resp.Status, events, attempt)
	} else {
		// 1xx, 3xx: Unexpected status codes, treat as client error and drop
		d.loggerAdapter.Warn("Unexpected status code, dropping events", map[string]any{
			"status":      resp.Status,
			"eventsCount": len(events),
		})
		if err := d.clearStorage(); err != nil {
			d.loggerAdapter.Error("Failed to clear storage after unexpected status", map[string]any{
				"error": err.Error(),
			})
			return err
		}
		return nil
	}
}

// handleServerError handles 5xx server errors with retry logic.
func (d *Dispatcher) handleServerError(ctx context.Context, status int, events []Event, attempt int) error {
	if attempt < d.config.MaxRetries {
		d.loggerAdapter.Warn("5xx server error, retrying", map[string]any{
			"status":     status,
			"attempt":    attempt + 1,
			"maxRetries": d.config.MaxRetries,
		})

		time.Sleep(d.calculateBackoff(attempt))
		return d.sendWithRetry(ctx, events, attempt+1)
	} else {
		d.loggerAdapter.Error("5xx server error, max retries reached", map[string]any{
			"status":      status,
			"maxRetries":  d.config.MaxRetries,
			"eventsCount": len(events),
		})
		if err := d.requeueEvents(events); err != nil {
			d.loggerAdapter.Error("Failed to persist events after max retries", map[string]any{
				"error":       err.Error(),
				"eventsCount": d.queue.Len(),
			})
			return err
		}
		return nil
	}
}

// handleNetworkError handles network errors with retry logic.
func (d *Dispatcher) handleNetworkError(ctx context.Context, err error, events []Event, attempt int) error {
	d.loggerAdapter.Error("Network error occurred", map[string]any{"error": err.Error()})

	if attempt < d.config.MaxRetries {
		d.loggerAdapter.Warn("Network error, retrying", map[string]any{
			"attempt":    attempt + 1,
			"maxRetries": d.config.MaxRetries,
			"error":      err.Error(),
		})

		time.Sleep(d.calculateBackoff(attempt))
		return d.sendWithRetry(ctx, events, attempt+1)
	} else {
		d.loggerAdapter.Error("Network error, max retries reached", map[string]any{
			"maxRetries":  d.config.MaxRetries,
			"eventsCount": len(events),
			"error":       err.Error(),
		})
		if err := d.requeueEvents(events); err != nil {
			d.loggerAdapter.Error("Failed to persist events after network error", map[string]any{
				"error":       err.Error(),
				"eventsCount": d.queue.Len(),
			})
			return err
		}
		return nil
	}
}

// clearStorage clears storage and returns any error.
func (d *Dispatcher) clearStorage() error {
	return d.storageAdapter.Clear()
}

// requeueEvents re-queues events and persists to storage, returning any error.
func (d *Dispatcher) requeueEvents(events []Event) error {
	currentQueue := d.queue.ToSlice()

	// Re-queue failed events at the front
	events = append(events, currentQueue...)

	// Apply buffer limit before loading into queue
	limited := d.applyQueueLimit(events)
	d.queue.Clear()
	d.queue.LoadFromSlice(limited)

	return d.storageAdapter.Save(limited)
}

// scheduleFlush schedules an automatic flush after the configured interval.
// Does nothing if a flush is already scheduled.
func (d *Dispatcher) scheduleFlush() {
	d.timerMu.Lock()
	defer d.timerMu.Unlock()

	if d.timerStarted || d.stopped {
		return
	}

	d.ticker = time.NewTicker(d.config.FlushInterval)
	d.timerStarted = true

	tickerChan := d.ticker.C

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		for {
			select {
			case <-tickerChan:
				d.Flush()
			case <-d.stopChan:
				return
			}
		}
	}()
}

// stopTimer stops the flush timer if it's running.
func (d *Dispatcher) stopTimer() {
	d.timerMu.Lock()
	defer d.timerMu.Unlock()

	if d.ticker != nil {
		d.ticker.Stop()
		d.ticker = nil
	}
	d.timerStarted = false
}

// calculateBackoff calculates exponential backoff with jitter.
func (d *Dispatcher) calculateBackoff(attempt int) time.Duration {
	backoff := time.Duration(1<<attempt) * time.Second
	if backoff > maxBackoffDuration {
		backoff = maxBackoffDuration
	}
	jitter := time.Duration(rand.Intn(maxJitterMs)) * time.Millisecond
	return backoff + jitter
}
