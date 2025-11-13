package sqbuf

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrQueueClosed is returned when trying to add to a closed queue
	ErrQueueClosed = errors.New("queue is closed")
)

// Queue -
type Queue struct {
	interval     int
	fn           func(ctx context.Context, data [][]any) error
	size         uint32
	rows         [][]any
	mu           sync.RWMutex
	closed       uint32
	async        bool           // If true, process data asynchronously
	wg           sync.WaitGroup // Tracks async operations for graceful shutdown
	flushTimeout time.Duration
	stats        struct {
		totalAdded   uint64
		totalFlushed uint64
		flushCount   uint64
	}
}

// New - will allocate, initialize, and return a slice queue buffer
// dataRows - data slice capacity
// interval - flush interval
// save - function for process a data slice(save to a storage)
func New(dataRows uint32, interval int, save func(ctx context.Context, data [][]any) error) *Queue {
	return &Queue{
		interval:     interval,
		fn:           save,
		size:         dataRows,
		rows:         make([][]any, 0, dataRows),
		async:        false,
		flushTimeout: 5 * time.Second,
	}
}

// NewAsync - creates a queue with asynchronous data processing
func NewAsync(dataRows uint32, interval int, save func(ctx context.Context, data [][]any) error) *Queue {
	return &Queue{
		interval:     interval,
		fn:           save,
		size:         dataRows,
		rows:         make([][]any, 0, dataRows),
		async:        true,
		flushTimeout: 5 * time.Second,
	}
}

// Add items to data slice
func (q *Queue) Add(items ...any) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if atomic.LoadUint32(&q.closed) != 0 {
		return ErrQueueClosed
	}

	// Check if we need to flush before adding
	if uint32(len(q.rows)) >= q.size {
		// In async mode, we must flush synchronously here to prevent buffer overflow
		// Create a copy to avoid holding the lock during processing
		data := make([][]any, len(q.rows))
		copy(data, q.rows)
		count := uint64(len(q.rows))
		q.rows = q.rows[:0]

		// Update stats
		atomic.AddUint64(&q.stats.totalFlushed, count)
		atomic.AddUint64(&q.stats.flushCount, 1)

		if q.async {
			// Launch async processing
			q.wg.Add(1)
			go func() {
				defer q.wg.Done()
				q.fn(context.Background(), data)
			}()
		} else {
			// Process synchronously with timeout
			timeoutCtx, cancel := context.WithTimeout(context.Background(), q.flushTimeout)
			err := q.fn(timeoutCtx, data)
			cancel()
			if err != nil {
				return err
			}
		}
	}

	q.rows = append(q.rows, items)
	atomic.AddUint64(&q.stats.totalAdded, 1)
	return nil
}

// flush - sends collected data to a storage
func (q *Queue) flush(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.unsafeFlush(ctx)
}

// unsafeFlush - internal flush without locking (must be called with lock held)
func (q *Queue) unsafeFlush(ctx context.Context) error {
	if len(q.rows) == 0 {
		return nil
	}

	// Create a copy to avoid race conditions
	data := make([][]any, len(q.rows))
	copy(data, q.rows)
	count := uint64(len(q.rows))

	// Reset the slice
	q.rows = q.rows[:0]

	// Update stats
	atomic.AddUint64(&q.stats.totalFlushed, count)
	atomic.AddUint64(&q.stats.flushCount, 1)

	// Process data based on async flag
	if q.async {
		q.wg.Add(1)
		go func() {
			defer q.wg.Done()
			// Use background context to prevent cancellation affecting data processing
			q.fn(context.Background(), data)
		}()
		return nil
	}

	// Process synchronously with provided context
	return q.fn(ctx, data)
}

// Run - timer for periodical to flush, save and finish gracefully
func (q *Queue) Run(ctx context.Context) {
	t := time.NewTicker(time.Millisecond * time.Duration(q.interval))
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			// Mark as closed to prevent new additions
			atomic.StoreUint32(&q.closed, 1)
			// Final flush - do it synchronously for graceful shutdown
			timeoutCtx, cancel := context.WithTimeout(context.Background(), q.flushTimeout)
			q.mu.Lock()
			if len(q.rows) > 0 {
				data := make([][]any, len(q.rows))
				copy(data, q.rows)
				count := uint64(len(q.rows))
				q.rows = q.rows[:0]
				atomic.AddUint64(&q.stats.totalFlushed, count)
				atomic.AddUint64(&q.stats.flushCount, 1)
				q.mu.Unlock()
				// Process synchronously to ensure completion
				q.fn(timeoutCtx, data)
			} else {
				q.mu.Unlock()
			}
			cancel()
			// Wait for all async operations to complete
			q.wg.Wait()
			return
		case <-t.C:
			timeoutCtx, cancel := context.WithTimeout(context.Background(), q.flushTimeout)
			q.flush(timeoutCtx)
			cancel() // Fix context leak: call immediately after use
		}
	}
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

// Close gracefully closes the queue and waits for all async operations
func (q *Queue) Close(ctx context.Context) error {
	atomic.StoreUint32(&q.closed, 1)

	q.mu.Lock()
	if len(q.rows) > 0 {
		data := make([][]any, len(q.rows))
		copy(data, q.rows)
		count := uint64(len(q.rows))
		q.rows = q.rows[:0]
		atomic.AddUint64(&q.stats.totalFlushed, count)
		atomic.AddUint64(&q.stats.flushCount, 1)
		q.mu.Unlock()

		if err := q.fn(ctx, data); err != nil {
			q.wg.Wait()
			return err
		}
	} else {
		q.mu.Unlock()
	}

	// Wait for all async operations to complete
	q.wg.Wait()
	return nil
}

// Stats returns queue statistics: (totalAdded, totalFlushed, flushCount)
func (q *Queue) Stats() (added, flushed, flushCount uint64) {
	return atomic.LoadUint64(&q.stats.totalAdded),
		atomic.LoadUint64(&q.stats.totalFlushed),
		atomic.LoadUint64(&q.stats.flushCount)
}

// SetFlushTimeout sets the timeout for flush operations
func (q *Queue) SetFlushTimeout(timeout time.Duration) {
	q.mu.Lock()
	q.flushTimeout = timeout
	q.mu.Unlock()
}

// Flush manually triggers a flush operation
func (q *Queue) Flush(ctx context.Context) error {
	return q.flush(ctx)
}
