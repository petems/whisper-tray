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

	return zerolog.New(multi).With().Timestamp().Caller().Logger()
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