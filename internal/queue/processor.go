package queue

import (
	"context"
	"log"
	"sync"
	"time"

	"gin-sample/internal/models"
	"gin-sample/internal/transcription"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// MaxRetries is the maximum number of automatic retries for failed transcriptions.
	MaxRetries = 3
	// RetryDelay is the base delay between retries (exponential backoff).
	RetryDelay = 5 * time.Second
	// StatusUpdateTimeout is the timeout for status updates during error handling.
	StatusUpdateTimeout = 5 * time.Second
)

// TranscriptionUpdater defines the interface for updating transcription results.
type TranscriptionUpdater interface {
	UpdateTranscriptionAndStatus(ctx context.Context, id primitive.ObjectID, transcription string, status models.VoiceMemoStatus) error
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.VoiceMemoStatus) error
}

// Processor processes transcription jobs from the queue.
type Processor struct {
	queue        *MemoryQueue
	transcriber  transcription.Service
	updater      TranscriptionUpdater
	workerCount  int
	wg           sync.WaitGroup
	shutdownOnce sync.Once
	shutdownCh   chan struct{}
}

// NewProcessor creates a new transcription job processor.
func NewProcessor(queue *MemoryQueue, transcriber transcription.Service, updater TranscriptionUpdater, workerCount int) *Processor {
	return &Processor{
		queue:       queue,
		transcriber: transcriber,
		updater:     updater,
		workerCount: workerCount,
		shutdownCh:  make(chan struct{}),
	}
}

// Start begins processing jobs with the configured number of workers.
func (p *Processor) Start(ctx context.Context) {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
	log.Printf("Transcription processor started with %d workers", p.workerCount)
}

// Stop gracefully stops the processor, waiting for workers to finish.
func (p *Processor) Stop() {
	p.shutdownOnce.Do(func() {
		close(p.shutdownCh)
		p.queue.Close()
	})
	p.wg.Wait()
	log.Println("Transcription processor stopped")
}

func (p *Processor) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		job, err := p.queue.Dequeue(ctx)
		if err != nil {
			if err == ErrQueueClosed || err == context.Canceled {
				log.Printf("Worker %d shutting down", id)
				return
			}
			continue
		}
		p.processJob(ctx, job)
	}
}

func (p *Processor) processJob(ctx context.Context, job TranscriptionJob) {
	log.Printf("Processing transcription job for memo %s (attempt %d)", job.MemoID.Hex(), job.RetryCount+1)

	// Perform transcription
	text, err := p.transcriber.Transcribe(ctx, job.AudioFileKey)
	if err != nil {
		log.Printf("Transcription failed for memo %s: %v", job.MemoID.Hex(), err)
		p.handleFailure(ctx, job)
		return
	}

	// Update memo with transcription result
	err = p.updater.UpdateTranscriptionAndStatus(ctx, job.MemoID, text, models.StatusReady)
	if err != nil {
		log.Printf("Failed to update memo %s with transcription: %v", job.MemoID.Hex(), err)
		p.handleFailure(ctx, job)
		return
	}

	log.Printf("Transcription completed for memo %s", job.MemoID.Hex())
}

func (p *Processor) handleFailure(ctx context.Context, job TranscriptionJob) {
	job.RetryCount++

	if job.RetryCount >= MaxRetries {
		// Max retries reached, mark as failed
		log.Printf("Max retries reached for memo %s, marking as failed", job.MemoID.Hex())
		if err := p.updater.UpdateStatus(ctx, job.MemoID, models.StatusFailed); err != nil {
			log.Printf("Failed to update status to failed for memo %s: %v", job.MemoID.Hex(), err)
		}
		return
	}

	// Calculate exponential backoff delay
	delay := RetryDelay * time.Duration(1<<uint(job.RetryCount-1))
	log.Printf("Retrying memo %s in %v (attempt %d/%d)", job.MemoID.Hex(), delay, job.RetryCount+1, MaxRetries)

	// Schedule retry with delay. Uses shutdownCh instead of ctx to allow
	// in-flight retries to complete during graceful shutdown.
	go func() {
		select {
		case <-p.shutdownCh:
			// Shutdown initiated - mark as failed since we can't retry
			log.Printf("Shutdown during retry delay for memo %s, marking as failed", job.MemoID.Hex())
			updateCtx, cancel := context.WithTimeout(context.Background(), StatusUpdateTimeout)
			defer cancel()
			if updateErr := p.updater.UpdateStatus(updateCtx, job.MemoID, models.StatusFailed); updateErr != nil {
				log.Printf("Failed to update status to failed: %v", updateErr)
			}
			return
		case <-time.After(delay):
			if err := p.queue.Enqueue(job); err != nil {
				log.Printf("Failed to re-enqueue job for memo %s: %v", job.MemoID.Hex(), err)
				// Mark as failed if we can't re-enqueue
				updateCtx, cancel := context.WithTimeout(context.Background(), StatusUpdateTimeout)
				defer cancel()
				if updateErr := p.updater.UpdateStatus(updateCtx, job.MemoID, models.StatusFailed); updateErr != nil {
					log.Printf("Failed to update status to failed: %v", updateErr)
				}
			}
		}
	}()
}
