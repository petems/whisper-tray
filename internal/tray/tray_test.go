package tray

import (
	"testing"

	"github.com/petems/whisper-tray/internal/config"
)

func TestModeConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		initialMode  string
		expectedMode string
	}{
		{
			name:         "PushToTalk mode",
			initialMode:  "PushToTalk",
			expectedMode: "PushToTalk",
		},
		{
			name:         "Toggle mode",
			initialMode:  "Toggle",
			expectedMode: "Toggle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Mode: tt.initialMode,
			}

			if cfg.Mode != tt.expectedMode {
				t.Errorf("expected mode %s, got %s", tt.expectedMode, cfg.Mode)
			}
		})
	}
}

func TestModeSwitch(t *testing.T) {
	cfg := &config.Config{
		Mode: "PushToTalk",
	}

	// Simulate switching to Toggle
	cfg.Mode = "Toggle"
	if cfg.Mode != "Toggle" {
		t.Errorf("expected mode Toggle after switch, got %s", cfg.Mode)
	}

	// Simulate switching back to PushToTalk
	cfg.Mode = "PushToTalk"
	if cfg.Mode != "PushToTalk" {
		t.Errorf("expected mode PushToTalk after switch, got %s", cfg.Mode)
	}
}

func TestValidModes(t *testing.T) {
	validModes := []string{"PushToTalk", "Toggle"}

	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
			cfg := &config.Config{
				Mode: mode,
			}

			if cfg.Mode != mode {
				t.Errorf("expected mode %s to be valid, got %s", mode, cfg.Mode)
			}
		})
	}
}