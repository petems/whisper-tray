package tray

import (
	"testing"

	"github.com/petems/whisper-tray/internal/config"
)

// TestConfigModeField verifies that the Config struct's Mode field
// can be set to valid mode values. This tests the config data structure
// only, not the actual mode switching logic in the UI.
func TestConfigModeField(t *testing.T) {
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

// TestConfigModeFieldMutation verifies that the Config struct's Mode field
// can be mutated. This tests the config data structure only, not the actual
// mode switching logic in the UI which involves app.SetMode() and UI updates.
func TestConfigModeFieldMutation(t *testing.T) {
	cfg := &config.Config{
		Mode: "PushToTalk",
	}

	// Mutate field to Toggle
	cfg.Mode = "Toggle"
	if cfg.Mode != "Toggle" {
		t.Errorf("expected mode Toggle after field mutation, got %s", cfg.Mode)
	}

	// Mutate field back to PushToTalk
	cfg.Mode = "PushToTalk"
	if cfg.Mode != "PushToTalk" {
		t.Errorf("expected mode PushToTalk after field mutation, got %s", cfg.Mode)
	}
}