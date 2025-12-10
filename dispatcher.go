package ripple

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

type Dispatcher struct {
	config         DispatcherConfig
	queue          *Queue
	httpAdapter    HTTPAdapter
	storageAdapter StorageAdapter
	headers        map[string]string
	ticker         *time.Ticker
	stopChan       chan struct{}
	flushMu        sync.Mutex
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
		headers:        headers,
		stopChan:       make(chan struct{}),
	}
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
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			for {
				select {
				case <-d.ticker.C:
					d.Flush()
				case <-d.stopChan:
					return
				}
			}
		}()
	}
}

func (d *Dispatcher) Flush() {
	d.flushMu.Lock()
	defer d.flushMu.Unlock()

	for !d.queue.IsEmpty() {
		batchSize := min(d.config.MaxBatchSize, d.queue.Len())
		batch := make([]Event, 0, batchSize)
		for i := 0; i < batchSize; i++ {
			if event, ok := d.queue.Dequeue(); ok {
				batch = append(batch, event)
			}
		}

		if len(batch) == 0 {
			break
		}

		if err := d.sendWithRetry(batch); err != nil {
			for _, event := range batch {
				d.queue.Enqueue(event)
			}
			break
		}
	}
}

func (d *Dispatcher) sendWithRetry(events []Event) error {
	var lastErr error
	for attempt := 0; attempt <= d.config.MaxRetries; attempt++ {
		resp, err := d.httpAdapter.Send(d.config.Endpoint, events, d.headers)
		if err == nil && resp.OK {
			d.storageAdapter.Clear()
			return nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = &HTTPError{Status: resp.Status}
		}

		if attempt < d.config.MaxRetries {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			time.Sleep(backoff + jitter)
		}
	}
	return lastErr
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
