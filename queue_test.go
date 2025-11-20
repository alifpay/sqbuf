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

func batchInsert(data [][]any) {
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

	done := make(chan struct{})
	go func() {
		qq.Run(ctx)
		close(done)
	}()
	wg.Add(1)
	go func() {
		for i := 1; i < 1500; i++ {
			qq.Add(i, "name surname", time.Now())
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		for i := 1500; i < 5001; i++ {
			qq.Add(i, "name surname", time.Now())
		}
		wg.Done()
	}()

	for i := 5001; i < 10001; i++ {
		qq.Add(i, "name surname", time.Now())
	}

	wg.Wait() // Wait for all producers to finish
	cancel()  // Signal Run to stop
	<-done    // Wait for Run to finish (graceful shutdown)

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

func BenchmarkQueue(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	qq := New(1000, 200, batchInsert2)

	done := make(chan struct{})
	go func() {
		qq.Run(ctx)
		close(done)
	}()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			qq.Add(time.Now().UnixNano(), "name surname", time.Now())
		}
	})
	cancel()
	<-done
}

func batchInsert2(data [][]any) {
	for _, v := range data {
		key := v[0] // Accept any type as key
		storage.Store(key, v)
	}
}
