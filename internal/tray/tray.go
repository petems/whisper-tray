package tray

import (
	"context"
	"fmt"

	"github.com/petems/whisper-tray/internal/app"
	"github.com/petems/whisper-tray/internal/config"
	"github.com/petems/whisper-tray/internal/logging"
	"github.com/getlantern/systray"
	"github.com/rs/zerolog"
)

type UI struct {
	app     *app.App
	cfg     *config.Config
	version string
	commit  string
	log     zerolog.Logger

	// Menu items
	mStartStop   *systray.MenuItem
	mMode        *systray.MenuItem
	mDevices     *systray.MenuItem
	mModels      *systray.MenuItem
	mPastePrefer *systray.MenuItem
	mRunAtLogin  *systray.MenuItem
}

// Status update methods for the app to call
func (u *UI) SetIdle() {
	u.updateStatus("idle")
}

func (u *UI) SetRecording() {
	u.updateStatus("recording")
}

func (u *UI) SetProcessing() {
	u.updateStatus("processing")
}

func (u *UI) SetError() {
	u.updateStatus("error")
}

func New(application *app.App, cfg *config.Config, version, commit string) *UI {
	log := logging.New()
	return &UI{
		app:     application,
		cfg:     cfg,
		version: version,
		commit:  commit,
		log:     log,
	}
}

// SetApp sets the app reference (for circular dependency resolution)
func (u *UI) SetApp(application *app.App) {
	u.app = application
}

func (u *UI) Run(ctx context.Context) error {
	systray.Run(u.onReady, u.onExit)
	return nil
}

func (u *UI) onReady() {
	// Use emoji instead of icon - microphone with initial status
	u.updateStatus("idle")
	systray.SetTooltip("Local voice dictation")

	// Build menu
	u.mStartStop = systray.AddMenuItem("Start Dictation", "Press hotkey to dictate")
	systray.AddSeparator()

	u.mMode = systray.AddMenuItem("Mode: Push-to-Talk", "Toggle between modes")
	systray.AddSeparator()

	u.mDevices = systray.AddMenuItem("Microphone", "Select audio device")
	u.buildDeviceMenu()

	u.mModels = systray.AddMenuItem("Model", "Select Whisper model")
	u.buildModelMenu()

	systray.AddSeparator()
	u.mPastePrefer = systray.AddMenuItemCheckbox("Prefer Paste", "Use clipboard paste", u.cfg.Inject.PreferPaste)
	u.mRunAtLogin = systray.AddMenuItemCheckbox("Run at Login", "Start on system boot", u.cfg.RunAtLogin)

	systray.AddSeparator()
	mLogs := systray.AddMenuItem("Open Logs", "View application logs")
	mAbout := systray.AddMenuItem("About", "About WhisperTray")
	mQuit := systray.AddMenuItem("Quit", "Exit application")

	// Event loop
	go u.handleEvents(mLogs, mAbout, mQuit)
}

func (u *UI) handleEvents(mLogs, mAbout, mQuit *systray.MenuItem) {
	for {
		select {
		case <-u.mMode.ClickedCh:
			u.toggleMode()
		case <-u.mPastePrefer.ClickedCh:
			u.togglePastePrefer()
		case <-u.mRunAtLogin.ClickedCh:
			u.toggleRunAtLogin()
		case <-mLogs.ClickedCh:
			u.openLogs()
		case <-mAbout.ClickedCh:
			u.showAbout()
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func (u *UI) buildDeviceMenu() {
	// Get devices from app
	devices, err := u.app.ListDevices()
	if err != nil {
		u.log.Error().Err(err).Msg("Failed to list audio devices")
		return
	}

	deviceItems := make(map[string]*systray.MenuItem)

	for _, dev := range devices {
		item := u.mDevices.AddSubMenuItem(dev.Name, "")
		if dev.Default {
			item.Check()
		}
		deviceItems[dev.ID] = item

		go func(deviceID, deviceName string, menuItem *systray.MenuItem) {
			for {
				<-menuItem.ClickedCh
				// Uncheck all other items
				for id, itm := range deviceItems {
					if id != deviceID {
						itm.Uncheck()
					}
				}
				// Check this item
				menuItem.Check()
				u.cfg.Audio.DeviceID = deviceID
				u.cfg.Save()
				u.log.Info().Str("device", deviceName).Msg("Changed audio device")
				u.app.SetDevice(deviceID)
			}
		}(dev.ID, dev.Name, item)
	}
}

func (u *UI) buildModelMenu() {
	models := []string{"base.en", "small.en", "medium.en", "large-v3"}
	modelItems := make(map[string]*systray.MenuItem)

	for _, model := range models {
		item := u.mModels.AddSubMenuItem(model, "")
		if model == u.cfg.Whisper.Model {
			item.Check()
		}
		modelItems[model] = item

		go func(m string, menuItem *systray.MenuItem) {
			for {
				<-menuItem.ClickedCh
				// Uncheck all other items
				for mdl, itm := range modelItems {
					if mdl != m {
						itm.Uncheck()
					}
				}
				// Check this item
				menuItem.Check()
				oldModel := u.cfg.Whisper.Model
				u.cfg.Whisper.Model = m
				u.cfg.Save()
				u.log.Info().Str("from", oldModel).Str("to", m).Msg("Changed Whisper model")
				u.app.SetModel(m)
			}
		}(model, item)
	}
}

func (u *UI) toggleMode() {
	oldMode := u.cfg.Mode
	if u.cfg.Mode == "PushToTalk" {
		u.cfg.Mode = "Toggle"
		u.mMode.SetTitle("Mode: Toggle")
		u.app.SetMode("Toggle")
	} else {
		u.cfg.Mode = "PushToTalk"
		u.mMode.SetTitle("Mode: Push-to-Talk")
		u.app.SetMode("PushToTalk")
	}
	u.cfg.Save()
	u.log.Info().Str("from", oldMode).Str("to", u.cfg.Mode).Msg("Changed mode")
}

func (u *UI) togglePastePrefer() {
	u.cfg.Inject.PreferPaste = !u.cfg.Inject.PreferPaste
	if u.cfg.Inject.PreferPaste {
		u.mPastePrefer.Check()
		u.log.Info().Msg("Enabled prefer paste (Cmd+V)")
	} else {
		u.mPastePrefer.Uncheck()
		u.log.Info().Msg("Disabled prefer paste (using keyboard typing)")
	}
	u.cfg.Save()
}

func (u *UI) toggleRunAtLogin() {
	u.cfg.RunAtLogin = !u.cfg.RunAtLogin
	if u.cfg.RunAtLogin {
		u.mRunAtLogin.Check()
		u.log.Info().Msg("Enabled run at login")
	} else {
		u.mRunAtLogin.Uncheck()
		u.log.Info().Msg("Disabled run at login")
	}
	u.cfg.Save()
	// TODO: Platform-specific login item registration
}

func (u *UI) openLogs() {
	// TODO: Open log file with default app
	fmt.Println("Open logs...")
}

func (u *UI) showAbout() {
	// TODO: Show about dialog with native UI
	fmt.Printf("WhisperTray %s (%s)\nLocal voice dictation\n", u.version, u.commit)
}

func (u *UI) onExit() {
	// Cleanup
}

// updateStatus sets the tray title with microphone emoji and status indicator
func (u *UI) updateStatus(status string) {
	emoji := emojiForStatus(status)
	systray.SetTitle(fmt.Sprintf("🎤 %s", emoji))
}

// emojiForStatus returns the appropriate status emoji
func emojiForStatus(status string) string {
	switch status {
	case "recording":
		return "🔴" // Red - recording
	case "processing":
		return "🟡" // Yellow - processing transcription
	case "idle":
		return "🟢" // Green - ready/idle
	case "error":
		return "⚪️" // White - error
	default:
		return "🟢" // Green - default to ready
	}
}