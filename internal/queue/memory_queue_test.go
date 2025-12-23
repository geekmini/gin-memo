package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewMemoryQueue(t *testing.T) {
	t.Run("creates queue with specified capacity", func(t *testing.T) {
		q := NewMemoryQueue(10)

		assert.NotNil(t, q)
		assert.Equal(t, 10, q.Capacity())
		assert.Equal(t, 0, q.Len())
	})

	t.Run("creates queue with zero capacity", func(t *testing.T) {
		q := NewMemoryQueue(0)

		assert.NotNil(t, q)
		assert.Equal(t, 0, q.Capacity())
	})
}

func TestMemoryQueue_Enqueue(t *testing.T) {
	t.Run("successfully enqueues job", func(t *testing.T) {
		q := NewMemoryQueue(10)
		job := TranscriptionJob{
			MemoID:       primitive.NewObjectID(),
			AudioFileKey: "test/audio.mp3",
			RetryCount:   0,
		}

		err := q.Enqueue(job)

		assert.NoError(t, err)
		assert.Equal(t, 1, q.Len())
	})

	t.Run("enqueues multiple jobs up to capacity", func(t *testing.T) {
		q := NewMemoryQueue(3)

		for i := 0; i < 3; i++ {
			err := q.Enqueue(TranscriptionJob{
				MemoID:       primitive.NewObjectID(),
				AudioFileKey: "test/audio.mp3",
			})
			assert.NoError(t, err)
		}

		assert.Equal(t, 3, q.Len())
	})

	t.Run("returns error when queue is full", func(t *testing.T) {
		q := NewMemoryQueue(2)

		// Fill the queue
		_ = q.Enqueue(TranscriptionJob{MemoID: primitive.NewObjectID()})
		_ = q.Enqueue(TranscriptionJob{MemoID: primitive.NewObjectID()})

		// Try to enqueue when full
		err := q.Enqueue(TranscriptionJob{MemoID: primitive.NewObjectID()})

		assert.Equal(t, ErrQueueFull, err)
		assert.Equal(t, 2, q.Len())
	})

	t.Run("returns error when queue is closed", func(t *testing.T) {
		q := NewMemoryQueue(10)
		q.Close()

		err := q.Enqueue(TranscriptionJob{MemoID: primitive.NewObjectID()})

		assert.Equal(t, ErrQueueClosed, err)
	})
}

func TestMemoryQueue_Dequeue(t *testing.T) {
	t.Run("successfully dequeues job", func(t *testing.T) {
		q := NewMemoryQueue(10)
		expectedJob := TranscriptionJob{
			MemoID:       primitive.NewObjectID(),
			AudioFileKey: "test/audio.mp3",
			RetryCount:   1,
		}
		_ = q.Enqueue(expectedJob)

		ctx := context.Background()
		job, err := q.Dequeue(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedJob.MemoID, job.MemoID)
		assert.Equal(t, expectedJob.AudioFileKey, job.AudioFileKey)
		assert.Equal(t, expectedJob.RetryCount, job.RetryCount)
		assert.Equal(t, 0, q.Len())
	})

	t.Run("dequeues in FIFO order", func(t *testing.T) {
		q := NewMemoryQueue(10)
		job1 := TranscriptionJob{MemoID: primitive.NewObjectID(), AudioFileKey: "first"}
		job2 := TranscriptionJob{MemoID: primitive.NewObjectID(), AudioFileKey: "second"}
		_ = q.Enqueue(job1)
		_ = q.Enqueue(job2)

		ctx := context.Background()
		result1, _ := q.Dequeue(ctx)
		result2, _ := q.Dequeue(ctx)

		assert.Equal(t, "first", result1.AudioFileKey)
		assert.Equal(t, "second", result2.AudioFileKey)
	})

	t.Run("returns error when context is cancelled", func(t *testing.T) {
		q := NewMemoryQueue(10)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := q.Dequeue(ctx)

		assert.Equal(t, context.Canceled, err)
	})

	t.Run("returns error when context deadline exceeded", func(t *testing.T) {
		q := NewMemoryQueue(10)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := q.Dequeue(ctx)

		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("returns error when queue is closed while waiting", func(t *testing.T) {
		q := NewMemoryQueue(10)

		// Close queue in background after short delay
		go func() {
			time.Sleep(50 * time.Millisecond)
			q.Close()
		}()

		ctx := context.Background()
		_, err := q.Dequeue(ctx)

		assert.Equal(t, ErrQueueClosed, err)
	})

	t.Run("blocks until job available", func(t *testing.T) {
		q := NewMemoryQueue(10)
		expectedJob := TranscriptionJob{MemoID: primitive.NewObjectID()}

		// Enqueue in background after short delay
		go func() {
			time.Sleep(50 * time.Millisecond)
			_ = q.Enqueue(expectedJob)
		}()

		ctx := context.Background()
		job, err := q.Dequeue(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedJob.MemoID, job.MemoID)
	})
}

func TestMemoryQueue_Close(t *testing.T) {
	t.Run("closes the queue", func(t *testing.T) {
		q := NewMemoryQueue(10)

		q.Close()

		// Verify closed by trying to enqueue
		err := q.Enqueue(TranscriptionJob{})
		assert.Equal(t, ErrQueueClosed, err)
	})

	t.Run("close is idempotent", func(t *testing.T) {
		q := NewMemoryQueue(10)

		// Should not panic when called multiple times
		q.Close()
		q.Close()
		q.Close()

		err := q.Enqueue(TranscriptionJob{})
		assert.Equal(t, ErrQueueClosed, err)
	})

	t.Run("allows draining existing jobs after close", func(t *testing.T) {
		q := NewMemoryQueue(10)
		job := TranscriptionJob{MemoID: primitive.NewObjectID()}
		_ = q.Enqueue(job)

		q.Close()

		// Should still be able to dequeue existing jobs
		ctx := context.Background()
		result, err := q.Dequeue(ctx)

		require.NoError(t, err)
		assert.Equal(t, job.MemoID, result.MemoID)

		// Next dequeue should return closed error
		_, err = q.Dequeue(ctx)
		assert.Equal(t, ErrQueueClosed, err)
	})
}

func TestMemoryQueue_Reset(t *testing.T) {
	t.Run("resets closed queue to usable state", func(t *testing.T) {
		q := NewMemoryQueue(10)

		// Close the queue
		q.Close()
		err := q.Enqueue(TranscriptionJob{MemoID: primitive.NewObjectID()})
		assert.Equal(t, ErrQueueClosed, err)

		// Reset the queue
		q.Reset()

		// Should be able to enqueue again
		err = q.Enqueue(TranscriptionJob{MemoID: primitive.NewObjectID()})
		assert.NoError(t, err)
		assert.Equal(t, 1, q.Len())
	})

	t.Run("clears existing jobs", func(t *testing.T) {
		q := NewMemoryQueue(10)

		// Add some jobs
		_ = q.Enqueue(TranscriptionJob{MemoID: primitive.NewObjectID()})
		_ = q.Enqueue(TranscriptionJob{MemoID: primitive.NewObjectID()})
		assert.Equal(t, 2, q.Len())

		// Reset clears all jobs
		q.Reset()

		assert.Equal(t, 0, q.Len())
	})

	t.Run("preserves capacity after reset", func(t *testing.T) {
		q := NewMemoryQueue(5)

		q.Reset()

		assert.Equal(t, 5, q.Capacity())
	})
}

func TestMemoryQueue_Len(t *testing.T) {
	t.Run("returns correct length", func(t *testing.T) {
		q := NewMemoryQueue(10)

		assert.Equal(t, 0, q.Len())

		_ = q.Enqueue(TranscriptionJob{MemoID: primitive.NewObjectID()})
		assert.Equal(t, 1, q.Len())

		_ = q.Enqueue(TranscriptionJob{MemoID: primitive.NewObjectID()})
		assert.Equal(t, 2, q.Len())

		ctx := context.Background()
		_, _ = q.Dequeue(ctx)
		assert.Equal(t, 1, q.Len())
	})
}

func TestMemoryQueue_Capacity(t *testing.T) {
	t.Run("returns configured capacity", func(t *testing.T) {
		testCases := []int{1, 10, 100, 1000}

		for _, capacity := range testCases {
			q := NewMemoryQueue(capacity)
			assert.Equal(t, capacity, q.Capacity())
		}
	})
}

func TestMemoryQueue_Concurrency(t *testing.T) {
	t.Run("handles concurrent enqueue and dequeue", func(t *testing.T) {
		q := NewMemoryQueue(100)
		ctx := context.Background()
		jobCount := 50

		// Start consumers
		results := make(chan TranscriptionJob, jobCount)
		for i := 0; i < 5; i++ {
			go func() {
				for {
					job, err := q.Dequeue(ctx)
					if err != nil {
						return
					}
					results <- job
				}
			}()
		}

		// Enqueue jobs concurrently
		for i := 0; i < jobCount; i++ {
			go func(id int) {
				_ = q.Enqueue(TranscriptionJob{
					MemoID:       primitive.NewObjectID(),
					AudioFileKey: "test",
				})
			}(i)
		}

		// Wait for all jobs to be processed
		receivedCount := 0
		timeout := time.After(2 * time.Second)
		for receivedCount < jobCount {
			select {
			case <-results:
				receivedCount++
			case <-timeout:
				t.Fatalf("Timed out waiting for jobs, received %d/%d", receivedCount, jobCount)
			}
		}

		q.Close()
		assert.Equal(t, jobCount, receivedCount)
	})
}
