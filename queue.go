package sqbuf

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Workiva/go-datastructures/queue"
)

//Queue -
type Queue struct {
	rb       *queue.RingBuffer
	interval int
	fn       func(data [][]interface{})
	index    uint32
	size     uint32
	rows     [][]interface{}
	mu       sync.Mutex
}

//New -
func New(queueSize uint64, dataRows uint32, interval int, job func(data [][]interface{})) *Queue {
	return &Queue{
		rb:       queue.NewRingBuffer(queueSize),
		interval: interval,
		fn:       job,
		size:     dataRows,
		rows:     make([][]interface{}, 0, dataRows),
	}
}

//Add - items to buffer
func (q *Queue) Add(items ...interface{}) error {
	ix := atomic.LoadUint32(&q.index)
	if ix == q.size {
		err := q.enQueue()
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

//flush - get from queue and send job function
func (q *Queue) flush() error {
	val, err := q.rb.Get()
	if err != nil {
		return err
	}
	if data, ok := val.([][]interface{}); ok {
		q.fn(data)
	}
	return nil
}

//enQueue -
func (q *Queue) enQueue() error {
	ix := atomic.LoadUint32(&q.index)
	if ix > 0 {
		q.mu.Lock()
		err := q.rb.Put(q.rows)
		if err != nil {
			return err
		}
		q.rows = make([][]interface{}, 0, q.size)
		q.mu.Unlock()
		atomic.StoreUint32(&q.index, 0)
		go q.flush()
	}
	return nil
}

//Run -
func (q *Queue) Run(ctx context.Context, wg *sync.WaitGroup) {
	t := time.NewTicker(time.Millisecond * time.Duration(q.interval))
	go func() {
		for {
			select {
			case <-ctx.Done():
				q.enQueue()
				wg.Done()
				return
			case <-t.C:
				q.enQueue()
			}
		}
	}()
}
