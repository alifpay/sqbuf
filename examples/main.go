package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	sqbuf "github.com/alifpay/sqbuf/v2"
)

func main() {
	// Example 1: Basic usage with context
	fmt.Println("=== Example 1: Basic usage with context ===")
	basicExample()

	// Example 2: Async processing
	fmt.Println("\n=== Example 2: Async processing ===")
	asyncExample()

	// Example 3: Context cancellation
	fmt.Println("\n=== Example 3: Context cancellation ===")
	contextCancellationExample()

	// Example 4: Legacy compatibility
	fmt.Println("\n=== Example 4: Legacy compatibility ===")
	legacyExample()
}

func basicExample() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a synchronous queue with context-aware processing
	processFunc := func(ctx context.Context, data [][]any) error {
		fmt.Printf("Processing batch of %d items\n", len(data))
		for _, item := range data {
			select {
			case <-ctx.Done():
				fmt.Println("Processing cancelled")
				return ctx.Err()
			default:
				// Simulate processing
				fmt.Printf("  Processing item: %v\n", item[0])
			}
		}
		return nil
	}

	queue := sqbuf.New(3, 100, processFunc) // Flush every 100ms or when 3 items

	go queue.Run(ctx)

	// Add some items
	for i := 1; i <= 10; i++ {
		queue.Add(i, fmt.Sprintf("item-%d", i), time.Now())
		time.Sleep(50 * time.Millisecond)
	}

	// Wait a bit for processing
	time.Sleep(500 * time.Millisecond)
	cancel()
	time.Sleep(200 * time.Millisecond)
}

func asyncExample() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processedBatches int32
	var processedItems int32

	// Create an asynchronous queue
	processFunc := func(ctx context.Context, data [][]any) error {
		batchNum := atomic.AddInt32(&processedBatches, 1)
		fmt.Printf("[Batch %d] Async processing started for %d items\n", batchNum, len(data))

		// Simulate time-consuming operation (e.g., database write)
		time.Sleep(100 * time.Millisecond)

		atomic.AddInt32(&processedItems, int32(len(data)))
		fmt.Printf("[Batch %d] Async processing finished for %d items\n", batchNum, len(data))
		return nil
	}

	queue := sqbuf.NewAsync(5, 50, processFunc) // Small batch to show multiple async operations

	go queue.Run(ctx)

	// Add items quickly - in async mode, Add() returns immediately
	fmt.Println("Adding items (async mode - returns immediately)...")
	start := time.Now()
	for i := 1; i <= 20; i++ {
		queue.Add(i, fmt.Sprintf("async-item-%d", i))
	}
	addDuration := time.Since(start)
	fmt.Printf("All items added in %v (without waiting for processing)\n", addDuration)

	// Wait a bit to see async processing in action
	time.Sleep(300 * time.Millisecond)

	// Get statistics
	added, flushed, flushCount := queue.Stats()
	fmt.Printf("Stats: Added=%d, Flushed=%d, FlushCount=%d\n", added, flushed, flushCount)

	// Graceful shutdown - waits for all async operations
	fmt.Println("Closing queue and waiting for all async operations...")
	cancel()

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel()

	if err := queue.Close(closeCtx); err != nil {
		fmt.Printf("Error during close: %v\n", err)
	}

	fmt.Printf("Total batches processed: %d\n", atomic.LoadInt32(&processedBatches))
	fmt.Printf("Total items processed: %d\n", atomic.LoadInt32(&processedItems))
}

func contextCancellationExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	processFunc := func(ctx context.Context, data [][]any) error {
		select {
		case <-ctx.Done():
			fmt.Println("Processing cancelled due to context timeout")
			return ctx.Err()
		default:
		}

		fmt.Printf("Processing batch of %d items before timeout\n", len(data))
		// Simulate slow processing
		time.Sleep(150 * time.Millisecond)
		fmt.Printf("Completed processing batch of %d items\n", len(data))
		return nil
	}

	queue := sqbuf.New(5, 50, processFunc)

	go queue.Run(ctx)

	// Add items
	for i := 1; i <= 10; i++ {
		err := queue.Add(i, fmt.Sprintf("timeout-item-%d", i))
		if err != nil {
			fmt.Printf("Failed to add item %d: %v\n", i, err)
		}
	}

	// Wait for context cancellation
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("Queue closed: %v\n", queue.IsClosed())
}

func legacyExample() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Legacy function without context
	legacyProcessFunc := func(data [][]any) {
		fmt.Printf("Legacy processing batch of %d items\n", len(data))
		for _, item := range data {
			fmt.Printf("  Legacy processing item: %v\n", item[0])
		}
	}

	// Use legacy wrapper
	queue := sqbuf.NewLegacy(3, 100, legacyProcessFunc)

	go queue.Run(ctx)

	// Add some items
	for i := 1; i <= 5; i++ {
		queue.Add(i, fmt.Sprintf("legacy-item-%d", i))
	}

	time.Sleep(500 * time.Millisecond)
	cancel()
	time.Sleep(200 * time.Millisecond)
}
