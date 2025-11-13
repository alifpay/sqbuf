package sqbuf

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var storage sync.Map
var processedCount int32

func batchInsert(ctx context.Context, data [][]interface{}) error {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	count := int32(len(data))
	atomic.AddInt32(&processedCount, count)

	for _, v := range data {
		key, _ := v[0].(int)
		storage.Store(key, v)
	}
	return nil
}

func TestQueue(t *testing.T) {
	// Clear storage before test
	storage = sync.Map{}
	atomic.StoreInt32(&processedCount, 0)

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	qq := New(1000, 200, batchInsert)

	go qq.Run(ctx)
	wg.Add(1)
	go func() {
		for i := 1; i < 1500; i++ {
			err := qq.Add(i, "name surname", time.Now())
			if err != nil {
				t.Error(err)
			}
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		for i := 1500; i < 5001; i++ {
			err := qq.Add(i, "name surname", time.Now())
			if err != nil {
				t.Error(err)
			}
		}
		wg.Done()
	}()

	for i := 5001; i < 10001; i++ {
		err := qq.Add(i, "name surname", time.Now())
		if err != nil {
			t.Error(err)
		}
	}

	time.Sleep(5 * time.Second)
	cancel()
	wg.Wait()

	// Wait a bit for final flush to complete
	time.Sleep(100 * time.Millisecond)

	cnt := 0
	storage.Range(func(k, v interface{}) bool {
		cnt++
		return true
	})
	processed := atomic.LoadInt32(&processedCount)
	t.Logf("Found %d items in storage, processed %d batches totaling items", cnt, processed)
	if cnt != 10000 {
		t.Error("less than 10000")
	}
}

func TestQueueWithContext(t *testing.T) {
	// Clear storage before test
	storage = sync.Map{}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	processed := int32(0)

	// Function that respects context cancellation
	processFunc := func(ctx context.Context, data [][]interface{}) error {
		select {
		case <-ctx.Done():
			t.Log("Processing cancelled due to context")
			return ctx.Err()
		default:
		}

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
		atomic.AddInt32(&processed, int32(len(data)))
		return nil
	}

	qq := New(10, 50, processFunc)

	go qq.Run(ctx)

	// Add some data
	for i := 0; i < 100; i++ {
		err := qq.Add(i, "test data")
		if err != nil {
			t.Error(err)
		}
	}

	// Wait for context timeout
	time.Sleep(2500 * time.Millisecond)

	processedCount := atomic.LoadInt32(&processed)
	t.Logf("Processed %d items before context cancellation", processedCount)

	// Should have processed some but not necessarily all due to timeout
	if processedCount == 0 {
		t.Error("Expected some items to be processed")
	}
}

func TestQueueAsync(t *testing.T) {
	// Clear storage before test
	storage = sync.Map{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Track async processing
	var asyncProcessed int32
	var processingStarted int32
	var processingFinished int32

	processFunc := func(ctx context.Context, data [][]interface{}) error {
		atomic.AddInt32(&processingStarted, 1)
		t.Logf("Async processing started for batch of %d items", len(data))

		// Simulate time-consuming async operation
		time.Sleep(100 * time.Millisecond)

		for _, v := range data {
			key, _ := v[0].(int)
			storage.Store(key, v)
		}

		atomic.AddInt32(&asyncProcessed, int32(len(data)))
		atomic.AddInt32(&processingFinished, 1)
		t.Logf("Async processing finished for batch of %d items", len(data))
		return nil
	}

	// Create async queue with small batch size to trigger multiple flushes
	qq := NewAsync(10, 50, processFunc)

	go qq.Run(ctx)

	// Add items quickly - should trigger multiple async flushes
	itemCount := 100
	for i := 1; i <= itemCount; i++ {
		err := qq.Add(i, "async test data", time.Now())
		if err != nil {
			t.Error(err)
		}
	}

	// Wait for periodic flush
	time.Sleep(200 * time.Millisecond)

	// Check stats
	added, flushed, flushCount := qq.Stats()
	t.Logf("Stats: Added=%d, Flushed=%d, FlushCount=%d", added, flushed, flushCount)

	// Verify that async processing was started
	started := atomic.LoadInt32(&processingStarted)
	if started == 0 {
		t.Error("No async processing was started")
	}
	t.Logf("Processing started %d times", started)

	// Cancel and wait for graceful shutdown
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Use Close to ensure all async operations complete
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel()

	if err := qq.Close(closeCtx); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Now all async operations should be complete
	finished := atomic.LoadInt32(&processingFinished)
	t.Logf("Processing finished %d times", finished)

	if started != finished {
		t.Errorf("Not all async operations finished: started=%d, finished=%d", started, finished)
	}

	// Verify all data was processed
	processed := atomic.LoadInt32(&asyncProcessed)
	t.Logf("Total items processed asynchronously: %d", processed)

	if processed != int32(itemCount) {
		t.Errorf("Expected %d items processed, got %d", itemCount, processed)
	}

	// Verify data in storage
	cnt := 0
	storage.Range(func(k, v interface{}) bool {
		cnt++
		return true
	})

	t.Logf("Items in storage: %d", cnt)
	if cnt != itemCount {
		t.Errorf("Expected %d items in storage, got %d", itemCount, cnt)
	}
}

func TestQueueAsyncVsSync(t *testing.T) {
	// Compare async vs sync performance
	itemCount := 50
	processingTime := 50 * time.Millisecond

	// Test synchronous queue
	t.Run("Synchronous", func(t *testing.T) {
		storage = sync.Map{}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var processCount int32
		processFunc := func(ctx context.Context, data [][]interface{}) error {
			time.Sleep(processingTime)
			atomic.AddInt32(&processCount, int32(len(data)))
			return nil
		}

		qq := New(10, 50, processFunc)
		go qq.Run(ctx)

		start := time.Now()
		for i := 1; i <= itemCount; i++ {
			qq.Add(i, "sync test")
		}

		time.Sleep(200 * time.Millisecond)
		cancel()
		time.Sleep(100 * time.Millisecond)

		duration := time.Since(start)
		processed := atomic.LoadInt32(&processCount)

		t.Logf("Sync mode: processed %d items in %v", processed, duration)
	})

	// Test asynchronous queue
	t.Run("Asynchronous", func(t *testing.T) {
		storage = sync.Map{}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var processCount int32
		processFunc := func(ctx context.Context, data [][]interface{}) error {
			time.Sleep(processingTime)
			atomic.AddInt32(&processCount, int32(len(data)))
			return nil
		}

		qq := NewAsync(10, 50, processFunc)
		go qq.Run(ctx)

		start := time.Now()
		for i := 1; i <= itemCount; i++ {
			qq.Add(i, "async test")
		}

		// In async mode, Add() returns immediately
		addDuration := time.Since(start)
		t.Logf("Async mode: Add() completed in %v", addDuration)

		// Wait for async processing
		time.Sleep(200 * time.Millisecond)
		cancel()

		closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer closeCancel()
		qq.Close(closeCtx)

		processed := atomic.LoadInt32(&processCount)
		t.Logf("Async mode: processed %d items", processed)

		if processed != int32(itemCount) {
			t.Errorf("Expected %d items processed, got %d", itemCount, processed)
		}
	})
}

func BenchmarkQueue(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	qq := New(1000, 200, batchInsert2)

	go qq.Run(ctx)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := qq.Add(time.Now().UnixNano(), "name surname", time.Now())
			if err != nil {
				b.Error(err)
			}
		}
	})
	cancel()
	time.Sleep(100 * time.Millisecond)
}

func BenchmarkQueueAsync(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	qq := NewAsync(1000, 200, batchInsert2)

	go qq.Run(ctx)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := qq.Add(time.Now().UnixNano(), "name surname", time.Now())
			if err != nil {
				b.Error(err)
			}
		}
	})
	cancel()

	// Wait for async operations to complete
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()
	qq.Close(closeCtx)
}

func batchInsert2(ctx context.Context, data [][]interface{}) error {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	for _, v := range data {
		key := v[0] // Accept any type as key
		storage.Store(key, v)
	}
	return nil
}
