# Slice Queue Buffer

A high-performance buffered queue for Go that accumulates data and processes it in batches. Perfect for bulk operations like database inserts, log aggregation, or any scenario where you need to batch process data efficiently.

## Features

- **Context Support**: Full context integration for cancellation and timeout handling
- **Error Handling**: Proper error propagation from processing functions
- **Synchronous & Asynchronous Processing**: Choose between immediate or background processing
- **Automatic Batching**: Configurable batch size and flush intervals  
- **Graceful Shutdown**: Ensures all data is processed before termination
- **Thread-Safe**: Concurrent-safe operations with optimized locking
- **Metrics**: Built-in statistics tracking
- **No Context Leaks**: Proper context cancellation in all operations
- **Legacy Compatibility**: Backward compatible with existing code

## Installation

```bash
go get github.com/alifpay/sqbuf/v2
```

## Quick Start

### Basic Usage with Context and Error Handling

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/alifpay/sqbuf/v2"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create a queue that processes data with context and error handling
    processFunc := func(ctx context.Context, data [][]any) error {
        fmt.Printf("Processing batch of %d items\n", len(data))
        for _, item := range data {
            select {
            case <-ctx.Done():
                return ctx.Err() // Handle cancellation
            default:
                // Your batch processing logic here
                if err := processItem(item); err != nil {
                    return err // Return errors for proper handling
                }
            }
        }
        return nil
    }

    // Create queue: 1000 items capacity, flush every 200ms
    queue := sqbuf.New(1000, 200, processFunc)

    go queue.Run(ctx)

    // Add items to the queue
    for i := 1; i <= 10000; i++ {
        err := queue.Add(i, "name surname", time.Now())
        if err != nil {
            fmt.Printf("Failed to add item: %v\n", err)
        }
    }

    // Graceful shutdown
    cancel()
    time.Sleep(100 * time.Millisecond)
    
    // Or use Close for explicit shutdown
    if err := queue.Close(context.Background()); err != nil {
        fmt.Printf("Error during shutdown: %v\n", err)
    }
}
```

### Asynchronous Processing

For high-throughput scenarios where you want non-blocking operation:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Create an asynchronous queue
processFunc := func(ctx context.Context, data [][]any) error {
    // This runs in a separate goroutine
    fmt.Printf("Async processing batch of %d items\n", len(data))
    
    // Simulate time-consuming operation (e.g., database write)
    if err := saveToDatabase(data); err != nil {
        return err
    }
    
    return nil
}

queue := sqbuf.NewAsync(1000, 200, processFunc)
go queue.Run(ctx)

// Add() returns immediately without waiting for processing
for i := 1; i <= 10000; i++ {
    queue.Add(i, "data")
}

// Graceful shutdown - waits for all async operations to complete
if err := queue.Close(context.Background()); err != nil {
    fmt.Printf("Error during shutdown: %v\n", err)
}
```

**Key differences in async mode:**
- `Add()` returns immediately after adding to buffer
- Processing happens in background goroutines
- Much faster for high-throughput scenarios
- All async operations are tracked via WaitGroup
- `Close()` waits for all background operations to complete
- Uses `context.Background()` for processing to prevent cancellation

**Performance comparison:**
- Sync mode: ~635 ns/op
- Async mode: ~492 ns/op (22% faster)

### Statistics

```go
added, flushed, flushCount := queue.Stats()
fmt.Printf("Added: %d, Flushed: %d, Flush count: %d\n", added, flushed, flushCount)
```

### Legacy Compatibility

```go
// For existing code without context/error support
legacyFunc := func(data [][]any) {
    // Your existing processing logic
}

queue := sqbuf.NewLegacy(1000, 200, legacyFunc)
```

## API Reference

### Queue Creation

- `New(capacity, interval, processFunc)` - Creates a synchronous queue with context and error support
- `NewAsync(capacity, interval, processFunc)` - Creates an asynchronous queue  
- `NewLegacy(capacity, interval, processFunc)` - Creates a queue compatible with old API

### Queue Operations

- `Add(items...)` - Adds items to the queue, returns `ErrQueueClosed` if queue is closed
- `Run(ctx)` - Starts the queue processing loop (run as goroutine)
- `Flush(ctx)` - Manually triggers immediate processing with context
- `Close(ctx)` - Gracefully closes queue and waits for all operations
- `SetFlushTimeout(duration)` - Sets timeout for flush operations (default: 5s)

### Queue Information

- `Len()` - Returns current number of items in queue
- `Cap()` - Returns queue capacity
- `IsClosed()` - Returns true if queue is closed
- `Stats()` - Returns (totalAdded, totalFlushed, flushCount)

## Improvements in v2

### ✅ Fixed Critical Issues

1. **No Context Leaks**: Fixed context leak in Run() loop - contexts are now properly cancelled after each use
2. **No Data Loss in Async Mode**: Async operations use background context to prevent cancellation affecting data processing
3. **Proper Error Handling**: Processing functions return errors for proper error propagation
4. **Graceful Shutdown**: WaitGroup tracks all async operations ensuring no data loss on shutdown
5. **Thread-Safe Closed Check**: Closed flag is checked after acquiring lock to prevent race conditions

### 🚀 New Features

1. **Metrics**: Track totalAdded, totalFlushed, and flushCount
2. **Configurable Timeout**: SetFlushTimeout allows custom flush timeouts
3. **Close Method**: Explicit graceful shutdown with error handling
4. **ErrQueueClosed**: Proper error returned when adding to closed queue

### 📊 Performance

- **~646 ns/op** - Fast operation speed
- **193 B/op** - Low memory allocation per operation
- **5 allocs/op** - Minimal allocations

## Use Cases

We use sqbuf for:
- **Database bulk inserts** (especially ClickHouse)
- **Log aggregation and batching**
- **Metrics collection**
- **Event processing pipelines**

Example repo with ClickHouse integration: https://github.com/alifpay/clickhz

## Migration from v1

If you're using the old API without error handling:

```go
// Old v1 API
func processData(ctx context.Context, data [][]any) {
    // process...
}

// New v2 API - add error return
func processData(ctx context.Context, data [][]any) error {
    // process...
    return nil // or return error
}

// Or use legacy wrapper
queue := sqbuf.NewLegacy(1000, 200, oldProcessFunc)
```
