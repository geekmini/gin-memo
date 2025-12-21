package testutil

import (
	"context"
	"time"
)

// TestContext creates a context with timeout for tests.
func TestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// TestContextWithTimeout creates a context with custom timeout.
func TestContextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
