package sqbuf

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Queue -
type Queue struct {
	interval int
	fn       func(ctx context.Context, data [][]any)
	size     uint32
	rows     [][]any
	mu       sync.RWMutex
	closed   uint32
	async    bool            // If true, process data asynchronously
	ctx      context.Context // Context for processing functions
}

// New - will allocate, initialize, and return a slice queue buffer
// dataRows - data slice capacity
// interval - flush interval
// save - function for process a data slice(save to a storage)
func New(dataRows uint32, interval int, save func(ctx context.Context, data [][]any)) *Queue {
	return &Queue{
		interval: interval,
		fn:       save,
		size:     dataRows,
		rows:     make([][]any, 0, dataRows),
		async:    false,
	}
}

// NewAsync - creates a queue with asynchronous data processing
func NewAsync(dataRows uint32, interval int, save func(ctx context.Context, data [][]any)) *Queue {
	return &Queue{
		interval: interval,
		fn:       save,
		size:     dataRows,
		rows:     make([][]any, 0, dataRows),
		async:    true,
	}
}

// Add items to data slice
func (q *Queue) Add(items ...any) error {
	if atomic.LoadUint32(&q.closed) != 0 {
		return nil // Queue is closed, ignore new items
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if we need to flush before adding
	if uint32(len(q.rows)) >= q.size {
		q.unsafeFlush()
	}

	q.rows = append(q.rows, items)
	return nil
}

// flush - sends collected data to a storage
func (q *Queue) flush() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.unsafeFlush()
	return nil
}

// Flush - public method to manually flush the queue
func (q *Queue) Flush() error {
	return q.flush()
}

// unsafeFlush - internal flush without locking (must be called with lock held)
func (q *Queue) unsafeFlush() {
	if len(q.rows) > 0 {
		// Create a copy to avoid race conditions
		data := make([][]any, len(q.rows))
		copy(data, q.rows)

		// Reset the slice
		q.rows = q.rows[:0]

		// Process data based on async flag
		if q.async {
			go q.fn(q.ctx, data)
		} else {
			// Process synchronously to ensure immediate completion
			q.fn(q.ctx, data)
		}
	}
}

// Run - timer for periodical to flush, save and finish gracefully
func (q *Queue) Run(ctx context.Context, wg *sync.WaitGroup) {
	// Store context for use in flush operations
	q.ctx = ctx

	t := time.NewTicker(time.Millisecond * time.Duration(q.interval))
	defer t.Stop()

	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				// Mark as closed to prevent new additions
				atomic.StoreUint32(&q.closed, 1)
				// Final flush - do it synchronously for graceful shutdown
				q.mu.Lock()
				if len(q.rows) > 0 {
					data := make([][]any, len(q.rows))
					copy(data, q.rows)
					q.rows = q.rows[:0]
					// Process synchronously to ensure completion
					q.fn(q.ctx, data)
				}
				q.mu.Unlock()
				return
			case <-t.C:
				q.flush()
			}
		}
	}()
}

// Len returns the current number of items in the queue
func (q *Queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.rows)
}

// IsClosed returns true if the queue is closed
func (q *Queue) IsClosed() bool {
	return atomic.LoadUint32(&q.closed) != 0
}

// Cap returns the capacity of the queue
func (q *Queue) Cap() uint32 {
	return q.size
}

// FlushWithTimeout flushes the queue with a custom timeout context
func (q *Queue) FlushWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	originalCtx := q.ctx
	q.ctx = ctx
	defer func() { q.ctx = originalCtx }()

	return q.flush()
}

// SetContext updates the context used for processing functions
func (q *Queue) SetContext(ctx context.Context) {
	q.ctx = ctx
}
