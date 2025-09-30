# WhisperMatic - Development Context

## Project Overview
WhisperMatic is a cross-platform local voice dictation tool with system tray integration. The goal is to provide privacy-focused, offline speech-to-text functionality triggered by hotkeys.

## Current MVP Focus
Currently focused on getting the MVP working with:
- Basic audio capture
- Whisper integration for transcription
- Hotkey support (push-to-talk primary mode)
- Text injection via clipboard-paste (primary) with keystroke fallback
- System tray presence

## High-Level Goals (from spec)
- Global hotkey to start/stop "dictation to focused app"
- Low-latency **local** transcription (whisper.cpp), no cloud required
- Cross-platform: **Windows, macOS, Linux (X11/Wayland)**
- **Tray icon** (getlantern/systray) with quick controls
- Robust text injection (**clipboard-paste primary**, keystroke fallback)
- Minimal friction: sensible defaults, works out of the box

## Target Architecture (Clean, Testable)
```
/cmd/whisper-tray/          # main
/internal/
  app/                      # app lifecycle, DI, supervisors
  audio/                    # device discovery & capture abstraction
    coreaudio_darwin.go
    wasapi_windows.go
    pulse_linux.go
    portaudio_all.go        # optional single backend via PortAudio
  hotkey/                   # global hotkey abstraction
    darwin.go (CGEvent tap)
    windows.go (RegisterHotKey)
    x11.go / wayland.go
  inject/                   # text injection abstraction
    paste.go                # clipboard + Ctrl/⌘+V strategy
    typekeys.go             # keystroke fallback (SendInput/CGEvent/XTest)
  vad/                      # optional voice activity detection (push-to-talk ok)
  whisper/                  # binding to whisper.cpp, model mgmt, streaming
  tray/                     # systray UI
  config/                   # load/save (XDG/Library/AppData), schema/validation
  logging/                  # zerolog setup
  metrics/                  # optional local (expvar); opt-in telemetry
  updater/                  # model download + checksum
/pkg/
  events/                   # pubsub channels (start, stop, partial, final)
  ui/                       # notifications/toasts (platform helpers)
```

## Data Flow (MVP)
Hotkey → start capture → audio ring buffer → whisper streaming session → partial/final text → inject (clipboard+paste) → focused app.

Use Go channels between stages with bounded buffers to apply backpressure:
- `audio.Chan <- []float32`
- `whisper.Partials <- string`
- `whisper.Finals <- string`

## Key Component Decisions

### 1. Whisper Engine
- Use **whisper.cpp** via C API (thin cgo layer or existing binding)
- Default model `base.en` (fast). Allow `small`, `medium`, `large-v3` later
- Runtime flags via config/tray: `language`, `temperature`, `noContext`, `beam_size`, `n_threads`, `gpu` (CUDA/Metal/OpenCL/CPU)
- Streaming: emit partials every ~300–500ms; finalize on VAD stop or push-to-talk release

### 2. Audio Capture
- Simplest cross-platform: **PortAudio** Go binding for MVP
- Alternative native backends (later) for latency:
  - macOS: CoreAudio/AVAudioEngine
  - Windows: WASAPI
  - Linux: PulseAudio/PipeWire
- 16 kHz mono float32 pipeline; small ring buffer (e.g., 2 × 512 samples)

### 3. Global Hotkeys
- Windows: `RegisterHotKey` (default **Alt+Space**; handle conflicts gracefully)
- macOS: CGEvent tap (default **Option+Space**); requires Accessibility permission
- Linux:
  - X11: `XGrabKey`
  - Wayland: prefer desktop portal shortcut or document manual binding; provide X11 fallback if available

### 4. Text Injection (to focused window)
- **Primary:** "clipboard swap + paste"
  1. Save clipboard (text only for MVP)
  2. Set transcript
  3. Send paste shortcut (Ctrl+V / ⌘+V)
  4. Restore clipboard (only if unchanged by user meanwhile)
- **Fallback:** Keystroke injection
  - Windows: `SendInput`
  - macOS: CGEvent keyboard events (Accessibility permission)
  - Linux: XTest (X11) / portals if available
- Options: "append space", "press Enter", "stream-paste partials (experimental)"

### 5. Tray UI (getlantern/systray)
Menu items:
- **Start/Stop Dictation** (shows current hotkey)
- **Mode**: Push-to-talk / Toggle
- **Microphone** (submenu of devices)
- **Model** (base.en/small/…)
- **Language** (Auto/en/…)
- **Punctuation/Smart Caps** (on/off)
- **Paste vs Type** (prefer paste)
- **Run at login**
- **About / Logs / Quit**

### 6. Config & State
- Use XDG/Library/AppData paths; JSON/TOML (e.g., `spf13/viper`)
- Persist: hotkey, device id, model, language, options, "run at login"
- Model cache dir with SHA256 checks & resumable downloads

### 7. Permissions
- macOS: microphone + Accessibility; show guided prompts if missing
- Windows: microphone privacy setting
- Linux: PipeWire portal permissions if using portals; otherwise none

### 8. Performance & UX
- Default CPU threads = `runtime.NumCPU()`; expose setting
- Debounce partials; default paste only on **final** (streaming paste optional)
- Pre/post text hooks: leading space if needed; auto-capitalize sentence starts

### 9. Reliability
- Supervisor goroutines with context cancellation; recover panics; surface errors to tray notifications
- Bounded channels + drop oldest on backpressure (prefer responsiveness over completeness)

### 10. Packaging
- Windows: `winget`/MSIX or NSIS; "Run at startup" registry toggle
- macOS: `.app` + (later) notarization; LaunchAgent for login item
- Linux: AppImage/Flatpak (Flatpak helps with Wayland portals), or .deb/.rpm

## Core Interfaces (Design Contracts)

```go
// audio
type Capture interface {
  Start(ctx context.Context, deviceID string, sampleRate int, out chan<- []float32) error
  ListDevices() ([]AudioDevice, error)
}

type AudioDevice struct { ID, Name string; Default bool }

// whisper
type Transcriber interface {
  StartSession(opts SessionOpts) (Session, error)
}

type Session interface {
  Feed(samples []float32) error
  Partials() <-chan string
  Finals() <-chan string
  Close() error
}

// hotkey
type HotkeyManager interface {
  Register(accel string, cb func(pressed bool)) error // pressed==true for keydown
  Unregister(accel string) error
}

// inject
type Injector interface {
  Paste(ctx context.Context, s string) error
  Type(ctx context.Context, s string) error
  PasteOrType(ctx context.Context, s string) error
}
```

## Core Orchestrator Pattern

```go
type Mode int
const (
  PushToTalk Mode = iota
  Toggle
)

type App struct {
  audio Capture
  stt   Transcriber
  inj   Injector
  hk    HotkeyManager
  cfg   *Config
  log   zerolog.Logger
}

func (a *App) OnHotkey(press bool) {
  switch a.cfg.Mode {
  case PushToTalk:
    if press { a.startDictation() } else { a.stopAndInject() }
  case Toggle:
    if !a.dictating { a.startDictation() } else { a.stopAndInject() }
  }
}

func (a *App) startDictation() { /* start capture + stt session, wire channels */ }
func (a *App) stopAndInject()  { /* flush finals, join, filters, inject PasteOrType */ }
```

## Config Schema

```toml
hotkey = "Alt+Space"         # win/linux
hotkey_darwin = "Option+Space"
mode = "PushToTalk"          # or "Toggle"
device_id = ""
model = "base.en"
language = "auto"
paste_preferred = true
stream_partials = false
append_space = true
enter_on_final = false
threads = 0                  # 0 => NumCPU
gpu = "auto"                 # auto|cpu|cuda|metal|opencl
```

## MVP User Stories (Acceptance Criteria)

1. **Dictate to cursor**
   - When app is running, pressing hotkey records and, on release (PTT) or stop (toggle), it **pastes** transcript into the focused field (Notepad/TextEdit/Chrome verified)

2. **Model auto-download**
   - If model isn't present, prompt once; download; verify integrity; continue automatically

3. **Device selection**
   - Change mic from tray; persists across restarts

4. **Permissions helpers**
   - Clear prompts + link to enable; retry button

5. **Resilient clipboard**
   - Clipboard restored even on failure; never clobbers new user content

## Key Dependencies

- `github.com/getlantern/systray` - System tray
- `github.com/gordonklaus/portaudio` - Audio (or native backends behind interface)
- `github.com/atotto/clipboard` - Clipboard handling
- Platform-specific hotkey wrappers; optional `github.com/go-vgo/robotgo` as fallback only
- `github.com/rs/zerolog` - Logging
- `github.com/spf13/viper` - Config
- (Optional VAD) simple energy threshold; later `webrtcvad` bindings

## Cross-Platform Implementation Details (from cross_platform_context_spec.md)

### Linux Hotkey (`internal/hotkey/hotkey_linux.go`)
**Build Tag**: `//go:build linux`

**CGO Configuration**:
```go
#cgo pkg-config: x11 xtst
#include <X11/Xlib.h>
#include <X11/keysym.h>
#include <X11/extensions/XTest.h>
```

**Implementation Details**:
- X11-based key grabbing with XTest extension
- Uses CGO bindings to X11/Xlib
- Global Display pointer (`displayPtr`) for X11 connection
- `grabKey(keycode, modifiers)`: Registers global hotkey on root window
  - Uses `XGrabKey` with GrabModeAsync
  - Selects KeyPressMask | KeyReleaseMask events
- `checkEvent(keycode, pressed)`: Polls for keyboard events
  - Uses `XPending` and `XNextEvent`
  - Returns keycode and press/release state
- Event loop: 10ms ticker polling for events
- Callback system: Maps keycodes to `func(bool)` callbacks
- Default: Alt+Space (keycode 65, Mod1Mask = 8)

**Interface Implementation**:
```go
type linuxManager struct {
  callbacks map[int]func(bool)
  stop      chan struct{}
}

func (m *linuxManager) Register(accel string, callback func(pressed bool)) error
func (m *linuxManager) Unregister(accel string) error
func (m *linuxManager) Close() error
```

**Notes**:
- TODO: Actual accelerator string parsing (currently hardcoded)
- TODO: `XUngrabKey` in Unregister
- Graceful shutdown via stop channel

### Logging (`internal/logging/logging.go`)
**Features**:
- Structured logging with `zerolog`
- Multi-writer output: console (stderr with colors) + file
- Console writer with RFC3339 timestamps
- File logging with automatic directory creation

**Platform-Specific Log Paths**:
```go
switch runtime.GOOS {
case "darwin":
  base = os.Getenv("HOME") + "/Library/Logs"
case "windows":
  base = os.Getenv("LOCALAPPDATA")
default: // Linux
  if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
    base = xdg
  } else {
    base = os.Getenv("HOME") + "/.local/state"
  }
}
// Final path: {base}/whisper-tray/whisper-tray.log
```

**Usage**:
```go
logger := logging.New()
logger.Info().Msg("Starting application")
logger.Error().Err(err).Msg("Failed to initialize")
```

### macOS Permissions (`internal/permissions/permissions_darwin.go`)
**Build Tag**: `//go:build darwin`

**CGO Configuration**:
```go
#cgo LDFLAGS: -framework AVFoundation -framework Cocoa
#import <AVFoundation/AVFoundation.h>
#import <Cocoa/Cocoa.h>
```

**Permission States**:
```go
const (
  PermissionNotDetermined = 0  // User hasn't been asked yet
  PermissionRestricted    = 1  // Device management restricts access
  PermissionDenied        = 2  // User explicitly denied
  PermissionAuthorized    = 3  // User granted permission
)
```

**Microphone Permission**:
- `CheckMicrophone()`: Returns AVAuthorizationStatus for audio
- `RequestMicrophone()`: Triggers system permission dialog
- Uses AVFoundation's `AVCaptureDevice` authorization

**Accessibility Permission** (required for hotkeys):
- `CheckAccessibility()`: Uses `AXIsProcessTrustedWithOptions`
- Automatically prompts user with `kAXTrustedCheckOptionPrompt`
- Returns boolean (granted/not granted)

**Helper Function**:
```go
func EnsurePermissions() error {
  // Checks both microphone and accessibility
  // Returns error if either not granted
  // Prints user-friendly guidance messages
  // Directs to: System Settings → Privacy & Security → Accessibility
}
```

**Integration**:
- Call `EnsurePermissions()` at app startup
- Handle errors by showing retry UI or guidance
- Permissions persist after first grant

### macOS App Bundle (`resources/Info.plist`)
```xml
<key>CFBundleExecutable</key>
<string>whisper-tray</string>

<key>CFBundleIdentifier</key>
<string>com.whispertray.app</string>

<key>CFBundleName</key>
<string>WhisperTray</string>

<key>LSMinimumSystemVersion</key>
<string>10.15</string>

<key>LSUIElement</key>
<true/>  <!-- Hides from Dock, tray/menubar only -->

<key>NSMicrophoneUsageDescription</key>
<string>WhisperTray needs microphone access to transcribe your voice.</string>
```

**Key Properties**:
- `LSUIElement: true` - Runs as menubar-only app (no Dock icon)
- `NSMicrophoneUsageDescription` - Required for microphone permission prompt
- Minimum macOS 10.15 (Catalina)
- Bundle version 1.0.0

### Linux Installation (`scripts/install-linux.sh`)
**Features**:
- Multi-distro package manager detection
- Automatic dependency installation
- System-wide binary installation
- XDG desktop entry creation

**Distro Support**:
```bash
# Debian/Ubuntu (apt-get)
sudo apt-get install -y portaudio19-dev libx11-dev libxtst-dev

# Fedora/RHEL (dnf)
sudo dnf install -y portaudio-devel libX11-devel libXtst-devel

# Arch Linux (pacman)
sudo pacman -S --noconfirm portaudio libx11 libxtst
```

**Installation Steps**:
1. Detect package manager (apt-get, dnf, or pacman)
2. Install system dependencies
3. Copy binary to `/usr/local/bin/whisper-tray`
4. Set executable permissions
5. Create desktop entry at `~/.local/share/applications/whisper-tray.desktop`

**Desktop Entry**:
```ini
[Desktop Entry]
Type=Application
Name=WhisperTray
Comment=Local voice dictation
Exec=/usr/local/bin/whisper-tray
Icon=whisper-tray
Terminal=false
Categories=Utility;
```

### Test Infrastructure (`internal/app/app_test.go`)
**Mock Audio Capture**:
```go
type mockCapture struct{}

func (m *mockCapture) Start(ctx context.Context, deviceID string, sampleRate int, out chan<- []float32) error {
  // Generates 10 chunks of 512 silent samples at 32ms intervals
  // Respects context cancellation
}
```

**Test Pattern**:
```go
func TestDictationFlow(t *testing.T) {
  // Tests full cycle: hotkey → start → audio → whisper → inject
  cfg := &config.Config{Mode: "PushToTalk"}
  app := New(Config{Audio: &mockCapture{}, Config: cfg})
  // Verify app initialization
}
```

**Testing Strategy**:
- Mock all external interfaces (audio, whisper, inject, hotkey)
- Test state transitions (idle → recording → transcribing → injecting)
- Verify channel wiring and data flow
- Test error handling and recovery

## Testing Strategy

- **Unit**: model manager (checksums), clipboard atomicity, text filters, config load/save
- **Integration**: fake audio (WAV replay) → assert transcript contains expected phrase
- **E2E (manual)**: matrix across 3 OS + apps (Notepad/TextEdit/VSCode/Chrome)

### Running Tests

The Makefile provides flexible test commands:

```bash
# Run all tests (auto-detects platform)
make test

# Run specific test package
make test TEST=./internal/audio

# Platform-specific tests
make test-osx          # macOS with Metal/Accelerate frameworks
make test-linux        # Linux with X11
make test-windows      # Windows

# Run specific test on specific platform
make test-osx TEST=./internal/hotkey
```

**Implementation Details**:
- `make test` auto-detects OS (Darwin/Linux/Windows) and delegates to platform-specific target
- Each platform target uses appropriate CGO flags for native dependencies
- Supports `TEST` variable to run specific packages or tests
- CI uses `make test` and `make build` in GitHub Actions

## Security/Privacy Principles

- Default offline; no network except model download
- No telemetry unless explicit opt-in
- Best-effort safeguard: avoid typing in password fields (skip when window title/class matches common password prompts; provide manual "safety lock")

## Best Practices & Gotchas

- **Keep UI thread light**: systray callbacks hand off to goroutines
- **Bound everything**: small-capacity channels to prevent runaway memory when CPU is pegged
- **Graceful shutdown**: contexts; stop audio → whisper → inject; drain channels
- **Cross-platform keys**: normalize accelerators (`Alt`/`Option`/`Cmd`) at config load
- **Wayland reality**: document limitations; suggest Flatpak + portals; X11 fallback if present
- **Logs**: rotating file in app data dir; "Open Logs" menu item
- **Feature flags**: gate experimental VAD/streaming paste behind config toggles

## Development Phases

### Phase 1: MVP ✅ COMPLETE
- [x] Basic audio capture working (PortAudio)
- [x] Whisper transcription functional (whisper.cpp with Metal)
- [x] Simple hotkey support (macOS Control+Space)
- [x] Text injection via clipboard-paste AND keyboard typing
- [x] System tray icon with emoji status indicators
- [x] Config loading/saving (JSON persistence)
- [x] Model auto-download with progress tracking
- [x] Multiple model support (base.en through large-v3-turbo)
- [x] Device selection UI in tray menu
- [x] Push-to-talk AND toggle modes
- [x] Settings UI in tray with visual feedback
- [x] Structured logging with zerolog

### Phase 2: Cross-Platform Core (In Progress)
- [x] macOS hotkey handler (Control+Space with Carbon)
- [x] macOS permissions system (Microphone + Accessibility)
- [x] Unified logging infrastructure (platform-specific paths)
- [ ] Linux hotkey handler (X11 implementation exists, needs testing)
- [ ] Windows hotkey handler (needs implementation)
- [ ] Linux audio/inject testing
- [ ] Windows audio/inject testing
- [x] GitHub Actions CI (macOS working, Linux/Windows experimental)

### Phase 3: Polish & Distribution
- [ ] Application bundling (macOS .app, Windows installer, Linux packages)
- [ ] Auto-updater
- [ ] Run at login implementation (UI exists, needs platform code)
- [ ] Icon design and assets
- [ ] Multiple language support
- [ ] Release automation

### Phase 4: Advanced Features (Post-MVP)
- [ ] Voice activity detection (VAD)
- [ ] Live typing mode (streaming partials)
- [ ] Command mode ("period", "new line", custom macros)
- [ ] Custom vocabulary
- [ ] Recording history/playback
- [ ] Auto language detection
- [ ] Noise suppression (RNNoise) pre-filter
- [ ] Model SHA256 verification
- [ ] Resumable downloads

## Build Considerations

- **CGO Dependencies**:
  - macOS: Xcode Command Line Tools
  - Linux: gcc, X11 dev packages, PortAudio dev packages
  - Windows: MinGW-w64 or MSVC
- **Cross-Compilation**: Requires platform-specific toolchains for CGO code
- **Testing**: Mock interfaces for platform-specific components

## Technology Stack

- **Language**: Go 1.22+
- **Audio**: PortAudio (cross-platform MVP) → native backends later
- **Transcription**: whisper.cpp bindings
- **Logging**: zerolog
- **Config**: viper
- **Tray**: getlantern/systray
- **Clipboard**: atotto/clipboard
- **Build Tags**: Platform-specific code with `//go:build` directives

## Licensing

- MIT for app code
- Acknowledge whisper.cpp (MIT) & dependencies

## References

- Whisper: https://github.com/openai/whisper
- whisper.cpp: https://github.com/ggerganov/whisper.cpp
- PortAudio: http://www.portaudio.com/
- Platform APIs:
  - macOS: AVFoundation, Accessibility, Carbon/Cocoa
  - Linux: X11, PulseAudio/PipeWire
  - Windows: Win32 API, WASAPI