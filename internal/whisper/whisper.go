package whisper

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	whisper "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/rs/zerolog/log"

	"github.com/petems/whisper-tray/internal/config"
)

// Transcriber interface for speech-to-text
type Transcriber interface {
	StartSession(opts SessionOpts) (Session, error)
	LoadModel(model string) error
	Close() error
}

// Session represents an active transcription session
type Session interface {
	Feed(samples []float32) error
	Partials() <-chan string
	Finals() <-chan string
	Close() error
}

// SessionOpts configures a transcription session
type SessionOpts struct {
	Language    string
	Temperature float32
	Threads     int
	BeamSize    int
	NoContext   bool
}

type whisperTranscriber struct {
	model     whisper.Model
	modelPath string
	mu        sync.Mutex
}

// New creates a new Whisper transcriber
func New(cfg config.WhisperConfig) (Transcriber, error) {
	modelPath := filepath.Join(config.ModelsPath(), cfg.Model+".bin")

	// Check if model exists, download if needed
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		if err := downloadModel(cfg.Model, modelPath); err != nil {
			return nil, fmt.Errorf("failed to download model: %w", err)
		}
	}

	// Load model using official bindings
	model, err := whisper.New(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	return &whisperTranscriber{
		model:     model,
		modelPath: modelPath,
	}, nil
}

func (w *whisperTranscriber) StartSession(opts SessionOpts) (Session, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	session := &whisperSession{
		transcriber: w,
		opts:        opts,
		partials:    make(chan string, 10),
		finals:      make(chan string, 10),
		samples:     make([]float32, 0, 16000*30), // 30 second buffer
		done:        make(chan struct{}),
	}

	return session, nil
}

func (w *whisperTranscriber) LoadModel(model string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.model != nil {
		w.model.Close()
	}

	modelPath := filepath.Join(config.ModelsPath(), model+".bin")

	newModel, err := whisper.New(modelPath)
	if err != nil {
		return fmt.Errorf("failed to load model: %w", err)
	}

	w.model = newModel
	w.modelPath = modelPath
	return nil
}

func (w *whisperTranscriber) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.model != nil {
		w.model.Close()
		w.model = nil
	}
	return nil
}

// ===== SESSION =====

type whisperSession struct {
	transcriber *whisperTranscriber
	opts        SessionOpts

	mu         sync.Mutex
	samples    []float32
	partials   chan string
	finals     chan string
	done       chan struct{}
	processing bool
}

func (s *whisperSession) Feed(samples []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Append to buffer
	s.samples = append(s.samples, samples...)

	// Process when we have enough audio (1 second chunks)
	if len(s.samples) >= 16000 && !s.processing {
		go s.processChunk()
	}

	return nil
}

func (s *whisperSession) processChunk() error {
	s.mu.Lock()
	if s.processing {
		s.mu.Unlock()
		return nil
	}
	s.processing = true

	// Copy samples to process
	samplesToProcess := make([]float32, len(s.samples))
	copy(samplesToProcess, s.samples)

	// Clear buffer (don't keep context for now - simpler)
	s.samples = s.samples[:0]
	s.mu.Unlock()

	// Log processing start
	duration := float64(len(samplesToProcess)) / 16000.0
	log.Debug().
		Int("samples", len(samplesToProcess)).
		Float64("duration_sec", duration).
		Msg("Processing audio chunk")

	start := time.Now()

	// Process audio with whisper
	model := s.transcriber.model

	// Create context
	context, err := model.NewContext()
	if err != nil {
		s.mu.Lock()
		s.processing = false
		s.mu.Unlock()
		return fmt.Errorf("failed to create context: %w", err)
	}

	// Set parameters
	if s.opts.Threads > 0 {
		context.SetThreads(uint(s.opts.Threads))
	}
	if s.opts.Language != "auto" && s.opts.Language != "" {
		context.SetLanguage(s.opts.Language)
	}
	context.SetTranslate(false)

	// Process the audio
	if err := context.Process(samplesToProcess, nil, nil); err != nil {
		s.mu.Lock()
		s.processing = false
		s.mu.Unlock()
		return fmt.Errorf("whisper process failed: %w", err)
	}

	processTime := time.Since(start)
	log.Debug().
		Dur("process_time", processTime).
		Float64("realtime_factor", processTime.Seconds()/duration).
		Msg("Whisper processing complete")

	// Get transcription segments
	segmentCount := 0
	for {
		segment, err := context.NextSegment()
		if err != nil {
			break // EOF or error
		}
		text := segment.Text
		segmentCount++

		log.Debug().
			Str("text", text).
			Int("segment", segmentCount).
			Msg("Got transcription segment")

		// Send as final (blocking send to ensure delivery)
		s.finals <- text
	}

	log.Debug().
		Int("segments", segmentCount).
		Msg("Finished processing chunk")

	s.mu.Lock()
	s.processing = false
	s.mu.Unlock()

	return nil
}

func (s *whisperSession) Partials() <-chan string {
	return s.partials
}

func (s *whisperSession) Finals() <-chan string {
	return s.finals
}

func (s *whisperSession) Close() error {
	log.Debug().Msg("Closing whisper session")

	// Signal done first
	close(s.done)

	// Wait for any ongoing processing to finish
	s.mu.Lock()
	waitCount := 0
	for s.processing {
		s.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		waitCount++
		if waitCount%100 == 0 {
			log.Debug().Int("wait_ms", waitCount*10).Msg("Still waiting for processing to complete")
		}
		s.mu.Lock()
	}

	hasRemaining := len(s.samples) > 0
	remainingSamples := len(s.samples)
	s.mu.Unlock()

	// Process any remaining samples if we have them
	if hasRemaining {
		log.Debug().Int("samples", remainingSamples).Msg("Processing remaining samples")
		s.processChunk()
	}

	// Wait again to ensure final processChunk completes
	s.mu.Lock()
	waitCount = 0
	for s.processing {
		s.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		waitCount++
		if waitCount%100 == 0 {
			log.Debug().Int("wait_ms", waitCount*10).Msg("Still waiting for final processing to complete")
		}
		s.mu.Lock()
	}
	s.mu.Unlock()

	log.Debug().Msg("Closing channels")

	// Now safe to close channels
	close(s.partials)
	close(s.finals)

	log.Debug().Msg("Session closed")
	return nil
}