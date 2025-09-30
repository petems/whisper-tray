package app

import (
	"context"
	"testing"
	"time"

	"github.com/petems/whisper-tray/internal/audio"
	"github.com/petems/whisper-tray/internal/config"
	"github.com/petems/whisper-tray/internal/whisper"
	"github.com/rs/zerolog"
)

// Mock implementations for testing
type mockCapture struct{}

func (m *mockCapture) Start(ctx context.Context, deviceID string, sampleRate int, out chan<- []float32) error {
	return nil
}

func (m *mockCapture) Stop() error {
	return nil
}

func (m *mockCapture) ListDevices() ([]audio.AudioDevice, error) {
	return []audio.AudioDevice{{ID: "default", Name: "Default", Default: true}}, nil
}

func (m *mockCapture) Close() error {
	return nil
}

type mockTranscriber struct{}

func (m *mockTranscriber) StartSession(opts whisper.SessionOpts) (whisper.Session, error) {
	return &mockSession{}, nil
}

func (m *mockTranscriber) LoadModel(model string) error {
	return nil
}

func (m *mockTranscriber) Close() error {
	return nil
}

type mockSession struct {
	partialsCh chan string
	finalsCh   chan string
}

func (m *mockSession) Feed(samples []float32) error {
	return nil
}

func (m *mockSession) Partials() <-chan string {
	if m.partialsCh == nil {
		m.partialsCh = make(chan string)
		close(m.partialsCh)
	}
	return m.partialsCh
}

func (m *mockSession) Finals() <-chan string {
	if m.finalsCh == nil {
		m.finalsCh = make(chan string)
		close(m.finalsCh)
	}
	return m.finalsCh
}

func (m *mockSession) Close() error {
	return nil
}

type mockInjector struct{}

func (m *mockInjector) Paste(ctx context.Context, text string) error {
	return nil
}

func (m *mockInjector) Type(ctx context.Context, text string) error {
	return nil
}

func (m *mockInjector) PasteOrType(ctx context.Context, text string) error {
	return nil
}

func TestToggleModeKeyPress(t *testing.T) {
	cfg := &config.Config{
		Mode: config.ModeToggle,
		Audio: config.AudioConfig{
			DeviceID: "default",
		},
		Whisper: config.WhisperConfig{
			Model:       "base.en",
			Language:    "auto",
			Temperature: 0.0,
			Threads:     0,
		},
	}

	app := New(Config{
		Audio:       &mockCapture{},
		Transcriber: &mockTranscriber{},
		Injector:    &mockInjector{},
		Config:      cfg,
		Logger:      zerolog.Nop(),
	})

	// Initially not dictating
	if app.IsDictating() {
		t.Error("App should not be dictating initially")
	}

	// First key press - should start dictating
	app.OnHotkey(true)
	if !app.IsDictating() {
		t.Error("App should be dictating after first key press")
	}

	// Key release - should NOT stop dictating in Toggle mode
	app.OnHotkey(false)
	if !app.IsDictating() {
		t.Error("App should still be dictating after key release in Toggle mode")
	}

	// Second key press - should stop dictating
	app.OnHotkey(true)

	// Wait for dictation to stop
	var stopped bool
	for i := 0; i < 100; i++ { // Poll for 1 second
		if !app.IsDictating() {
			stopped = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !stopped {
		t.Error("App should have stopped dictating after second key press")
	}
}

func TestPushToTalkModeKeyPress(t *testing.T) {
	cfg := &config.Config{
		Mode: config.ModePushToTalk,
		Audio: config.AudioConfig{
			DeviceID: "default",
		},
		Whisper: config.WhisperConfig{
			Model:       "base.en",
			Language:    "auto",
			Temperature: 0.0,
			Threads:     0,
		},
	}

	app := New(Config{
		Audio:       &mockCapture{},
		Transcriber: &mockTranscriber{},
		Injector:    &mockInjector{},
		Config:      cfg,
		Logger:      zerolog.Nop(),
	})

	// Initially not dictating
	if app.IsDictating() {
		t.Error("App should not be dictating initially")
	}

	// Key press - should start dictating
	app.OnHotkey(true)
	if !app.IsDictating() {
		t.Error("App should be dictating after key press")
	}

	// Key release - should stop dictating in PushToTalk mode
	app.OnHotkey(false)

	// Wait for dictation to stop
	var stopped bool
	for i := 0; i < 100; i++ { // Poll for 1 second
		if !app.IsDictating() {
			stopped = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !stopped {
		t.Error("App should have stopped dictating after key release")
	}
}

func TestToggleModeIgnoresKeyRelease(t *testing.T) {
	cfg := &config.Config{
		Mode: config.ModeToggle,
		Audio: config.AudioConfig{
			DeviceID: "default",
		},
		Whisper: config.WhisperConfig{
			Model:       "base.en",
			Language:    "auto",
			Temperature: 0.0,
			Threads:     0,
		},
	}

	app := New(Config{
		Audio:       &mockCapture{},
		Transcriber: &mockTranscriber{},
		Injector:    &mockInjector{},
		Config:      cfg,
		Logger:      zerolog.Nop(),
	})

	// Key release when not dictating - should do nothing
	app.OnHotkey(false)
	if app.IsDictating() {
		t.Error("App should not start dictating on key release")
	}

	// Start dictating with key press
	app.OnHotkey(true)
	if !app.IsDictating() {
		t.Error("App should be dictating after key press")
	}

	// Multiple key releases - should not stop dictating
	app.OnHotkey(false)
	app.OnHotkey(false)
	app.OnHotkey(false)
	if !app.IsDictating() {
		t.Error("App should still be dictating after multiple key releases in Toggle mode")
	}
}