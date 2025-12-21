package queue

import "context"

//go:generate mockgen -destination=mocks/mock_queue.go -package=mocks gin-sample/internal/queue Queue

// Queue defines the interface for job queue operations.
type Queue interface {
	// Enqueue adds a job to the queue.
	Enqueue(job TranscriptionJob) error
	// Dequeue removes and returns the next job from the queue.
	Dequeue(ctx context.Context) (TranscriptionJob, error)
	// Close closes the queue.
	Close()
	// Len returns the current number of jobs in the queue.
	Len() int
	// Capacity returns the queue capacity.
	Capacity() int
}

// Ensure MemoryQueue implements Queue interface
var _ Queue = (*MemoryQueue)(nil)
