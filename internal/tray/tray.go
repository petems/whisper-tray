package tray

import (
	"context"
	"fmt"

	"github.com/petems/whisper-tray/internal/app"
	"github.com/petems/whisper-tray/internal/config"
	"github.com/getlantern/systray"
)

type UI struct {
	app     *app.App
	cfg     *config.Config
	version string
	commit  string

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
	return &UI{
		app:     application,
		cfg:     cfg,
		version: version,
		commit:  commit,
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
		return
	}

	for _, dev := range devices {
		item := u.mDevices.AddSubMenuItem(dev.Name, "")
		if dev.Default {
			item.Check()
		}

		go func(deviceID string, menuItem *systray.MenuItem) {
			for {
				<-menuItem.ClickedCh
				u.app.SetDevice(deviceID)
			}
		}(dev.ID, item)
	}
}

func (u *UI) buildModelMenu() {
	models := []string{"base.en", "small.en", "medium.en", "large-v3"}

	for _, model := range models {
		item := u.mModels.AddSubMenuItem(model, "")
		if model == u.cfg.Whisper.Model {
			item.Check()
		}

		go func(m string, menuItem *systray.MenuItem) {
			for {
				<-menuItem.ClickedCh
				u.app.SetModel(m)
			}
		}(model, item)
	}
}

func (u *UI) toggleMode() {
	if u.cfg.Mode == "PushToTalk" {
		u.cfg.Mode = "Toggle"
		u.mMode.SetTitle("Mode: Toggle")
		u.app.SetMode("Toggle")
	} else {
		u.cfg.Mode = "PushToTalk"
		u.mMode.SetTitle("Mode: Push-to-Talk")
		u.app.SetMode("PushToTalk")
	}
}

func (u *UI) togglePastePrefer() {
	u.cfg.Inject.PreferPaste = !u.cfg.Inject.PreferPaste
	if u.cfg.Inject.PreferPaste {
		u.mPastePrefer.Check()
	} else {
		u.mPastePrefer.Uncheck()
	}
	u.cfg.Save()
}

func (u *UI) toggleRunAtLogin() {
	u.cfg.RunAtLogin = !u.cfg.RunAtLogin
	if u.cfg.RunAtLogin {
		u.mRunAtLogin.Check()
	} else {
		u.mRunAtLogin.Uncheck()
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