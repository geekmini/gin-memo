package queue

import "errors"

var (
	// ErrQueueFull is returned when the queue is at capacity.
	ErrQueueFull = errors.New("queue is full")
	// ErrQueueClosed is returned when trying to use a closed queue.
	ErrQueueClosed = errors.New("queue is closed")
)
