package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/petems/whisper-tray/internal/audio"
	"github.com/petems/whisper-tray/internal/config"
	"github.com/petems/whisper-tray/internal/hotkey"
	"github.com/petems/whisper-tray/internal/inject"
	"github.com/petems/whisper-tray/internal/whisper"
	"github.com/rs/zerolog"
)

type Mode int

const (
	PushToTalk Mode = iota
	Toggle
)

// StatusUpdater is an interface for updating status (e.g., tray icon)
type StatusUpdater interface {
	SetIdle()
	SetRecording()
	SetProcessing()
	SetError()
}

type Config struct {
	Audio         audio.Capture
	Transcriber   whisper.Transcriber
	Injector      inject.Injector
	Hotkeys       hotkey.Manager
	Config        *config.Config
	Logger        zerolog.Logger
	StatusUpdater StatusUpdater // Optional - can be nil
}

type App struct {
	audio  audio.Capture
	stt    whisper.Transcriber
	inj    inject.Injector
	cfg    *config.Config
	log    zerolog.Logger
	status StatusUpdater

	mu         sync.Mutex
	dictating  bool
	session    whisper.Session
	audioCtx   context.Context
	audioStop  context.CancelFunc
	textBuffer []string
}

func New(cfg Config) *App {
	return &App{
		audio:  cfg.Audio,
		stt:    cfg.Transcriber,
		inj:    cfg.Injector,
		cfg:    cfg.Config,
		log:    cfg.Logger,
		status: cfg.StatusUpdater,
	}
}

func (a *App) OnHotkey(pressed bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	mode := PushToTalk
	if a.cfg.Mode == "Toggle" {
		mode = Toggle
	}

	switch mode {
	case PushToTalk:
		if pressed {
			a.startDictationLocked()
		} else {
			a.stopAndInjectLocked()
		}
	case Toggle:
		if !a.dictating {
			a.startDictationLocked()
		} else {
			a.stopAndInjectLocked()
		}
	}
}

func (a *App) startDictationLocked() {
	if a.dictating {
		return
	}

	a.log.Info().Msg("Starting dictation")
	a.dictating = true
	a.textBuffer = nil

	// Update status to recording
	if a.status != nil {
		a.status.SetRecording()
	}

	a.audioCtx, a.audioStop = context.WithCancel(context.Background())

	// Start whisper session
	session, err := a.stt.StartSession(whisper.SessionOpts{
		Language:    a.cfg.Whisper.Language,
		Temperature: a.cfg.Whisper.Temperature,
		Threads:     a.cfg.Whisper.Threads,
	})
	if err != nil {
		a.log.Error().Err(err).Msg("Failed to start session")
		a.dictating = false
		a.audioStop()
		return
	}
	a.session = session

	// Bounded audio buffer
	audioChan := make(chan []float32, 8)

	// Start audio capture
	go func() {
		if err := a.audio.Start(a.audioCtx, a.cfg.Audio.DeviceID, 16000, audioChan); err != nil {
			a.log.Error().Err(err).Msg("Audio error")
		}
	}()

	// Feed whisper
	go func() {
		for {
			select {
			case <-a.audioCtx.Done():
				return
			case samples, ok := <-audioChan:
				if !ok {
					return
				}
				if err := session.Feed(samples); err != nil {
					a.log.Error().Err(err).Msg("Feed error")
				}
			}
		}
	}()

	// Collect results
	go a.collectTranscripts()
}

func (a *App) stopAndInjectLocked() {
	if !a.dictating {
		return
	}

	a.log.Info().Msg("Stopping dictation")
	a.dictating = false

	// Update status to processing
	if a.status != nil {
		a.status.SetProcessing()
	}

	if a.audioStop != nil {
		a.audioStop()
	}

	if a.session != nil {
		time.Sleep(100 * time.Millisecond) // Allow finals
		a.session.Close()
	}

	text := a.joinText()
	if text == "" {
		a.log.Info().Msg("No text to inject")
		return
	}

	text = a.applyFilters(text)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.inj.PasteOrType(ctx, text); err != nil {
		a.log.Error().Err(err).Msg("Inject error")
		if a.status != nil {
			a.status.SetError()
		}
	} else {
		a.log.Info().Str("text", text).Msg("Injected")
		if a.status != nil {
			a.status.SetIdle()
		}
	}
}

func (a *App) collectTranscripts() {
	for {
		select {
		case <-a.audioCtx.Done():
			return
		case partial := <-a.session.Partials():
			if a.cfg.StreamPartials {
				a.log.Debug().Str("partial", partial).Msg("Partial")
			}
		case final, ok := <-a.session.Finals():
			if !ok {
				return
			}
			a.mu.Lock()
			a.textBuffer = append(a.textBuffer, final)
			a.mu.Unlock()
			a.log.Info().Str("final", final).Msg("Final")
		}
	}
}

func (a *App) joinText() string {
	result := strings.Join(a.textBuffer, " ")
	return strings.TrimSpace(result)
}

func (a *App) applyFilters(text string) string {
	if len(text) == 0 {
		return text
	}

	// Auto-capitalize first letter
	if text[0] >= 'a' && text[0] <= 'z' {
		text = string(text[0]-32) + text[1:]
	}

	if a.cfg.AppendSpace {
		text += " "
	}

	return text
}

func (a *App) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.dictating {
		a.stopAndInjectLocked()
	}

	return nil
}

// Tray actions

func (a *App) SetMode(mode string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.Mode = mode
	a.cfg.Save()
}

func (a *App) SetDevice(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.dictating {
		return fmt.Errorf("cannot change while dictating")
	}

	a.cfg.Audio.DeviceID = id
	return a.cfg.Save()
}

func (a *App) SetModel(model string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.dictating {
		return fmt.Errorf("cannot change while dictating")
	}

	a.cfg.Whisper.Model = model
	return a.cfg.Save()
}

func (a *App) IsDictating() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.dictating
}

func (a *App) ListDevices() ([]audio.AudioDevice, error) {
	return a.audio.ListDevices()
}