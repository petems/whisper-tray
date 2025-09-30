package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/petems/whisper-tray/internal/app"
	"github.com/petems/whisper-tray/internal/audio"
	"github.com/petems/whisper-tray/internal/config"
	"github.com/petems/whisper-tray/internal/hotkey"
	"github.com/petems/whisper-tray/internal/inject"
	"github.com/petems/whisper-tray/internal/logging"
	"github.com/petems/whisper-tray/internal/permissions"
	"github.com/petems/whisper-tray/internal/tray"
	"github.com/petems/whisper-tray/internal/whisper"
)

var (
	// Version is set via ldflags at build time
	Version = "dev"
	// Commit is set via ldflags at build time
	Commit = "unknown"
)

func main() {
	// Load config from XDG/Library/AppData
	cfg, err := config.Load()
	if err != nil {
		// Use default logger if config fails to load
		log := logging.New()
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	// Initialize logger with configured level
	log := logging.NewWithLevel(cfg.LogLevel)

	// macOS requires explicit microphone + accessibility approval before capture or hotkeys work
	if err := permissions.EnsurePermissions(); err != nil {
		log.Fatal().Err(err).Msg("Required permissions not granted")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize audio capture
	capture, err := audio.New(cfg.Audio)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize audio")
	}
	defer capture.Close()

	// Initialize whisper
	transcriber, err := whisper.New(cfg.Whisper)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize whisper")
	}
	defer transcriber.Close()

	// Initialize text injector
	injector := inject.New(cfg.Inject)

	// Initialize hotkey manager
	hkManager, err := hotkey.New()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize hotkeys")
	}
	defer hkManager.Close()

	// Create tray UI first (we'll pass it to app)
	trayUI := tray.New(nil, cfg, Version, Commit) // App reference set below

	// Create app with tray as status updater
	application := app.New(app.Config{
		Audio:         capture,
		Transcriber:   transcriber,
		Injector:      injector,
		Hotkeys:       hkManager,
		Config:        cfg,
		Logger:        log,
		StatusUpdater: trayUI,
	})

	// Set app reference in tray
	trayUI.SetApp(application)

	// Register global hotkey
	if err := hkManager.Register(cfg.PlatformHotkey(), application.OnHotkey); err != nil {
		log.Fatal().Err(err).Msg("Failed to register hotkey")
	}

	log.Info().Msg("WhisperTray starting...")

	// Setup shutdown signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("Shutting down...")
		if err := application.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Shutdown error")
		}
		os.Exit(0)
	}()

	// Start tray UI - MUST run on main thread
	if err := trayUI.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("Tray error")
	}
}
