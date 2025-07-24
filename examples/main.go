package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alifpay/sqbuf/v2"
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
	ctx := context.Background()
	wg := sync.WaitGroup{}

	// Create a synchronous queue with context-aware processing
	processFunc := func(ctx context.Context, data [][]any) {
		fmt.Printf("Processing batch of %d items\n", len(data))
		for _, item := range data {
			select {
			case <-ctx.Done():
				fmt.Println("Processing cancelled")
				return
			default:
				// Simulate processing
				fmt.Printf("  Processing item: %v\n", item[0])
			}
		}
	}

	queue := sqbuf.New(3, 100, processFunc) // Flush every 100ms or when 3 items

	wg.Add(1)
	queue.Run(ctx, &wg)

	// Add some items
	for i := 1; i <= 10; i++ {
		queue.Add(i, fmt.Sprintf("item-%d", i), time.Now())
		time.Sleep(50 * time.Millisecond)
	}

	// Manual flush to ensure all items are processed
	queue.Flush()
	time.Sleep(200 * time.Millisecond)
}

func asyncExample() {
	ctx := context.Background()
	wg := sync.WaitGroup{}

	// Create an asynchronous queue
	processFunc := func(ctx context.Context, data [][]any) {
		fmt.Printf("Async processing batch of %d items\n", len(data))
		// Simulate time-consuming operation
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Finished processing batch of %d items\n", len(data))
	}

	queue := sqbuf.NewAsync(2, 50, processFunc) // Async processing

	wg.Add(1)
	queue.Run(ctx, &wg)

	// Add items quickly
	for i := 1; i <= 6; i++ {
		queue.Add(i, fmt.Sprintf("async-item-%d", i))
	}

	// Wait for async processing to complete
	time.Sleep(500 * time.Millisecond)
}

func contextCancellationExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	wg := sync.WaitGroup{}

	processFunc := func(ctx context.Context, data [][]any) {
		select {
		case <-ctx.Done():
			fmt.Println("Processing cancelled due to context timeout")
			return
		default:
		}

		fmt.Printf("Processing batch of %d items before timeout\n", len(data))
		// Simulate slow processing
		time.Sleep(150 * time.Millisecond)
		fmt.Printf("Completed processing batch of %d items\n", len(data))
	}

	queue := sqbuf.New(5, 50, processFunc)

	wg.Add(1)
	queue.Run(ctx, &wg)

	// Add items
	for i := 1; i <= 10; i++ {
		err := queue.Add(i, fmt.Sprintf("timeout-item-%d", i))
		if err != nil {
			fmt.Printf("Failed to add item %d: %v\n", i, err)
		}
	}

	// Wait for context cancellation
	wg.Wait()
	fmt.Printf("Queue closed: %v\n", queue.IsClosed())
}

func legacyExample() {
	ctx := context.Background()
	wg := sync.WaitGroup{}

	// Legacy function without context
	legacyProcessFunc := func(data [][]any) {
		fmt.Printf("Legacy processing batch of %d items\n", len(data))
		for _, item := range data {
			fmt.Printf("  Legacy processing item: %v\n", item[0])
		}
	}

	// Use legacy wrapper
	queue := sqbuf.NewLegacy(3, 100, legacyProcessFunc)

	wg.Add(1)
	queue.Run(ctx, &wg)

	// Add some items
	for i := 1; i <= 5; i++ {
		queue.Add(i, fmt.Sprintf("legacy-item-%d", i))
	}

	queue.Flush()
	time.Sleep(200 * time.Millisecond)
}
