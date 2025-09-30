package audio

import "context"

// Capture defines the interface for audio capture
type Capture interface {
	Start(ctx context.Context, deviceID string, sampleRate int, out chan<- []float32) error
	Stop() error
	ListDevices() ([]AudioDevice, error)
	Close() error
}

// AudioDevice represents an audio input device
type AudioDevice struct {
	ID      string
	Name    string
	Default bool
}