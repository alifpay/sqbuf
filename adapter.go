package sqbuf

import "context"

// NewLegacy - creates a queue compatible with old API (without context)
// This is a compatibility wrapper for existing code
func NewLegacy(dataRows uint32, interval int, save func(data [][]any)) *Queue {
	return New(dataRows, interval, func(ctx context.Context, data [][]any) {
		// Ignore context in legacy mode
		save(data)
	})
}

// NewAsyncLegacy - creates an async queue compatible with old API (without context)
func NewAsyncLegacy(dataRows uint32, interval int, save func(data [][]any)) *Queue {
	return NewAsync(dataRows, interval, func(ctx context.Context, data [][]any) {
		// Ignore context in legacy mode
		save(data)
	})
}
