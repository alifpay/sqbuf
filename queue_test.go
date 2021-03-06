package sqbuf

import (
	"context"
	"sync"
	"testing"
	"time"
)

var storage sync.Map

func batchInsert(data [][]interface{}) {
	for _, v := range data {
		key, _ := v[0].(int)
		storage.Store(key, v)
	}
}

func TestQueue(t *testing.T) {
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
	time.Sleep(5 * time.Second)
	cancel()
	wg.Wait()
	cnt := 0
	storage.Range(func(k, v interface{}) bool {
		cnt++
		return true
	})
	if cnt != 10000 {
		t.Error("less than 10000")
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

func batchInsert2(data [][]interface{}) {
	for _, v := range data {
		key, _ := v[0].(string)
		storage.Store(key, v)
	}
}
