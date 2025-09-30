package logging

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// New creates a new zerolog logger with console and file output
func New() zerolog.Logger {
	return NewWithLevel("info")
}

// NewWithLevel creates a logger with a specific log level
func NewWithLevel(level string) zerolog.Logger {
	logPath := getLogPath()

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(logPath), 0755)

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open log file")
	}

	// Multi-writer: console + file
	multi := zerolog.MultiLevelWriter(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
		logFile,
	)

	// Parse log level
	logLevel := zerolog.InfoLevel
	switch level {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	}

	return zerolog.New(multi).With().Timestamp().Caller().Logger().Level(logLevel)
}

// getLogPath returns platform-specific log file path
func getLogPath() string {
	var base string

	switch runtime.GOOS {
	case "darwin":
		base = os.Getenv("HOME") + "/Library/Logs"
	case "windows":
		base = os.Getenv("LOCALAPPDATA")
	default:
		if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
			base = xdg
		} else {
			base = os.Getenv("HOME") + "/.local/state"
		}
	}

	return filepath.Join(base, "whisper-tray", "whisper-tray.log")
}