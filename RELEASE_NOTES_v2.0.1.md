# Release v2.0.1

## 🎉 Major Release - Complete Rewrite with Context Support

This release includes significant improvements to reliability, performance, and API design.

## ✨ New Features

### Context Support
- Full `context.Context` integration for all operations
- Proper context cancellation handling
- Timeout support for flush operations
- No context leaks - all contexts are properly cancelled

### Error Handling
- Processing functions now return `error` for proper error propagation
- `ErrQueueClosed` returned when adding to closed queue
- Graceful error handling during shutdown

### Asynchronous Mode
- `NewAsync()` for high-performance non-blocking processing
- Background goroutines tracked via `WaitGroup`
- Graceful shutdown waits for all async operations
- **22% faster than synchronous mode** (492 ns/op vs 635 ns/op)

### Metrics & Monitoring
- `Stats()` method returns processing statistics
- Track total added, flushed items, and flush count
- Real-time visibility into queue state

### Improved API
- `Close(ctx)` - Explicit graceful shutdown
- `Flush(ctx)` - Manual flush with context support
- `SetFlushTimeout(duration)` - Configurable flush timeout
- `Len()`, `Cap()`, `IsClosed()` - Queue introspection

## 🐛 Bug Fixes

### Critical Fixes
- **Fixed context leak** in Run() loop - contexts now properly cancelled after each use
- **Fixed data loss in async mode** - async operations use background context
- **Fixed race condition** - closed flag check after acquiring lock
- **No more goroutine leaks** - all async operations tracked with WaitGroup

## 🚀 Performance

- **Synchronous mode**: ~635 ns/op, 195 B/op, 5 allocs/op
- **Asynchronous mode**: ~492 ns/op, 195 B/op, 5 allocs/op
- **Test coverage**: 60.2%
- **Race detector**: All tests pass

## 📝 API Changes

### Breaking Changes

Function signatures now return `error`:

```go
// Old (v1)
func process(ctx context.Context, data [][]any)

// New (v2)
func process(ctx context.Context, data [][]any) error
```

`Run()` no longer requires `WaitGroup`:

```go
// Old (v1)
wg.Add(1)
queue.Run(ctx, &wg)

// New (v2)
go queue.Run(ctx)
```

### Migration Guide

Update your processing functions to return `error`:

```go
// Old code
processFunc := func(ctx context.Context, data [][]any) {
    // process...
}

// New code
processFunc := func(ctx context.Context, data [][]any) error {
    // process...
    return nil // or return error
}
```

Or use legacy wrapper for backwards compatibility:

```go
queue := sqbuf.NewLegacy(1000, 200, oldProcessFunc)
```

## 📚 Examples

### Basic Usage

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

processFunc := func(ctx context.Context, data [][]any) error {
    // Your processing logic
    return saveToDatabase(data)
}

queue := sqbuf.New(1000, 200, processFunc)
go queue.Run(ctx)

// Add items
for i := 0; i < 10000; i++ {
    queue.Add(i, "data")
}

// Graceful shutdown
if err := queue.Close(context.Background()); err != nil {
    log.Printf("Error during shutdown: %v", err)
}
```

### Async Mode

```go
queue := sqbuf.NewAsync(1000, 200, processFunc)
go queue.Run(ctx)

// Add() returns immediately - processing happens in background
queue.Add(items...)
```

### Get Statistics

```go
added, flushed, flushCount := queue.Stats()
fmt.Printf("Added: %d, Flushed: %d, Flush count: %d\n", 
    added, flushed, flushCount)
```

## 🔗 Links

- **Repository**: https://github.com/alifpay/sqbuf
- **Documentation**: See [README.md](README.md)
- **Example usage**: See [examples/](examples/)

## 🙏 Acknowledgments

Thanks to all contributors and users for feedback and bug reports!

## 📦 Installation

```bash
go get github.com/alifpay/sqbuf/v2@v2.0.1
```

## Full Changelog

- Add context support for all operations
- Add error handling to processing functions
- Add async mode with background processing
- Add metrics and statistics tracking
- Add graceful shutdown with Close() method
- Add configurable flush timeout
- Fix context leaks in Run() loop
- Fix race conditions in closed flag check
- Fix data loss in async mode
- Improve test coverage to 60.2%
- Add comprehensive async mode tests
- Add benchmarks for sync and async modes
- Update documentation and examples
