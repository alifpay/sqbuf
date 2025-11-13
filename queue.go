package sqbuf

import (
	"context"
	"log"
	"sync"
	"time"
)

// Queue - will allocate, initialize, and return a slice queue buffer.
// The batch buffer is pooled using sync.Pool to minimize GC pressure.
type Queue struct {
	interval  int
	fn        func(data [][]any)
	size      uint32
	rows      [][]any // The live buffer for accumulating data
	mu        sync.Mutex
	batchPool sync.Pool
}

// New - will allocate, initialize, and return a slice queue buffer
// queueCap - data slice capacity
// interval - flush interval (in milliseconds)
// save - function for process a data slice (save to a storage)
func New(queueCap uint32, interval int, save func(data [][]any)) *Queue {
	q := &Queue{
		interval: interval,
		fn:       save,
		size:     queueCap,
		batchPool: sync.Pool{
			New: func() interface{} {
				return make([][]any, 0, queueCap)
			},
		},
	}

	// Get the initial buffer from the pool and reset its length to 0.
	q.rows = q.batchPool.Get().([][]any)
	q.rows = q.rows[:0]

	return q
}

// Add items to data slice
func (q *Queue) Add(items ...any) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check capacity using the length, which is protected by the mutex.
	if len(q.rows) >= int(q.size) {
		// Buffer is full, flush the current content before appending
		q.flushInternal()
	}

	// Append the new item row to the current live buffer.
	q.rows = append(q.rows, items)
}

// flush is the external entry point for timer-based flushes.
func (q *Queue) flush() {
	q.mu.Lock()
	q.flushInternal()
	q.mu.Unlock()
}

// flushInternal - performs the buffer swap under lock.
// MUST be called while q.mu is locked.
func (q *Queue) flushInternal() {
	if len(q.rows) == 0 {
		return
	}

	// 1. Swap the buffer: The old q.rows becomes the batch to flush.
	batchToFlush := q.rows

	// 2. Get a new, empty buffer from the pool for the next cycle.
	newRows := q.batchPool.Get().([][]any)

	// CRITICAL: Reset the new buffer's length to 0 (while retaining its capacity)
	// and assign it as the new live buffer.
	q.rows = newRows[:0]

	// 3. Process the batch in a goroutine and ensure the buffer is returned to the pool.
	go func() {
		// Recover from panics in the user-provided function to prevent app crash.
		defer func() {
			if r := recover(); r != nil {
				log.Printf("sqbuf: panic recovered in processing function: %v", r)
			}
			q.batchPool.Put(batchToFlush[:0])
		}()
		q.fn(batchToFlush)
	}()
}

// Run - timer for periodical to flush, save and finish gracefully
func (q *Queue) Run(ctx context.Context) {
	t := time.NewTicker(time.Millisecond * time.Duration(q.interval))
	defer t.Stop() // Always stop the ticker to prevent resource leaks

	for {
		select {
		case <-ctx.Done():
			q.flush() // Final flush before exiting
			return
		case <-t.C:
			q.flush()
		}
	}
}
