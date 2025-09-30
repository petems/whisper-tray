# WhisperTray

Local, privacy-focused voice dictation for macOS (with cross-platform support planned).

## Status

✅ **MVP Complete** - Full macOS functionality with Whisper.cpp integration!

## Features

- ✅ **Global hotkey** (Control+Space) - Push-to-talk or toggle mode
- ✅ **System tray integration** with emoji status indicators (🎤 🟢/🔴/🟡/⚪️)
- ✅ **Audio capture** via PortAudio with device selection
- ✅ **Text injection** - Both clipboard-paste (Cmd+V) and keyboard typing
- ✅ **Whisper.cpp integration** with Metal acceleration on macOS
- ✅ **Model auto-download** with progress tracking
- ✅ **Multiple models** - base.en, small.en, medium.en, large-v3, large-v3-turbo
- ✅ **Settings persistence** - All configuration saved automatically
- ✅ **Structured logging** - Detailed logs with zerolog

## Quick Start (macOS)

### Prerequisites

```bash
# Install PortAudio via Homebrew
brew install portaudio

# Ensure you have Xcode Command Line Tools
xcode-select --install
```

### Build

```bash
# Install Go dependencies
make install-deps

# Build everything (downloads whisper.cpp, compiles, and builds binary)
make all

# Or quick dev build (if whisper.cpp already set up)
make dev
```

### Run

```bash
# Run the binary
./bin/whisper-tray
```

**Important**: You'll need to grant the app:
1. **Microphone permission** - macOS will prompt automatically
2. **Accessibility permission** - Required for global hotkeys
   - Go to: System Settings → Privacy & Security → Accessibility
   - Add whisper-tray to the allowed apps

## Usage

1. **Start the app** - Look for 🎤 🟢 in your menu bar
2. **Press Control+Space** to start dictation (icon changes to 🔴)
3. **Speak your text**
4. **Release Control+Space** - Icon changes to 🟡 while processing
5. **Text appears** in your focused application (icon returns to 🟢)

### Tray Menu Options

- **Mode** - Switch between Push-to-Talk and Toggle
- **Microphone** - Select audio input device
- **Model** - Choose Whisper model (shows downloaded status)
- **Prefer Paste** - Use clipboard (Cmd+V) or keyboard typing
- **Run at Login** - Auto-start with macOS

### Configuration

Settings are saved to `~/Library/Application Support/whisper-tray/config.json`

Logs are written to `~/Library/Logs/whisper-tray/whisper-tray.log`

## Current Limitations

- **macOS only** - Linux/Windows implementations exist but need testing
- **First model download** - ~141MB for base.en, ~465MB for small.en
- **No streaming** - Transcription happens after you finish speaking
- **Metal shader** - Requires `ggml-metal.metal` file in working directory

## Architecture

```
whisper-tray/
├── cmd/whisper-tray/         # Entry point
├── internal/
│   ├── app/                  # Application orchestrator
│   ├── audio/                # PortAudio capture
│   ├── config/               # Configuration
│   ├── hotkey/               # Global hotkeys (macOS/Linux/Windows)
│   ├── inject/               # Text injection
│   ├── logging/              # Structured logging
│   ├── permissions/          # Permission handling (macOS)
│   ├── tray/                 # System tray UI
│   └── whisper/              # Whisper.cpp integration
├── resources/                # macOS app bundle resources
└── scripts/                  # Installation scripts
```

## Development

See [CLAUDE.md](CLAUDE.md) for detailed architecture and implementation notes.

## License

MIT (to be added)