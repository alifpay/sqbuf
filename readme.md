# Slice Queue Buffer

`sqbuf` is a lightweight batching helper for Go. It collects rows in-memory and periodically passes full batches to a user-defined function. The buffer is reused with a `sync.Pool`, keeping GC pressure minimal while remaining goroutine-safe.

## Features

- **Automatic batching**: Flushes when the buffer reaches capacity or the interval elapses
- **Safe concurrency**: Mutex-protected writes with minimal critical sections
- **Pooled buffers**: Reuses backing slices via `sync.Pool` to reduce allocations
- **Panic guard**: Recovers from panics inside the processing function and logs them
- **Context-aware runner**: `Run` listens for context cancellation and performs a final flush

## Installation

```bash
go get github.com/alifpay/sqbuf
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/alifpay/sqbuf"
)

func main() {
    // processFunc is invoked with every flushed batch.
    processFunc := func(batch [][]any) {
        fmt.Printf("flushing %d rows\n", len(batch))
        // persist batch to storage, send to API, etc.
    }

    // Create a queue that holds up to 500 rows and flushes every 250ms.
    queue := sqbuf.New(500, 250, processFunc)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Start periodic flushing in a background goroutine.
    go queue.Run(ctx)

    // Produce rows. Each row can contain arbitrary values.
    for i := 0; i < 2000; i++ {
        queue.Add(i, fmt.Sprintf("row-%d", i), time.Now())
    }

    // Stop the runner and wait a moment for the final flush.
    cancel()
    time.Sleep(100 * time.Millisecond)
}
```

## How It Works

- `New(capacity, intervalMS, processFunc)` creates a queue with the provided slice capacity and flush interval (milliseconds).
- `Add(items...)` appends a row (a variadic slice of `any`) to the in-memory buffer. When the buffer reaches capacity it flushes immediately.
- `Run(ctx)` starts a ticker that flushes on every interval tick. When `ctx` is cancelled, it triggers one last flush and waits for all pending batches to be processed before returning.
- During flushing the live buffer is swapped with a fresh pooled slice. The old slice is processed in a separate goroutine and returned to the pool afterward.

## Use Cases

- Bulk database or API writes
- Log aggregation
- Metrics collection pipelines
- Buffered event dispatching

## Notes

- The processing function must be safe for concurrent execution because flushes happen in a dedicated goroutine.
- If a panic occurs inside the processing function it is recovered, logged, and the buffer is recycled so subsequent batches continue processing.
