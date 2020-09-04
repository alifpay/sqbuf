package sqbuf

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

//Queue -
type Queue struct {
	interval int
	fn       func(data [][]interface{})
	index    uint32
	size     uint32
	rows     [][]interface{}
	mu       sync.Mutex
}

//New - will allocate, initialize, and return a slice queue buffer
//queueSize - ring buffer size
//dataRows - data slice capacity
//interval - flush interval
//fun - function for process a data slice
func New(dataRows uint32, interval int, fun func(data [][]interface{})) *Queue {
	return &Queue{
		interval: interval,
		fn:       fun,
		size:     dataRows,
		rows:     make([][]interface{}, 0, dataRows),
	}
}

//Add - items to data slice
func (q *Queue) Add(items ...interface{}) error {
	ix := atomic.LoadUint32(&q.index)
	if ix == q.size {
		err := q.flush()
		if err != nil {
			return err
		}
	}

	q.mu.Lock()
	q.rows = append(q.rows, items)
	q.mu.Unlock()
	atomic.AddUint32(&q.index, 1)

	return nil
}

//flush - sends collected data to a storage
func (q *Queue) flush() error {
	ix := atomic.LoadUint32(&q.index)
	if ix > 0 {
		q.mu.Lock()
		go q.fn(q.rows)
		q.rows = make([][]interface{}, 0, q.size)
		q.mu.Unlock()
		atomic.StoreUint32(&q.index, 0)
	}
	return nil
}

//Run - timer for periodical savings data, save and finish gracefully
func (q *Queue) Run(ctx context.Context, wg *sync.WaitGroup) {
	t := time.NewTicker(time.Millisecond * time.Duration(q.interval))
	go func() {
		for {
			select {
			case <-ctx.Done():
				q.flush()
				wg.Done()
				return
			case <-t.C:
				q.flush()
			}
		}
	}()
}
