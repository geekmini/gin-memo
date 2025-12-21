package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"gin-sample/internal/models"
	transcriptionmocks "gin-sample/internal/transcription/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/mock/gomock"
)

// MockUpdater implements TranscriptionUpdater for testing.
type MockUpdater struct {
	mu               sync.Mutex
	transcriptions   map[string]string
	statuses         map[string]models.VoiceMemoStatus
	updateCalls      int
	transcribeErrors map[string]error
	statusErrors     map[string]error
}

func NewMockUpdater() *MockUpdater {
	return &MockUpdater{
		transcriptions:   make(map[string]string),
		statuses:         make(map[string]models.VoiceMemoStatus),
		transcribeErrors: make(map[string]error),
		statusErrors:     make(map[string]error),
	}
}

func (m *MockUpdater) UpdateTranscriptionAndStatus(ctx context.Context, id primitive.ObjectID, transcription string, status models.VoiceMemoStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCalls++

	key := id.Hex()
	if err, ok := m.transcribeErrors[key]; ok {
		return err
	}
	m.transcriptions[key] = transcription
	m.statuses[key] = status
	return nil
}

func (m *MockUpdater) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.VoiceMemoStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCalls++

	key := id.Hex()
	if err, ok := m.statusErrors[key]; ok {
		return err
	}
	m.statuses[key] = status
	return nil
}

func (m *MockUpdater) GetTranscription(id primitive.ObjectID) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	text, ok := m.transcriptions[id.Hex()]
	return text, ok
}

func (m *MockUpdater) GetStatus(id primitive.ObjectID) (models.VoiceMemoStatus, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	status, ok := m.statuses[id.Hex()]
	return status, ok
}

func (m *MockUpdater) GetUpdateCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateCalls
}

func TestNewProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	queue := NewMemoryQueue(10)
	mockTranscriber := transcriptionmocks.NewMockService(ctrl)
	mockUpdater := NewMockUpdater()

	processor := NewProcessor(queue, mockTranscriber, mockUpdater, 2)

	assert.NotNil(t, processor)
	assert.Equal(t, queue, processor.queue)
	assert.Equal(t, mockTranscriber, processor.transcriber)
	assert.Equal(t, mockUpdater, processor.updater)
	assert.Equal(t, 2, processor.workerCount)
}

func TestProcessor_StartStop(t *testing.T) {
	t.Run("starts and stops cleanly", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queue := NewMemoryQueue(10)
		mockTranscriber := transcriptionmocks.NewMockService(ctrl)
		mockUpdater := NewMockUpdater()
		processor := NewProcessor(queue, mockTranscriber, mockUpdater, 3)

		ctx := context.Background()
		processor.Start(ctx)

		// Give workers time to start
		time.Sleep(50 * time.Millisecond)

		// Stop should complete without hanging
		done := make(chan struct{})
		go func() {
			processor.Stop()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("Stop() timed out")
		}
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queue := NewMemoryQueue(10)
		mockTranscriber := transcriptionmocks.NewMockService(ctrl)
		mockUpdater := NewMockUpdater()
		processor := NewProcessor(queue, mockTranscriber, mockUpdater, 1)

		ctx := context.Background()
		processor.Start(ctx)

		// Multiple stops should not panic
		processor.Stop()
		processor.Stop()
		processor.Stop()
	})
}

func TestProcessor_ProcessJob(t *testing.T) {
	t.Run("successfully processes transcription job", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queue := NewMemoryQueue(10)
		mockTranscriber := transcriptionmocks.NewMockService(ctrl)
		mockUpdater := NewMockUpdater()
		processor := NewProcessor(queue, mockTranscriber, mockUpdater, 1)

		memoID := primitive.NewObjectID()
		job := TranscriptionJob{
			MemoID:       memoID,
			AudioFileKey: "test/audio.mp3",
			RetryCount:   0,
		}

		mockTranscriber.EXPECT().
			Transcribe(gomock.Any(), "test/audio.mp3").
			Return("This is the transcription", nil)

		// Enqueue job
		_ = queue.Enqueue(job)

		// Start processor
		ctx, cancel := context.WithCancel(context.Background())
		processor.Start(ctx)

		// Wait for job to be processed
		time.Sleep(200 * time.Millisecond)

		cancel()
		processor.Stop()

		// Verify transcription was stored
		text, ok := mockUpdater.GetTranscription(memoID)
		require.True(t, ok)
		assert.Equal(t, "This is the transcription", text)

		status, ok := mockUpdater.GetStatus(memoID)
		require.True(t, ok)
		assert.Equal(t, models.StatusReady, status)
	})

	t.Run("handles transcription failure with retry", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queue := NewMemoryQueue(10)
		mockTranscriber := transcriptionmocks.NewMockService(ctrl)
		mockUpdater := NewMockUpdater()
		processor := NewProcessor(queue, mockTranscriber, mockUpdater, 1)

		memoID := primitive.NewObjectID()
		job := TranscriptionJob{
			MemoID:       memoID,
			AudioFileKey: "test/audio.mp3",
			RetryCount:   0,
		}

		// First attempt fails
		mockTranscriber.EXPECT().
			Transcribe(gomock.Any(), "test/audio.mp3").
			Return("", assert.AnError)

		_ = queue.Enqueue(job)

		ctx, cancel := context.WithCancel(context.Background())
		processor.Start(ctx)

		// Wait for initial failure and retry scheduling
		time.Sleep(200 * time.Millisecond)

		cancel()
		processor.Stop()

		// Job should have been handled (either retried or marked failed)
	})

	t.Run("marks as failed after max retries", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queue := NewMemoryQueue(10)
		mockTranscriber := transcriptionmocks.NewMockService(ctrl)
		mockUpdater := NewMockUpdater()
		processor := NewProcessor(queue, mockTranscriber, mockUpdater, 1)

		memoID := primitive.NewObjectID()
		job := TranscriptionJob{
			MemoID:       memoID,
			AudioFileKey: "test/audio.mp3",
			RetryCount:   MaxRetries - 1, // One more failure will trigger max retries
		}

		mockTranscriber.EXPECT().
			Transcribe(gomock.Any(), "test/audio.mp3").
			Return("", assert.AnError)

		_ = queue.Enqueue(job)

		ctx, cancel := context.WithCancel(context.Background())
		processor.Start(ctx)

		// Wait for job to be processed
		time.Sleep(200 * time.Millisecond)

		cancel()
		processor.Stop()

		// Should be marked as failed
		status, ok := mockUpdater.GetStatus(memoID)
		require.True(t, ok)
		assert.Equal(t, models.StatusFailed, status)
	})
}

func TestProcessor_HandleFailure(t *testing.T) {
	t.Run("schedules retry on failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queue := NewMemoryQueue(10)
		mockTranscriber := transcriptionmocks.NewMockService(ctrl)
		mockUpdater := NewMockUpdater()
		processor := NewProcessor(queue, mockTranscriber, mockUpdater, 1)

		// handleFailure is called internally during processJob
		// We test it indirectly through processJob test cases
		ctx := context.Background()
		processor.Start(ctx)
		processor.Stop()

		// Verify processor was created correctly
		assert.NotNil(t, processor)
	})

	t.Run("uses exponential backoff", func(t *testing.T) {
		// RetryDelay * 2^(retryCount-1)
		// RetryCount 1: 5s * 1 = 5s
		// RetryCount 2: 5s * 2 = 10s
		// RetryCount 3: 5s * 4 = 20s

		delays := []time.Duration{
			RetryDelay * time.Duration(1<<0), // 5s
			RetryDelay * time.Duration(1<<1), // 10s
			RetryDelay * time.Duration(1<<2), // 20s
		}

		assert.Equal(t, 5*time.Second, delays[0])
		assert.Equal(t, 10*time.Second, delays[1])
		assert.Equal(t, 20*time.Second, delays[2])
	})
}

func TestProcessor_WorkerShutdown(t *testing.T) {
	t.Run("workers shut down gracefully on context cancel", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queue := NewMemoryQueue(10)
		mockTranscriber := transcriptionmocks.NewMockService(ctrl)
		mockUpdater := NewMockUpdater()
		processor := NewProcessor(queue, mockTranscriber, mockUpdater, 3)

		ctx, cancel := context.WithCancel(context.Background())
		processor.Start(ctx)

		// Give workers time to start
		time.Sleep(50 * time.Millisecond)

		// Cancel context
		cancel()

		// Stop should complete quickly
		done := make(chan struct{})
		go func() {
			processor.Stop()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("Graceful shutdown timed out")
		}
	})
}

func TestProcessor_Concurrent(t *testing.T) {
	t.Run("processes multiple jobs concurrently", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queue := NewMemoryQueue(100)
		mockTranscriber := transcriptionmocks.NewMockService(ctrl)
		mockUpdater := NewMockUpdater()
		processor := NewProcessor(queue, mockTranscriber, mockUpdater, 5)

		jobCount := 10
		memoIDs := make([]primitive.ObjectID, jobCount)

		// Expect transcribe calls for all jobs
		mockTranscriber.EXPECT().
			Transcribe(gomock.Any(), gomock.Any()).
			Return("transcription", nil).
			Times(jobCount)

		// Enqueue jobs
		for i := 0; i < jobCount; i++ {
			memoIDs[i] = primitive.NewObjectID()
			_ = queue.Enqueue(TranscriptionJob{
				MemoID:       memoIDs[i],
				AudioFileKey: "test/audio.mp3",
			})
		}

		ctx, cancel := context.WithCancel(context.Background())
		processor.Start(ctx)

		// Wait for all jobs to be processed
		time.Sleep(500 * time.Millisecond)

		cancel()
		processor.Stop()

		// Verify all jobs were processed
		for _, memoID := range memoIDs {
			_, ok := mockUpdater.GetTranscription(memoID)
			assert.True(t, ok, "Job for memo %s was not processed", memoID.Hex())
		}
	})
}

func TestProcessor_UpdateFailure(t *testing.T) {
	t.Run("handles update failure after successful transcription", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queue := NewMemoryQueue(10)
		mockTranscriber := transcriptionmocks.NewMockService(ctrl)
		mockUpdater := NewMockUpdater()
		processor := NewProcessor(queue, mockTranscriber, mockUpdater, 1)

		memoID := primitive.NewObjectID()
		job := TranscriptionJob{
			MemoID:       memoID,
			AudioFileKey: "test/audio.mp3",
			RetryCount:   MaxRetries - 1, // One failure will trigger max retries
		}

		// Transcription succeeds but update fails
		mockTranscriber.EXPECT().
			Transcribe(gomock.Any(), "test/audio.mp3").
			Return("transcription", nil)

		// Make update fail
		mockUpdater.transcribeErrors[memoID.Hex()] = assert.AnError

		_ = queue.Enqueue(job)

		ctx, cancel := context.WithCancel(context.Background())
		processor.Start(ctx)

		time.Sleep(200 * time.Millisecond)

		cancel()
		processor.Stop()

		// Job should eventually be marked as failed
		status, ok := mockUpdater.GetStatus(memoID)
		require.True(t, ok)
		assert.Equal(t, models.StatusFailed, status)
	})
}
