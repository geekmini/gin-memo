// Package queue provides job queue functionality for background processing.
package queue

import (
	"context"
	"sync"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TranscriptionJob represents a job to transcribe a voice memo.
type TranscriptionJob struct {
	MemoID       primitive.ObjectID
	AudioFileKey string
	RetryCount   int
}

// MemoryQueue is an in-memory job queue for transcription jobs.
type MemoryQueue struct {
	jobs     chan TranscriptionJob
	capacity int
	mu       sync.RWMutex
	closed   bool
}

// NewMemoryQueue creates a new in-memory queue with the given capacity.
func NewMemoryQueue(capacity int) *MemoryQueue {
	return &MemoryQueue{
		jobs:     make(chan TranscriptionJob, capacity),
		capacity: capacity,
	}
}

// Enqueue adds a job to the queue. Returns error if queue is full or closed.
// Lock is held during the entire operation to prevent race condition with Close().
func (q *MemoryQueue) Enqueue(job TranscriptionJob) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return ErrQueueClosed
	}

	select {
	case q.jobs <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

// Dequeue returns the next job from the queue, blocking until one is available.
// Returns error if context is cancelled or queue is closed.
func (q *MemoryQueue) Dequeue(ctx context.Context) (TranscriptionJob, error) {
	select {
	case <-ctx.Done():
		return TranscriptionJob{}, ctx.Err()
	case job, ok := <-q.jobs:
		if !ok {
			return TranscriptionJob{}, ErrQueueClosed
		}
		return job, nil
	}
}

// Close closes the queue. No more jobs can be enqueued after closing.
func (q *MemoryQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.closed {
		q.closed = true
		close(q.jobs)
	}
}

// Reset resets the queue to a fresh state. This is primarily for testing.
func (q *MemoryQueue) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = false
	q.jobs = make(chan TranscriptionJob, q.capacity)
}

// Len returns the current number of jobs in the queue.
func (q *MemoryQueue) Len() int {
	return len(q.jobs)
}

// Capacity returns the queue capacity.
func (q *MemoryQueue) Capacity() int {
	return q.capacity
}
