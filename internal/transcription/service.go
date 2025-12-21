// Package transcription provides audio transcription functionality.
package transcription

import (
	"context"
	"time"
)

// Service defines the interface for audio transcription.
type Service interface {
	// Transcribe converts audio to text. Returns the transcription text or error.
	Transcribe(ctx context.Context, audioKey string) (string, error)
}

// MockService is a mock implementation of Service for development/testing.
type MockService struct {
	// SimulatedDelay is the time to simulate transcription processing.
	SimulatedDelay time.Duration
	// FailureRate is the probability of failure (0.0 to 1.0) for testing retry logic.
	FailureRate float64
}

// NewMockService creates a new MockService with default settings.
func NewMockService() *MockService {
	return &MockService{
		SimulatedDelay: 2 * time.Second,
		FailureRate:    0.0, // No failures by default
	}
}

// Transcribe simulates audio transcription.
func (s *MockService) Transcribe(ctx context.Context, audioKey string) (string, error) {
	// Simulate processing time
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(s.SimulatedDelay):
	}

	// Return mock transcription
	return "This is a mock transcription of the audio file. " +
		"In production, this would contain the actual transcribed text from the audio.", nil
}
