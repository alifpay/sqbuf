package main

import (
	"context"
	"fmt"
	"time"

	sqbuf "github.com/alifpay/sqbuf/v2"
)

func main() {
	// Example 1: Basic usage
	fmt.Println("=== Example 1: Basic usage ===")
	basicExample()

	// Example 2: Multiple goroutines
	fmt.Println("\n=== Example 2: Multiple goroutines ===")
	concurrentExample()
}

func basicExample() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a queue
	processFunc := func(data [][]any) {
		fmt.Printf("Processing batch of %d items\n", len(data))
		for _, item := range data {
			// Simulate processing
			fmt.Printf("  Processing item: %v\n", item[0])
		}
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

func concurrentExample() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processFunc := func(data [][]any) {
		fmt.Printf("Processing batch of %d items\n", len(data))
		// Simulate database insert
		time.Sleep(50 * time.Millisecond)
	}

	queue := sqbuf.New(100, 200, processFunc)

	go queue.Run(ctx)

	// Simulate multiple goroutines adding data
	fmt.Println("Starting concurrent data insertion...")
	for g := 0; g < 3; g++ {
		go func(goroutineID int) {
			for i := 1; i <= 50; i++ {
				queue.Add(goroutineID*100+i, fmt.Sprintf("goroutine-%d-item-%d", goroutineID, i))
				time.Sleep(10 * time.Millisecond)
			}
		}(g)
	}

	// Wait for processing
	time.Sleep(2 * time.Second)
	cancel()
	time.Sleep(200 * time.Millisecond)
	fmt.Println("Done")
}
