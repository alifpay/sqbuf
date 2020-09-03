package sqbuf

import (
	"fmt"
	"testing"
	"time"

	"github.com/Workiva/go-datastructures/queue"
)

func TestPutToFull(t *testing.T) {
	rb := queue.NewRingBuffer(3)

	data := make([][]interface{}, 0, 100)

	for i := 0; i < 100; i++ {
		data = append(data, []interface{}{i, "item", time.Now()})
	}

	fmt.Println(len(data))

	err := rb.Put(data)
	if err != nil {
		t.Error("put", err)
	}

	data = make([][]interface{}, 0, 100)

	result, err := rb.Get()
	rtn, ok := result.([][]interface{})
	if !ok {
		t.Error("Type assertions")
	}

	for k, v := range rtn {
		fmt.Println(k, v)
	}
}
