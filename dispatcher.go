package ripple

import (
	"math/rand"
	"sync"
	"time"

	"github.com/Tap30/ripple-go/adapters"
)

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
	timerMu        sync.Mutex
}

func NewDispatcher(config DispatcherConfig, httpAdapter HTTPAdapter, storageAdapter StorageAdapter, headers map[string]string) *Dispatcher {
	return &Dispatcher{
		config:         config,
		queue:          NewQueue(),
		httpAdapter:    httpAdapter,
		storageAdapter: storageAdapter,
		loggerAdapter:  adapters.NewPrintLoggerAdapter(adapters.LogLevelWarn),
		headers:        headers,
		stopChan:       make(chan struct{}),
		flushMutex:     NewMutex(),
	}
}

// SetLoggerAdapter sets a custom logger adapter
func (d *Dispatcher) SetLoggerAdapter(logger LoggerAdapter) {
	d.loggerAdapter = logger
}

func (d *Dispatcher) Start() error {
	events, err := d.storageAdapter.Load()
	if err != nil {
		return err
	}
	d.queue.LoadFromSlice(events)

	// Don't start timer yet - wait for first new event
	return nil
}

func (d *Dispatcher) Enqueue(event Event) {
	d.queue.Enqueue(event)

	// Start timer on first new event
	d.startTimerIfNeeded()

	if d.queue.Len() >= d.config.MaxBatchSize {
		go d.Flush()
	}
}

func (d *Dispatcher) startTimerIfNeeded() {
	d.timerMu.Lock()
	defer d.timerMu.Unlock()

	if !d.timerStarted {
		d.ticker = time.NewTicker(d.config.FlushInterval)
		d.timerStarted = true
		d.wg.Go(func() {
			for {
				select {
				case <-d.ticker.C:
					d.Flush()
				case <-d.stopChan:
					return
				}
			}
		})
	}
}

func (d *Dispatcher) Flush() {
	d.flushMutex.RunAtomic(func() error {
		// Early return if queue is empty
		if d.queue.IsEmpty() {
			return nil
		}

		d.loggerAdapter.Debug("Starting flush operation")

		// Get all events and clear queue
		allEvents := d.queue.ToSlice()
		d.queue.Clear()

		// Process events in batches
		for i := 0; i < len(allEvents); i += d.config.MaxBatchSize {
			end := i + d.config.MaxBatchSize
			if end > len(allEvents) {
				end = len(allEvents)
			}
			batch := allEvents[i:end]

			d.loggerAdapter.Debug("Sending batch of %d events", len(batch))
			if err := d.sendWithRetry(batch); err != nil {
				d.loggerAdapter.Error("Failed to send batch: %v", err)
				// sendWithRetry handles re-queuing internally for 5xx and network errors
			} else {
				d.loggerAdapter.Debug("Successfully sent batch of %d events", len(batch))
			}
		}

		return nil
	})
}

func (d *Dispatcher) sendWithRetry(events []Event) error {
	return d.sendWithRetryAttempt(events, 0)
}

func (d *Dispatcher) sendWithRetryAttempt(events []Event, attempt int) error {
	d.loggerAdapter.Debug("Sending HTTP request, attempt %d/%d", attempt+1, d.config.MaxRetries+1)

	resp, err := d.httpAdapter.Send(d.config.Endpoint, events, d.headers)

	if err != nil {
		// Network error
		d.loggerAdapter.Error("Network error occurred: %v", err)

		if attempt < d.config.MaxRetries {
			d.loggerAdapter.Warn("Network error, retrying", map[string]any{
				"attempt":    attempt + 1,
				"maxRetries": d.config.MaxRetries,
				"error":      err.Error(),
			})

			backoff := time.Duration(1<<attempt) * time.Second
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			sleepDuration := backoff + jitter
			d.loggerAdapter.Debug("Retrying in %v", sleepDuration)
			time.Sleep(sleepDuration)

			return d.sendWithRetryAttempt(events, attempt+1)
		} else {
			// Max retries reached for network error - re-queue and persist
			d.loggerAdapter.Error("Network error, max retries reached", map[string]any{
				"maxRetries":  d.config.MaxRetries,
				"eventsCount": len(events),
				"error":       err.Error(),
			})

			// Re-queue events at the front
			for i := len(events) - 1; i >= 0; i-- {
				d.queue.Enqueue(events[i])
			}

			// Persist all events
			allEvents := d.queue.ToSlice()
			if len(allEvents) > 0 {
				d.storageAdapter.Save(allEvents)
			}

			return err
		}
	}

	// HTTP response received
	if resp.Status >= 200 && resp.Status < 300 {
		// 2xx: Success - clear storage
		d.loggerAdapter.Debug("HTTP request successful, clearing storage")
		d.storageAdapter.Clear()
		return nil
	} else if resp.Status >= 400 && resp.Status < 500 {
		// 4xx: Client error - no retry, drop events
		d.loggerAdapter.Warn("4xx client error, dropping events", map[string]any{
			"status":      resp.Status,
			"eventsCount": len(events),
		})

		d.storageAdapter.Clear()
		return nil // Don't return error for 4xx - events are intentionally dropped
	} else if resp.Status >= 500 {
		// 5xx: Server error - retry with backoff
		if attempt < d.config.MaxRetries {
			d.loggerAdapter.Warn("5xx server error, retrying", map[string]any{
				"status":     resp.Status,
				"attempt":    attempt + 1,
				"maxRetries": d.config.MaxRetries,
			})

			backoff := time.Duration(1<<attempt) * time.Second
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			sleepDuration := backoff + jitter
			d.loggerAdapter.Debug("Retrying in %v", sleepDuration)
			time.Sleep(sleepDuration)

			return d.sendWithRetryAttempt(events, attempt+1)
		} else {
			// 5xx: Max retries reached - re-queue and persist
			d.loggerAdapter.Error("5xx server error, max retries reached", map[string]any{
				"status":      resp.Status,
				"maxRetries":  d.config.MaxRetries,
				"eventsCount": len(events),
			})

			// Re-queue events at the front
			for i := len(events) - 1; i >= 0; i-- {
				d.queue.Enqueue(events[i])
			}

			// Persist all events
			allEvents := d.queue.ToSlice()
			if len(allEvents) > 0 {
				d.storageAdapter.Save(allEvents)
			}

			return &HTTPError{Status: resp.Status}
		}
	} else {
		// Unexpected status code - treat as server error
		d.loggerAdapter.Warn("Unexpected status code: %d", resp.Status)
		return &HTTPError{Status: resp.Status}
	}
}

func (d *Dispatcher) Stop() error {
	if d.ticker != nil {
		d.ticker.Stop()
	}
	close(d.stopChan)
	d.wg.Wait()

	d.Flush()

	events := d.queue.ToSlice()
	if len(events) > 0 {
		return d.storageAdapter.Save(events)
	}
	return nil
}

// StopWithoutFlush stops the dispatcher and persists events to storage without flushing to server
func (d *Dispatcher) StopWithoutFlush() error {
	if d.ticker != nil {
		d.ticker.Stop()
	}
	close(d.stopChan)
	d.wg.Wait()

	// Skip flush, just save events to storage
	events := d.queue.ToSlice()
	if len(events) > 0 {
		return d.storageAdapter.Save(events)
	}
	return nil
}
