# Open-Source Cross-Platform Golang Based Whisper Transcrption App 

---

## High-level goals
- Global hotkey to start/stop “dictation to focused app.”
- Low-latency **local** transcription (whisper.cpp), no cloud required.
- Cross-platform: **Windows, macOS, Linux (X11/Wayland)**.
- **Tray icon** (getlantern/systray) with quick controls.
- Robust text injection (**clipboard-paste primary**, keystroke fallback).
- Minimal friction: sensible defaults, works out of the box.

---

## Architecture (clean, testable)
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

### Data flow (MVP)
Hotkey → start capture → audio ring buffer → whisper streaming session → partial/final text → inject (clipboard+paste) → focused app.

Use Go channels between stages with bounded buffers to apply backpressure:
- `audio.Chan <- []float32`
- `whisper.Partials <- string`
- `whisper.Finals <- string`

---

## Key components & choices

### 1) Whisper engine
- Use **whisper.cpp** via C API (thin cgo layer or an existing binding).
- Default model `base.en` (fast). Allow `small`, `medium`, `large-v3` later.
- Runtime flags via config/tray: `language`, `temperature`, `noContext`, `beam_size`, `n_threads`, `gpu` (CUDA/Metal/OpenCL/CPU).
- Streaming: emit partials every ~300–500ms; finalize on VAD stop or push-to-talk release.

### 2) Audio capture
- Simplest cross-platform: **PortAudio** Go binding for MVP.
- Alternative native backends (later) for latency:
  - macOS: CoreAudio/AVAudioEngine.
  - Windows: WASAPI.
  - Linux: PulseAudio/PipeWire.
- 16 kHz mono float32 pipeline; small ring buffer (e.g., 2 × 512 samples).

### 3) Global hotkeys
- Windows: `RegisterHotKey` (default **Alt+Space**; handle conflicts gracefully).
- macOS: CGEvent tap (default **Option+Space**); requires Accessibility permission.
- Linux:
  - X11: `XGrabKey`.
  - Wayland: prefer desktop portal shortcut or document manual binding; provide X11 fallback if available.

### 4) Text injection (to focused window)
- **Primary:** “clipboard swap + paste”
  1) Save clipboard (text only for MVP).
  2) Set transcript.
  3) Send paste shortcut (Ctrl+V / ⌘+V).
  4) Restore clipboard (only if unchanged by user meanwhile).
- **Fallback:** Keystroke injection
  - Windows: `SendInput`.
  - macOS: CGEvent keyboard events (Accessibility permission).
  - Linux: XTest (X11) / portals if available.
- Options: “append space”, “press Enter”, “stream-paste partials (experimental)”.

### 5) Tray UI (getlantern/systray)
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

### 6) Config & state
- Use XDG/Library/AppData paths; JSON/TOML (e.g., `spf13/viper`).
- Persist: hotkey, device id, model, language, options, “run at login”.
- Model cache dir with SHA256 checks & resumable downloads.

### 7) Permissions
- macOS: microphone + Accessibility; show guided prompts if missing.
- Windows: microphone privacy setting.
- Linux: PipeWire portal permissions if using portals; otherwise none.

### 8) Performance & UX
- Default CPU threads = `runtime.NumCPU()`; expose setting.
- Debounce partials; default paste only on **final** (streaming paste optional).
- Pre/post text hooks: leading space if needed; auto-capitalize sentence starts.

### 9) Reliability
- Supervisor goroutines with context cancellation; recover panics; surface errors to tray notifications.
- Bounded channels + drop oldest on backpressure (prefer responsiveness over completeness).

### 10) Packaging
- Windows: `winget`/MSIX or NSIS; “Run at startup” registry toggle.
- macOS: `.app` + (later) notarization; LaunchAgent for login item.
- Linux: AppImage/Flatpak (Flatpak helps with Wayland portals), or .deb/.rpm.

---

## MVP User stories (acceptance criteria)
1) **Dictate to cursor**
   - When app is running, pressing hotkey records and, on release (PTT) or stop (toggle), it **pastes** transcript into the focused field (Notepad/TextEdit/Chrome verified).
2) **Model auto-download**
   - If model isn’t present, prompt once; download; verify integrity; continue automatically.
3) **Device selection**
   - Change mic from tray; persists across restarts.
4) **Permissions helpers**
   - Clear prompts + link to enable; retry button.
5) **Resilient clipboard**
   - Clipboard restored even on failure; never clobbers new user content.

---

## Initial agent spec (task breakdown)
Use this as the “source of truth” for an agentic builder.

### Repository bootstrapping
- Init Go module `github.com/<you>/whisper-tray`.
- Set Go 1.22+. `Makefile` with `build-{windows,mac,linux}`, `lint`, `test`.
- CI (GitHub Actions): build 3 OS targets; (optional) cache whisper models.

### Dependencies
- `github.com/getlantern/systray`
- Audio: `github.com/gordonklaus/portaudio` (or native backends behind `audio` interface)
- Clipboard: `github.com/atotto/clipboard`
- Hotkeys: platform-specific thin wrappers; optional `github.com/go-vgo/robotgo` as fallback only
- Logging: `github.com/rs/zerolog`
- Config: `github.com/spf13/viper`
- (Optional VAD) simple energy threshold; later `webrtcvad` bindings

### Interfaces (design contracts)
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
  Register(accel string, cb func(pressed bool)) error // pressed==true for keydown; map to toggle if configured
  Unregister(accel string) error
}

// inject
type Injector interface {
  Paste(ctx context.Context, s string) error
  Type(ctx context.Context, s string) error
  PasteOrType(ctx context.Context, s string) error
}
```

### Core orchestrator
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

### Tray menus (systray)
- Build menu on start; keep references to items; event loop handles clicks.
- Reflect state (disable “Stop” when idle); rebuild device/model submenus on change.

### Model manager
- On model select, ensure file exists (download if needed).
- Keep manifest with checksums; verify SHA256; show progress.

### Clipboard-safe paste
- Save current clipboard value (text for MVP).
- Set UTF-8 text; send paste chord (Ctrl+V/⌘+V).
- Restore clipboard after short delay **only if** unchanged by user.

### Config schema
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

### Testing plan
- **Unit**: model manager (checksums), clipboard atomicity, text filters, config load/save.
- **Integration**: fake audio (WAV replay) → assert transcript contains expected phrase.
- **E2E (manual)**: matrix across 3 OS + apps (Notepad/TextEdit/VSCode/Chrome).

### Security/Privacy
- Default offline; no network except model download.
- No telemetry unless explicit opt-in.
- Best-effort safeguard: avoid typing in password fields (skip when window title/class matches common password prompts; provide manual “safety lock”).

### Licensing
- MIT for your app; acknowledge whisper.cpp (MIT) & deps.

---

## Minimal MVP pseudocode (wiring)
```go
func main() {
  log := logging.New()
  cfg := config.LoadOrDefault()
  go tray.Run(cfg) // menus -> channels

  hk  := hotkey.New()
  inj := inject.New()
  cap := audio.New()     // PortAudio init
  stt := whisper.New(cfg)

  app := app.New(cap, stt, inj, hk, cfg, log)

  hk.Register(cfg.PlatformHotkey(), app.OnHotkey)

  // react to tray changes
  for {
    select {
    case ev := <-tray.Events():
      app.HandleTrayEvent(ev)
    }
  }
}
```

---

## Best practices & gotchas
- **Keep UI thread light**: systray callbacks hand off to goroutines.
- **Bound everything**: small-capacity channels to prevent runaway memory when CPU is pegged.
- **Graceful shutdown**: contexts; stop audio → whisper → inject; drain channels.
- **Cross-platform keys**: normalize accelerators (`Alt`/`Option`/`Cmd`) at config load.
- **Wayland reality**: document limitations; suggest Flatpak + portals; X11 fallback if present.
- **Logs**: rotating file in app data dir; “Open Logs” menu item.
- **Feature flags**: gate experimental VAD/streaming paste behind config toggles.

---

## Stretch goals (post-MVP)
- **Live typing mode**: partial hypotheses typed as you speak.
- **Command mode**: “period”, “new line”, custom macros.
- **Auto language detection** and on-the-fly model switch.
- **Noise suppression (RNNoise)** pre-filter.
- **Optional cloud fallback** (only if user opts in).

---

## Ready-to-feed “Agent Spec” (concise)
**Title:** Build cross-platform Whisper-type Go app with tray and hotkey

**Objectives:**
1. Global hotkey → local whisper.cpp transcription → paste into focused window.
2. Tray UI (getlantern/systray) for start/stop, device/model selection, options.
3. Windows/macOS/Linux support (noting Wayland caveats).
4. Reproducible builds for all platforms.

**Deliverables:**
- Repo with structure above.
- Working binaries for win/mac/linux.
- Config + model manager.
- Manual test plan & README (permissions notes).
- CI pipeline building per-OS artifacts.

**Non-goals (MVP):**
- Perfect Wayland hotkeys/injection; streaming paste; advanced VAD.

**Constraints:**
- Local transcription only (whisper.cpp). No external APIs.
- Preserve user clipboard; restore safely.
- Telemetry only with explicit opt-in.

**Acceptance tests:**
- On each OS, default hotkey records and pastes into Notepad/TextEdit.
- Changing mic/model via tray works and persists.
- Missing model auto-downloads, verifies, and proceeds.
- Clipboard unchanged after dictation.
- macOS prompts guide user through Accessibility + Microphone permissions.
