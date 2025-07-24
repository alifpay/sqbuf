# Slice Queue Buffer

A high-performance buffered queue for Go that accumulates data and processes it in batches. Perfect for bulk operations like database inserts, log aggregation, or any scenario where you need to batch process data efficiently.

## Features

- **Context Support**: Full context integration for cancellation and timeout handling
- **Synchronous & Asynchronous Processing**: Choose between immediate or background processing
- **Automatic Batching**: Configurable batch size and flush intervals  
- **Graceful Shutdown**: Ensures all data is processed before termination
- **Thread-Safe**: Concurrent-safe operations with optimized locking
- **Legacy Compatibility**: Backward compatible with existing code

## Installation

```bash
go get github.com/alifpay/sqbuf
```

## Quick Start

### Basic Usage with Context

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/alifpay/sqbuf"
)

func main() {
    ctx := context.Background()
    wg := sync.WaitGroup{}

    // Create a queue that processes data with context
    processFunc := func(ctx context.Context, data [][]any) {
        fmt.Printf("Processing batch of %d items\n", len(data))
        for _, item := range data {
            select {
            case <-ctx.Done():
                fmt.Println("Processing cancelled")
                return
            default:
                // Your batch processing logic here
                fmt.Printf("Processing: %v\n", item[0])
            }
        }
    }

    // Create queue: 1000 items capacity, flush every 200ms
    queue := sqbuf.New(1000, 200, processFunc)

    wg.Add(1)
    queue.Run(ctx, &wg)

    // Add items to the queue
    for i := 1; i <= 10000; i++ {
        err := queue.Add(i, "name surname", time.Now())
        if err != nil {
            fmt.Println(err)
        }
    }

    // Ensure all data is processed before shutdown
    queue.Flush()
    wg.Wait()
}
```

### Asynchronous Processing

```go
// For non-blocking, background processing
queue := sqbuf.NewAsync(1000, 200, processFunc)
```

### Legacy Compatibility

```go
// For existing code without context support
legacyFunc := func(data [][]any) {
    // Your existing processing logic
}

queue := sqbuf.NewLegacy(1000, 200, legacyFunc)
```

## API Reference

### Queue Creation

- `New(capacity, interval, processFunc)` - Creates a synchronous queue with context support
- `NewAsync(capacity, interval, processFunc)` - Creates an asynchronous queue  
- `NewLegacy(capacity, interval, processFunc)` - Creates a queue compatible with old API

### Queue Operations

- `Add(items...)` - Adds items to the queue
- `Run(ctx, wg)` - Starts the queue processing loop
- `Flush()` - Manually triggers immediate processing
- `FlushWithTimeout(duration)` - Flushes with custom timeout context

### Queue Information

- `Len()` - Returns current number of items in queue
- `Cap()` - Returns queue capacity
- `IsClosed()` - Returns true if queue is closed

## Use Cases

We use sqbuf for:
- **Database bulk inserts** (especially ClickHouse)
- **Log aggregation and batching**
- **Metrics collection**
- **Event processing pipelines**

Example repo with ClickHouse integration: https://github.com/alifpay/clickhz

## Optimizations

The latest version includes several performance improvements:

- **Context-aware processing** for better cancellation handling
- **Reduced memory allocations** through slice reuse
- **Improved synchronization** with reader-writer locks
- **Race condition fixes** in concurrent scenarios
- **Graceful shutdown** ensuring no data loss
