package ripple

import (
	"math/rand"
	"sync"
	"time"

	"github.com/Tap30/ripple-go/adapters"
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

		allEvents := d.queue.ToSlice()
		d.queue.Clear()

		for i := 0; i < len(allEvents); i += d.config.MaxBatchSize {
			end := i + d.config.MaxBatchSize
			if end > len(allEvents) {
				end = len(allEvents)
			}
			batch := allEvents[i:end]

			d.sendWithRetry(batch, 0)
		}

		return nil
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
// Should be called when disposing the client.
func (d *Dispatcher) Dispose() {
	d.timerMu.Lock()
	if d.stopped {
		d.timerMu.Unlock()
		return
	}
	d.stopped = true
	d.timerMu.Unlock()

	d.stopTimer()
	close(d.stopChan)
	d.wg.Wait()
	d.queue.Clear()
}

// Stop stops the dispatcher and flushes all events.
func (d *Dispatcher) Stop() error {
	d.timerMu.Lock()
	if d.stopped {
		d.timerMu.Unlock()
		return nil
	}
	d.stopped = true
	d.timerMu.Unlock()

	d.stopTimer()
	close(d.stopChan)
	d.wg.Wait()

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
	d.timerMu.Lock()
	if d.stopped {
		d.timerMu.Unlock()
		return nil
	}
	d.stopped = true
	d.timerMu.Unlock()

	d.stopTimer()
	close(d.stopChan)
	d.wg.Wait()

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
func (d *Dispatcher) sendWithRetry(events []Event, attempt int) {
	resp, err := d.httpAdapter.Send(d.config.Endpoint, events, d.headers)

	if err != nil {
		d.handleNetworkError(err, events, attempt)
	} else {
		d.handleResponse(resp, events, attempt)
	}
}

// handleResponse handles HTTP response based on status code.
func (d *Dispatcher) handleResponse(resp *HTTPResponse, events []Event, attempt int) {
	if resp.Status >= 200 && resp.Status < 300 {
		d.clearStorage("Failed to clear storage after successful send")
	} else if resp.Status >= 400 && resp.Status < 500 {
		d.loggerAdapter.Warn("4xx client error, dropping events", map[string]any{
			"status":      resp.Status,
			"eventsCount": len(events),
		})
		d.clearStorage("Failed to clear storage after 4xx error")
	} else if resp.Status >= 500 {
		d.handleServerError(resp.Status, events, attempt)
	} else {
		// 1xx, 3xx: Unexpected status codes, treat as client error and drop
		d.loggerAdapter.Warn("Unexpected status code, dropping events", map[string]any{
			"status":      resp.Status,
			"eventsCount": len(events),
		})
		d.clearStorage("Failed to clear storage after unexpected status")
	}
}

// handleServerError handles 5xx server errors with retry logic.
func (d *Dispatcher) handleServerError(status int, events []Event, attempt int) {
	if attempt < d.config.MaxRetries {
		d.loggerAdapter.Warn("5xx server error, retrying", map[string]any{
			"status":     status,
			"attempt":    attempt + 1,
			"maxRetries": d.config.MaxRetries,
		})

		time.Sleep(d.calculateBackoff(attempt))
		d.sendWithRetry(events, attempt+1)
	} else {
		d.loggerAdapter.Error("5xx server error, max retries reached", map[string]any{
			"status":      status,
			"maxRetries":  d.config.MaxRetries,
			"eventsCount": len(events),
		})
		d.requeueEvents(events, "Failed to persist events after max retries")
	}
}

// handleNetworkError handles network errors with retry logic.
func (d *Dispatcher) handleNetworkError(err error, events []Event, attempt int) {
	d.loggerAdapter.Error("Network error occurred", map[string]any{"error": err.Error()})

	if attempt < d.config.MaxRetries {
		d.loggerAdapter.Warn("Network error, retrying", map[string]any{
			"attempt":    attempt + 1,
			"maxRetries": d.config.MaxRetries,
			"error":      err.Error(),
		})

		time.Sleep(d.calculateBackoff(attempt))
		d.sendWithRetry(events, attempt+1)
	} else {
		d.loggerAdapter.Error("Network error, max retries reached", map[string]any{
			"maxRetries":  d.config.MaxRetries,
			"eventsCount": len(events),
			"error":       err.Error(),
		})
		d.requeueEvents(events, "Failed to persist events after network error")
	}
}

// clearStorage clears storage and logs errors if clearing fails.
func (d *Dispatcher) clearStorage(errorMessage string) {
	if err := d.storageAdapter.Clear(); err != nil {
		d.loggerAdapter.Error(errorMessage, map[string]any{
			"error": err.Error(),
		})
	}
}

// requeueEvents re-queues events and persists to storage.
func (d *Dispatcher) requeueEvents(events []Event, errorMessage string) {
	currentQueue := d.queue.ToSlice()

	// Re-queue failed events at the front
	events = append(events, currentQueue...)
	d.queue.Clear()
	d.queue.LoadFromSlice(events)

	eventsToSave := d.applyQueueLimit(d.queue.ToSlice())
	if err := d.storageAdapter.Save(eventsToSave); err != nil {
		d.loggerAdapter.Error(errorMessage, map[string]any{
			"error":       err.Error(),
			"eventsCount": d.queue.Len(),
		})
	}
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

	d.wg.Go(func() {
		for {
			select {
			case <-tickerChan:
				d.Flush()
			case <-d.stopChan:
				return
			}
		}
	})
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
	jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
	return backoff + jitter
}
