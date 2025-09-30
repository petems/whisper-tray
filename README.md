# WhisperTray

Local, privacy-focused voice dictation for macOS (with cross-platform support planned).

## Status

✅ **MVP Complete** - Full macOS functionality with Whisper.cpp integration!

## Features (MVP)

- ✅ macOS hotkey support (Option+Space)
- ✅ System tray integration
- ✅ Audio capture via PortAudio
- ✅ Clipboard-paste text injection (Cmd+V)
- ✅ Cross-platform architecture
- ✅ Whisper.cpp integration with Metal acceleration
- ✅ Model auto-download

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

1. Press **Option+Space** to start dictation
2. Speak your text
3. Release **Option+Space** to stop and paste
4. Text appears in your focused application

## Current Limitations

- Hotkey is hardcoded to Option+Space (configurable in code)
- First run requires model download (~75MB for base.en)
- macOS only for now (Linux/Windows implementations exist but untested)
- No streaming transcription yet (processes on release)

## Next Steps

1. **Test with real audio**: Try speaking and see transcription!
2. **Add more languages**: Configure in `~/.config/whisper-tray/config.json`
3. **Try different models**: `small.en` for better accuracy, `large-v3` for best quality
4. **Customize hotkey**: Edit `internal/hotkey/hotkey_darwin.go` (line 79-80)

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