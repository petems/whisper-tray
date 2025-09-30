package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	Hotkey         string        `json:"hotkey"`
	HotkeyDarwin   string        `json:"hotkey_darwin"`
	Mode           string        `json:"mode"` // "PushToTalk" or "Toggle"
	Audio          AudioConfig   `json:"audio"`
	Whisper        WhisperConfig `json:"whisper"`
	Inject         InjectConfig  `json:"inject"`
	AppendSpace    bool          `json:"append_space"`
	StreamPartials bool          `json:"stream_partials"`
	EnterOnFinal   bool          `json:"enter_on_final"`
	RunAtLogin     bool          `json:"run_at_login"`
}

type AudioConfig struct {
	DeviceID string `json:"device_id"`
}

type WhisperConfig struct {
	Model       string  `json:"model"`        // "base.en", "small", etc.
	Language    string  `json:"language"`     // "auto", "en", etc.
	Temperature float32 `json:"temperature"`
	Threads     int     `json:"threads"`
	GPU         string  `json:"gpu"`          // "auto", "cpu", "cuda", "metal"
}

type InjectConfig struct {
	PreferPaste bool `json:"prefer_paste"`
}

// Load reads the config from disk or returns defaults
func Load() (*Config, error) {
	path := configPath()

	// Default config
	cfg := &Config{
		Hotkey:       "Alt+Space",
		HotkeyDarwin: "Alt+Space", // Option+Space
		Mode:         "PushToTalk",
		Audio: AudioConfig{
			DeviceID: "",
		},
		Whisper: WhisperConfig{
			Model:       "base.en",
			Language:    "auto",
			Temperature: 0.0,
			Threads:     0, // Auto-detect
			GPU:         "auto",
		},
		Inject: InjectConfig{
			PreferPaste: true,
		},
		AppendSpace:    true,
		StreamPartials: false,
		EnterOnFinal:   false,
		RunAtLogin:     false,
	}

	// Load existing config if it exists
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// Save writes the config to disk
func (c *Config) Save() error {
	path := configPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// PlatformHotkey returns the appropriate hotkey for the current platform
func (c *Config) PlatformHotkey() string {
	if runtime.GOOS == "darwin" && c.HotkeyDarwin != "" {
		return c.HotkeyDarwin
	}
	return c.Hotkey
}

// configPath returns the platform-specific config file path
func configPath() string {
	var base string

	switch runtime.GOOS {
	case "darwin":
		base = os.Getenv("HOME") + "/Library/Application Support"
	case "windows":
		base = os.Getenv("APPDATA")
	default: // linux
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			base = xdg
		} else {
			base = os.Getenv("HOME") + "/.config"
		}
	}

	return filepath.Join(base, "whisper-tray", "config.json")
}

// ModelsPath returns the platform-specific models directory path
func ModelsPath() string {
	var base string

	switch runtime.GOOS {
	case "darwin":
		base = os.Getenv("HOME") + "/Library/Application Support"
	case "windows":
		base = os.Getenv("LOCALAPPDATA")
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			base = xdg
		} else {
			base = os.Getenv("HOME") + "/.local/share"
		}
	}

	return filepath.Join(base, "whisper-tray", "models")
}