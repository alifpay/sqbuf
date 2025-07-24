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

func batchInsert(ctx context.Context, data [][]interface{}) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return // Don't process if cancelled
	default:
	}

	count := int32(len(data))
	atomic.AddInt32(&processedCount, count)

	for _, v := range data {
		key, _ := v[0].(int)
		storage.Store(key, v)
	}
}

func TestQueue(t *testing.T) {
	// Clear storage before test
	storage = sync.Map{}
	atomic.StoreInt32(&processedCount, 0)

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	qq := New(1000, 200, batchInsert)

	wg.Add(1)
	qq.Run(ctx, &wg)
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

	// Добавим принудительный flush перед завершением
	qq.Flush()

	time.Sleep(5 * time.Second)
	cancel()
	wg.Wait()
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

	wg := sync.WaitGroup{}
	processed := int32(0)

	// Function that respects context cancellation
	processFunc := func(ctx context.Context, data [][]interface{}) {
		select {
		case <-ctx.Done():
			t.Log("Processing cancelled due to context")
			return
		default:
		}

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
		atomic.AddInt32(&processed, int32(len(data)))
	}

	qq := New(10, 50, processFunc)

	wg.Add(1)
	qq.Run(ctx, &wg)

	// Add some data
	for i := 0; i < 100; i++ {
		err := qq.Add(i, "test data")
		if err != nil {
			t.Error(err)
		}
	}

	// Wait for context timeout
	wg.Wait()

	processedCount := atomic.LoadInt32(&processed)
	t.Logf("Processed %d items before context cancellation", processedCount)

	// Should have processed some but not necessarily all due to timeout
	if processedCount == 0 {
		t.Error("Expected some items to be processed")
	}
}

func BenchmarkQueue(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	qq := New(1000, 200, batchInsert2)

	wg.Add(1)
	qq.Run(ctx, &wg)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := qq.Add(time.Now().UnixNano(), "name surname", time.Now())
			if err != nil {
				b.Error(err)
			}
		}
	})
	cancel()
	wg.Wait()
}

func batchInsert2(ctx context.Context, data [][]interface{}) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return // Don't process if cancelled
	default:
	}

	for _, v := range data {
		key := v[0] // Accept any type as key
		storage.Store(key, v)
	}
}
