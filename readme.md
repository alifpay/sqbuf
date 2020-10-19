Slice queue buffer

We use this buffer to save batch logs to clickhouse.

```Go


func main() {

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	qq := sqbuf.New(1000, 200, batchInsert)

	wg.Add(1)
	qq.Run(ctx, &wg)

	for i := 1; i < 10001; i++ {
		err := qq.Add(i, "name surname", time.Now())
		if err != nil {
			fmt.Println(err)
		}
	}

	cancel()
	wg.Wait()
}

func batchInsert(data [][]interface{}) {
	for k, v := range data {
		fmt.Println(k, v)
	}
}


```

Example repo with clickhouse

https://github.com/alifpay/clickhz
